package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"mime"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"github.com/maidulcu/masaar-crm/internal/mediastore"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/whatsapp"
)

// ErrNoMedia means the message has no downloadable file.
var ErrNoMedia = errors.New("message has no media")

// WAMediaService downloads the files customers send over WhatsApp and keeps them in the media
// store. Meta only keeps a file for about 30 days and gives out short-lived download URLs, so
// files are fetched in the background as soon as the message arrives (Prefetch) and, if that
// failed or the file is missing, on demand when someone opens the message (Ensure).
type WAMediaService struct {
	sender   *whatsapp.Sender
	store    *mediastore.Store
	msgs     *repo.WhatsAppRepo
	maxBytes int64

	group singleflight.Group // one download per message at a time
	sem   chan struct{}      // bounds concurrent background downloads
}

// NewWAMediaService returns a service; store may be nil, which disables media (Enabled is false).
func NewWAMediaService(sender *whatsapp.Sender, store *mediastore.Store, msgs *repo.WhatsAppRepo, maxMB int) *WAMediaService {
	if maxMB < 1 {
		maxMB = 25
	}
	return &WAMediaService{sender: sender, store: store, msgs: msgs, maxBytes: int64(maxMB) << 20, sem: make(chan struct{}, 4)}
}

// Enabled reports whether media can be stored and fetched.
func (m *WAMediaService) Enabled() bool {
	return m != nil && m.store != nil
}

// Store exposes the underlying file store (for saving outgoing files).
func (m *WAMediaService) Store() *mediastore.Store { return m.store }

// Prefetch downloads a message's file in the background. It is best effort: on failure the file
// is fetched on demand later.
func (m *WAMediaService) Prefetch(companyID, threadID, messageID uuid.UUID) {
	if !m.Enabled() || !m.sender.IsConfigured() {
		return
	}
	go func() {
		m.sem <- struct{}{}
		defer func() { <-m.sem }()
		ctx, cancel := context.WithTimeout(tenant.With(context.Background(), companyID), 2*time.Minute)
		defer cancel()
		if _, err := m.Ensure(ctx, threadID, messageID); err != nil {
			log.Printf("whatsapp: media prefetch for message %s: %v", messageID, err)
		}
	}()
}

// Ensure returns the stored file for a message, downloading it from Meta first if necessary.
func (m *WAMediaService) Ensure(ctx context.Context, threadID, messageID uuid.UUID) (*repo.MessageMedia, error) {
	if !m.Enabled() {
		return nil, ErrNoMedia
	}
	ref, err := m.msgs.GetMessageMedia(ctx, threadID, messageID)
	if err != nil {
		return nil, err
	}
	if ref.Path != "" {
		if f, err := m.store.Open(ref.Path); err == nil {
			f.Close()
			return ref, nil
		}
		// The record points at a file that is gone (volume lost): forget it and refetch.
		if err := m.msgs.ClearMessageMedia(ctx, messageID); err != nil {
			return nil, err
		}
		ref.Path = ""
	}
	if ref.WAMediaID == "" {
		return nil, ErrNoMedia
	}
	if !m.sender.IsConfigured() {
		return nil, ErrNoMedia
	}

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	// The download must not be cancelled just because the first requester went away: others may
	// be waiting on the same result.
	_, err, _ = m.group.Do(messageID.String(), func() (any, error) {
		dctx, cancel := context.WithTimeout(tenant.With(context.Background(), cid), 2*time.Minute)
		defer cancel()
		return nil, m.download(dctx, threadID, ref)
	})
	if err != nil {
		return nil, err
	}
	return m.msgs.GetMessageMedia(ctx, threadID, messageID)
}

func (m *WAMediaService) download(ctx context.Context, threadID uuid.UUID, ref *repo.MessageMedia) error {
	stream, err := m.sender.FetchMedia(ctx, ref.WAMediaID, m.maxBytes)
	if err != nil {
		return err
	}
	defer stream.Body.Close()

	name, size, err := m.store.Save(stream.Body, m.maxBytes)
	if errors.Is(err, mediastore.ErrTooLarge) {
		return whatsapp.ErrMediaTooLarge
	}
	if err != nil {
		return err
	}
	set, err := m.msgs.SetMessageMedia(ctx, ref.MessageID, name, size, stream.Mime)
	if err != nil || !set {
		_ = m.store.Remove(name) // lost a race with another download, or the update failed
		return err
	}
	return nil
}

// inlineMedia lists the types that are safe to display directly in the browser. Anything else a
// customer sends (HTML, SVG, scripts, …) is only ever offered as a download.
var inlineMedia = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
	"audio/ogg": true, "audio/mpeg": true, "audio/mp4": true, "audio/aac": true, "audio/amr": true,
	"video/mp4": true, "video/3gpp": true,
	"application/pdf": true,
}

// servedType returns the Content-Type to answer with and whether the file may be shown inline.
// The type is the customer's claim, so only known-safe types are passed through.
func servedType(claimed string) (contentType string, inline bool) {
	mt, _, err := mime.ParseMediaType(claimed)
	if err != nil {
		return "application/octet-stream", false
	}
	mt = strings.ToLower(mt)
	if inlineMedia[mt] {
		return mt, true
	}
	return "application/octet-stream", false
}

// GetMedia godoc
// @Summary      Download a WhatsApp message attachment
// @Description  Streams the image/document/audio/video attached to a message. The file is downloaded from WhatsApp on first access if it has not been stored yet.
// @Tags         WhatsApp
// @Param        id   path  string  true  "Thread UUID"
// @Param        mid  path  string  true  "Message UUID"
// @Success      200
// @Failure      404  {object}  object{error=string}
// @Failure      410  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /threads/{id}/messages/{mid}/media [get]
func (h *WhatsAppHandler) GetMedia(c *fiber.Ctx) error {
	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	msgID, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	ref, err := h.media.Ensure(c.Context(), threadID, msgID)
	switch {
	case err == nil:
	case errors.Is(err, ErrNoMedia), errors.Is(err, repo.ErrForeignReference):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no attachment"})
	case errors.Is(err, whatsapp.ErrMediaGone):
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "attachment is no longer available from WhatsApp"})
	case errors.Is(err, whatsapp.ErrMediaTooLarge):
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "attachment is too large"})
	default:
		status, msg := apiError(err)
		if status == fiber.StatusInternalServerError {
			log.Printf("whatsapp: media %s: %v", msgID, err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "could not fetch the attachment"})
		}
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	f, err := h.media.Store().Open(ref.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no attachment"})
		}
		return serverError(c, err)
	}

	contentType, inline := servedType(ref.Mime)
	c.Set(fiber.HeaderContentType, contentType)
	// Customer-supplied bytes: never let a browser sniff them into something executable, and
	// forbid scripts/frames even if one is navigated to directly.
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set(fiber.HeaderCacheControl, "private, max-age=3600")
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	c.Set(fiber.HeaderContentDisposition, mime.FormatMediaType(disposition, map[string]string{"filename": safeFilename(ref.Filename)}))
	return c.SendStream(f, int(ref.Size))
}

// safeFilename reduces a customer-supplied file name to something harmless in a header.
func safeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, r == 0x7f, r == '/', r == '\\', r == '"', r == ';':
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "attachment"
	}
	if len([]rune(name)) > 120 {
		name = string([]rune(name)[:120])
	}
	return name
}

// readAllLimited reads r fully but fails with ErrTooLarge past limit.
func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, mediastore.ErrTooLarge
	}
	return b, nil
}
