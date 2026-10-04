package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/safehttp"
)

var (
	// ErrMediaGone means Meta no longer has the file (media ids expire after about 30 days).
	ErrMediaGone = errors.New("whatsapp media no longer available")
	// ErrMediaTooLarge means the file exceeds the configured size limit.
	ErrMediaTooLarge = errors.New("whatsapp media too large")
	// ErrInvalidMedia is returned for unusable media arguments (bad id, type, host).
	ErrInvalidMedia = errors.New("invalid whatsapp media")
)

// defaultMediaHosts are the domains Meta serves downloads from. The access token is sent to the
// download URL, so it is only ever sent to these.
var defaultMediaHosts = []string{"fbsbx.com", "facebook.com", "fbcdn.net", "whatsapp.net", "whatsapp.com"}

var mediaIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

// MediaTypes are the message types that carry a file.
var MediaTypes = map[string]bool{"image": true, "document": true, "audio": true, "video": true}

// ValidMediaType reports whether t is a media message type WhatsApp accepts.
func ValidMediaType(t string) bool { return MediaTypes[t] }

// SupportsCaption reports whether a media message of this type can carry a caption.
func SupportsCaption(t string) bool { return t == "image" || t == "video" || t == "document" }

// MediaRef points at a file for an outgoing media message: either an id from UploadMedia or a
// public https link Meta fetches itself.
type MediaRef struct {
	ID   string
	Link string
}

// SendMedia sends an image, document, audio or video message.
func (s *Sender) SendMedia(ctx context.Context, to, mediaType string, ref MediaRef, caption, filename string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("WhatsApp sender not configured")
	}
	// mediaType becomes a JSON key below, so it must be one of the known types.
	if !ValidMediaType(mediaType) {
		return "", fmt.Errorf("%w: media type %q", ErrInvalidMedia, mediaType)
	}
	if (ref.ID == "") == (ref.Link == "") {
		return "", fmt.Errorf("%w: exactly one of id or link is required", ErrInvalidMedia)
	}
	obj := map[string]string{}
	if ref.ID != "" {
		obj["id"] = ref.ID
	} else {
		obj["link"] = ref.Link
	}
	if caption != "" && SupportsCaption(mediaType) {
		obj["caption"] = caption
	}
	if filename != "" && mediaType == "document" {
		obj["filename"] = filename
	}
	return s.sendRequest(ctx, map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              mediaType,
		mediaType:           obj,
	})
}

// UploadMedia uploads a file to WhatsApp and returns its media id, for use with SendMedia.
func (s *Sender) UploadMedia(ctx context.Context, filename, mimeType string, data []byte) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("WhatsApp sender not configured")
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("messaging_product", "whatsapp")
	_ = w.WriteField("type", mimeType)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, strings.NewReplacer("\"", "", "\r", "", "\n", "").Replace(filename)))
	h.Set("Content-Type", mimeType)
	part, err := w.CreatePart(h)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/media", s.config.BaseURL, s.config.PhoneNumberID), &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", parseAPIError(resp)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.ID == "" {
		return "", fmt.Errorf("no media id in upload response")
	}
	return out.ID, nil
}

// MediaStream is a downloaded file. The caller must close Body.
type MediaStream struct {
	Body io.ReadCloser
	Mime string
	Size int64 // -1 when unknown
}

// FetchMedia downloads the file behind a media id received in a webhook. maxBytes bounds it:
// a larger file fails with ErrMediaTooLarge before (or while) it is read.
//
// Meta returns a short-lived URL for the id; downloading needs the access token, which is only
// ever sent to Meta's own hosts, and the connection itself refuses internal addresses.
func (s *Sender) FetchMedia(ctx context.Context, mediaID string, maxBytes int64) (*MediaStream, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("WhatsApp sender not configured")
	}
	if !mediaIDRe.MatchString(mediaID) {
		return nil, fmt.Errorf("%w: media id", ErrInvalidMedia)
	}

	// 1. Resolve the id to a download URL.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", s.config.BaseURL, mediaID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to look up media: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		apiErr := parseAPIError(resp)
		// An unknown or expired id: 404, or 400 with code 100 ("object does not exist"). A 5xx
		// is a temporary Meta problem, not a missing file, whatever its body says.
		if apiErr.HTTPStatus == http.StatusNotFound || (apiErr.HTTPStatus == http.StatusBadRequest && apiErr.Code == 100) {
			return nil, fmt.Errorf("%w: %v", ErrMediaGone, apiErr)
		}
		return nil, apiErr
	}
	var info struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
		FileSize int64  `json:"file_size"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&info)
	resp.Body.Close()
	if err != nil || info.URL == "" {
		return nil, fmt.Errorf("unexpected media lookup response")
	}
	if info.FileSize > maxBytes {
		return nil, ErrMediaTooLarge
	}
	if err := s.checkMediaURL(info.URL); err != nil {
		return nil, err
	}

	// 2. Download it.
	dreq, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return nil, err
	}
	dreq.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	dresp, err := s.mediaClient.Do(dreq)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	if dresp.StatusCode != http.StatusOK {
		dresp.Body.Close()
		if dresp.StatusCode == http.StatusNotFound || dresp.StatusCode == http.StatusGone {
			return nil, ErrMediaGone
		}
		return nil, fmt.Errorf("media download failed: HTTP %d", dresp.StatusCode)
	}
	if dresp.ContentLength > maxBytes {
		dresp.Body.Close()
		return nil, ErrMediaTooLarge
	}
	mimeType := info.MimeType
	if mimeType == "" {
		mimeType = dresp.Header.Get("Content-Type")
	}
	return &MediaStream{Body: dresp.Body, Mime: mimeType, Size: dresp.ContentLength}, nil
}

// checkMediaURL ensures a download URL points at one of Meta's hosts over https.
func (s *Sender) checkMediaURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: media url", ErrInvalidMedia)
	}
	if s.config.AllowInsecureMedia {
		return nil
	}
	if u.Scheme != "https" || u.User != nil {
		return fmt.Errorf("%w: media url must be https", ErrInvalidMedia)
	}
	if !hostAllowed(u.Hostname(), s.mediaHosts()) {
		return fmt.Errorf("%w: media host %q is not a WhatsApp host", ErrInvalidMedia, u.Hostname())
	}
	return nil
}

func (s *Sender) mediaHosts() []string {
	if len(s.config.MediaHosts) > 0 {
		return s.config.MediaHosts
	}
	return defaultMediaHosts
}

// hostAllowed reports whether host equals, or is a subdomain of, one of the suffixes.
func hostAllowed(host string, suffixes []string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, s := range suffixes {
		s = strings.ToLower(s)
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}
	return false
}

// newMediaClient returns the client used for downloads. It refuses internal destinations (SSRF)
// and re-checks the host on every redirect so the token cannot be sent elsewhere.
func newMediaClient(cfg *SenderConfig) *http.Client {
	if cfg != nil && cfg.AllowInsecureMedia {
		return &http.Client{Timeout: 120 * time.Second}
	}
	hosts := defaultMediaHosts
	if cfg != nil && len(cfg.MediaHosts) > 0 {
		hosts = cfg.MediaHosts
	}
	c := safehttp.NewClient(120 * time.Second)
	prev := c.CheckRedirect
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" || !hostAllowed(req.URL.Hostname(), hosts) {
			return fmt.Errorf("%w: redirect to %q", ErrInvalidMedia, req.URL.Host)
		}
		if prev != nil {
			return prev(req, via)
		}
		return nil
	}
	return c
}
