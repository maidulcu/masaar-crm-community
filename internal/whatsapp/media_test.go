package whatsapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// metaMedia fakes the two-step media download: GET /{id} returns metadata with a download URL,
// GET /dl/{id} returns the bytes. It records the Authorization header of every request.
type metaMedia struct {
	srv       *httptest.Server
	mu        sync.Mutex
	auth      map[string]string
	lookupURL func(srvURL string) string // overrides the download URL returned by the lookup
	fileSize  int64
	body      []byte
	lookupErr int // HTTP status for the lookup, 0 = ok
}

func newMetaMedia(t *testing.T, body []byte) *metaMedia {
	m := &metaMedia{auth: map[string]string{}, body: body, fileSize: int64(len(body))}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.auth[r.URL.Path] = r.Header.Get("Authorization")
		m.mu.Unlock()
		switch {
		case strings.HasPrefix(r.URL.Path, "/dl/"):
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(m.body)
		case m.lookupErr != 0:
			w.WriteHeader(m.lookupErr)
			_, _ = w.Write([]byte(`{"error":{"message":"Unsupported get request","code":100}}`))
		default:
			u := m.srv.URL + "/dl" + r.URL.Path
			if m.lookupURL != nil {
				u = m.lookupURL(m.srv.URL)
			}
			_, _ = fmt.Fprintf(w, `{"url":%q,"mime_type":"image/jpeg","file_size":%d}`, u, m.fileSize)
		}
	}))
	t.Cleanup(m.srv.Close)
	return m
}

func (m *metaMedia) sender(insecure bool) *Sender {
	return NewSender(&SenderConfig{BaseURL: m.srv.URL, PhoneNumberID: "PN", AccessToken: "SECRET-TOKEN", AllowInsecureMedia: insecure})
}

func TestFetchMedia_Downloads(t *testing.T) {
	m := newMetaMedia(t, []byte("JPEGDATA"))
	st, err := m.sender(true).FetchMedia(context.Background(), "MEDIA1", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Body.Close()
	if b, _ := io.ReadAll(st.Body); string(b) != "JPEGDATA" || st.Mime != "image/jpeg" {
		t.Errorf("got %q (%s)", b, st.Mime)
	}
	if m.auth["/MEDIA1"] != "Bearer SECRET-TOKEN" || m.auth["/dl/MEDIA1"] != "Bearer SECRET-TOKEN" {
		t.Errorf("token not sent to both steps: %v", m.auth)
	}
}

// The access token must only ever be sent to Meta's own hosts: Meta's lookup response is data,
// not something to trust blindly.
func TestFetchMedia_RefusesNonMetaHosts(t *testing.T) {
	for _, evil := range []string{
		"https://evil.example/steal",
		"http://lookaside.fbsbx.com/x",               // not https
		"https://lookaside.fbsbx.com.evil.example/x", // suffix trick
		"https://evilfbsbx.com/x",                    // missing dot boundary
		"https://lookaside.fbsbx.com@evil.example/x", // userinfo trick: the host is evil.example
		"https://169.254.169.254/latest/meta-data/",  // cloud metadata
		"https://localhost:8080/internal",            // internal service
		"file:///etc/passwd",
	} {
		t.Run(evil, func(t *testing.T) {
			m := newMetaMedia(t, []byte("x"))
			m.lookupURL = func(string) string { return evil }
			_, err := m.sender(false).FetchMedia(context.Background(), "MEDIA1", 1<<20)
			if !errors.Is(err, ErrInvalidMedia) {
				t.Fatalf("err = %v, want ErrInvalidMedia", err)
			}
			if _, downloaded := m.auth["/steal"]; downloaded || len(m.auth) != 1 {
				t.Errorf("a download was attempted: %v", m.auth)
			}
		})
	}
}

func TestFetchMedia_AcceptsMetaSubdomains(t *testing.T) {
	s := NewSender(&SenderConfig{BaseURL: "http://unused", PhoneNumberID: "P", AccessToken: "t"})
	for _, ok := range []string{"https://lookaside.fbsbx.com/whatsapp_business/attachments/?mid=1", "https://scontent.xx.fbcdn.net/v/t.jpg", "https://mmg.whatsapp.net/d/f/x"} {
		if err := s.checkMediaURL(ok); err != nil {
			t.Errorf("%s rejected: %v", ok, err)
		}
	}
}

func TestFetchMedia_RejectsBadIDsWithoutRequest(t *testing.T) {
	m := newMetaMedia(t, []byte("x"))
	for _, id := range []string{"", "../etc/passwd", "a/b", "a?b=c", "a b", strings.Repeat("a", 101)} {
		if _, err := m.sender(true).FetchMedia(context.Background(), id, 1<<20); !errors.Is(err, ErrInvalidMedia) {
			t.Errorf("id %q: err = %v", id, err)
		}
	}
	if len(m.auth) != 0 {
		t.Errorf("requests were made for invalid ids: %v", m.auth)
	}
}

func TestFetchMedia_SizeLimitAndGone(t *testing.T) {
	m := newMetaMedia(t, bytes.Repeat([]byte("a"), 2000))
	if _, err := m.sender(true).FetchMedia(context.Background(), "M", 1000); !errors.Is(err, ErrMediaTooLarge) {
		t.Errorf("declared size over limit: err = %v, want ErrMediaTooLarge", err)
	}
	if _, downloaded := m.auth["/dl/M"]; downloaded {
		t.Error("oversized file was downloaded")
	}
	// Meta's metadata understates the size, but the download itself reports its real length.
	m.fileSize = 10
	if _, err := m.sender(true).FetchMedia(context.Background(), "M", 1000); !errors.Is(err, ErrMediaTooLarge) {
		t.Errorf("understated metadata: err = %v, want ErrMediaTooLarge from Content-Length", err)
	}

	// A temporary Meta failure is not "gone", even though the body carries error code 100.
	m3 := newMetaMedia(t, nil)
	m3.lookupErr = 503
	if _, err := m3.sender(true).FetchMedia(context.Background(), "M", 1000); err == nil || errors.Is(err, ErrMediaGone) {
		t.Errorf("503 from Meta: err = %v, want a retryable error that is not ErrMediaGone", err)
	}

	m2 := newMetaMedia(t, nil)
	m2.lookupErr = 404
	if _, err := m2.sender(true).FetchMedia(context.Background(), "M", 1000); !errors.Is(err, ErrMediaGone) {
		t.Errorf("expired media: err = %v, want ErrMediaGone", err)
	}
}

func TestUploadMedia(t *testing.T) {
	var gotType, gotField, gotFile, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
		}
		gotField = r.FormValue("messaging_product")
		gotType = r.FormValue("type")
		f, hdr, _ := r.FormFile("file")
		b, _ := io.ReadAll(f)
		gotFile = hdr.Filename + ":" + string(b)
		_, _ = w.Write([]byte(`{"id":"UPLOADED1"}`))
	}))
	defer srv.Close()
	s := NewSender(&SenderConfig{BaseURL: srv.URL, PhoneNumberID: "PN", AccessToken: "tok"})
	id, err := s.UploadMedia(context.Background(), "brochure.pdf", "application/pdf", []byte("%PDF-1.4"))
	if err != nil || id != "UPLOADED1" {
		t.Fatalf("UploadMedia = %q, %v", id, err)
	}
	if gotField != "whatsapp" || gotType != "application/pdf" || gotFile != "brochure.pdf:%PDF-1.4" || auth != "Bearer tok" {
		t.Errorf("upload request: product=%q type=%q file=%q auth=%q", gotField, gotType, gotFile, auth)
	}
}
