package handler

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/internal/whatsapp"
)

// outgoingMedia is a validated request to send a file.
type outgoingMedia struct {
	Type     string // image | document | audio | video
	Caption  string
	Filename string
	Link     string // link mode: public https URL Meta fetches
	Data     []byte // upload mode: the file
	Mime     string
}

// maxMediaCaption is WhatsApp's caption limit.
const maxMediaCaption = 1024

// uploadMimes lists, per message type, the file types WhatsApp accepts.
var uploadMimes = map[string]map[string]bool{
	"image":    {"image/jpeg": true, "image/png": true},
	"audio":    {"audio/aac": true, "audio/mp4": true, "audio/mpeg": true, "audio/amr": true, "audio/ogg": true},
	"video":    {"video/mp4": true, "video/3gpp": true},
	"document": {"application/pdf": true, "text/plain": true, "application/msword": true, "application/vnd.ms-excel": true, "application/vnd.ms-powerpoint": true, "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true, "application/vnd.openxmlformats-officedocument.presentationml.presentation": true},
}

// uploadLimits are WhatsApp's per-type size limits in bytes (documents are further capped by the
// API's request size limit).
var uploadLimits = map[string]int64{"image": 5 << 20, "audio": 16 << 20, "video": 16 << 20, "document": 9 << 20}

func readLinkedMedia(c *fiber.Ctx) (outgoingMedia, error) {
	var req SendMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return outgoingMedia{}, errors.New("invalid request")
	}
	if req.MediaURL == "" || req.MediaType == "" {
		return outgoingMedia{}, errors.New("media_url and media_type are required")
	}
	in := outgoingMedia{Type: req.MediaType, Caption: strings.TrimSpace(req.Caption), Link: req.MediaURL}
	if err := validateMediaType(in); err != nil {
		return outgoingMedia{}, err
	}
	// Meta fetches the link itself, and only public https links work.
	u, err := url.Parse(req.MediaURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return outgoingMedia{}, errors.New("media_url must be a public https URL")
	}
	if in.Type == "document" {
		in.Filename = safeFilename(firstNonEmpty(req.Filename, filepath.Base(u.Path)))
	}
	return in, nil
}

func readUploadedMedia(c *fiber.Ctx) (outgoingMedia, error) {
	fh, err := c.FormFile("file")
	if err != nil {
		return outgoingMedia{}, errors.New("file is required")
	}
	declared := strings.ToLower(strings.TrimSpace(fh.Header.Get(fiber.HeaderContentType)))
	if mt, _, err := mime.ParseMediaType(declared); err == nil {
		declared = mt
	}
	if declared == "" || declared == "application/octet-stream" {
		declared = strings.ToLower(mime.TypeByExtension(filepath.Ext(fh.Filename)))
		if mt, _, err := mime.ParseMediaType(declared); err == nil {
			declared = mt
		}
	}

	mediaType := c.FormValue("media_type")
	if mediaType == "" {
		mediaType = typeForMime(declared)
	}
	in := outgoingMedia{Type: mediaType, Caption: strings.TrimSpace(c.FormValue("caption")), Mime: declared}
	if err := validateMediaType(in); err != nil {
		return outgoingMedia{}, err
	}
	if !uploadMimes[in.Type][declared] {
		return outgoingMedia{}, fmt.Errorf("WhatsApp does not accept %q files as %s", declared, in.Type)
	}
	if limit := uploadLimits[in.Type]; fh.Size > limit {
		return outgoingMedia{}, fmt.Errorf("file is too large (limit %d MB for %s)", limit>>20, in.Type)
	}
	if fh.Size == 0 {
		return outgoingMedia{}, errors.New("file is empty")
	}

	f, err := fh.Open()
	if err != nil {
		return outgoingMedia{}, errors.New("could not read file")
	}
	defer f.Close()
	data, err := readAllLimited(f, uploadLimits[in.Type])
	if err != nil {
		return outgoingMedia{}, errors.New("could not read file")
	}
	if err := checkContent(declared, data); err != nil {
		return outgoingMedia{}, err
	}
	in.Data = data
	in.Filename = safeFilename(filepath.Base(fh.Filename))
	return in, nil
}

func validateMediaType(in outgoingMedia) error {
	if !whatsapp.ValidMediaType(in.Type) {
		return errors.New("media_type must be one of image, document, audio, video")
	}
	if in.Caption != "" {
		if !whatsapp.SupportsCaption(in.Type) {
			return fmt.Errorf("%s messages cannot have a caption", in.Type)
		}
		if utf8.RuneCountInString(in.Caption) > maxMediaCaption {
			return fmt.Errorf("caption must be at most %d characters", maxMediaCaption)
		}
	}
	return nil
}

// typeForMime picks the WhatsApp message type for a file type.
func typeForMime(m string) string {
	switch {
	case strings.HasPrefix(m, "image/"):
		return "image"
	case strings.HasPrefix(m, "audio/"):
		return "audio"
	case strings.HasPrefix(m, "video/"):
		return "video"
	}
	return "document"
}

// checkContent makes sure the bytes match the type the client claimed for formats we can verify,
// so a script renamed to .png is not forwarded to customers as an "image".
func checkContent(declared string, data []byte) error {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	switch declared {
	case "image/jpeg", "image/png":
		if got := http.DetectContentType(head); got != declared {
			return fmt.Errorf("file content is not a valid %s", declared)
		}
	case "application/pdf":
		if !strings.HasPrefix(string(head), "%PDF-") {
			return errors.New("file content is not a valid PDF")
		}
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":
		if !strings.HasPrefix(string(head), "PK\x03\x04") {
			return errors.New("file content is not a valid Office document")
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" && v != "." && v != "/" {
			return v
		}
	}
	return ""
}
