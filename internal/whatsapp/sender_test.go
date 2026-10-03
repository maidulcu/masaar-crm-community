package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const okReply = `{"messages":[{"id":"wamid.X"}]}`

// fakeGraph starts a fake Graph API that records the last JSON body sent to it.
func fakeGraph(t *testing.T, status int, reply string) (*Sender, *map[string]any) {
	t.Helper()
	got := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return NewSender(&SenderConfig{BaseURL: srv.URL, PhoneNumberID: "PNID", AccessToken: "tok", AllowInsecureMedia: true}), &got
}

// Meta rejects a body component whose "parameters" is null; templates without variables are
// common, so the component must be omitted.
func TestSendTemplate_WithoutParametersOmitsBodyComponent(t *testing.T) {
	s, got := fakeGraph(t, 200, okReply)
	if _, err := s.SendTemplate(context.Background(), "9715", "welcome", "ar", nil); err != nil {
		t.Fatal(err)
	}
	tpl := (*got)["template"].(map[string]any)
	if _, present := tpl["components"]; present {
		t.Errorf("template with no variables sent components: %v", tpl["components"])
	}
	s2, got2 := fakeGraph(t, 200, okReply)
	_, _ = s2.SendTemplate(context.Background(), "9715", "hello", "en", []string{"Sara", "Villa 4"})
	comps := (*got2)["template"].(map[string]any)["components"].([]any)
	params := comps[0].(map[string]any)["parameters"].([]any)
	if len(params) != 2 || params[0].(map[string]any)["text"] != "Sara" {
		t.Errorf("parameters = %v", params)
	}
}

func TestSendMedia_Payloads(t *testing.T) {
	ctx := context.Background()

	s, got := fakeGraph(t, 200, okReply)
	if _, err := s.SendMedia(ctx, "9715", "image", MediaRef{Link: "https://example.com/a.jpg"}, "Sea view", ""); err != nil {
		t.Fatal(err)
	}
	img := (*got)["image"].(map[string]any)
	if img["link"] != "https://example.com/a.jpg" || img["caption"] != "Sea view" || (*got)["type"] != "image" {
		t.Errorf("image payload = %v", *got)
	}

	s, got = fakeGraph(t, 200, okReply)
	_, _ = s.SendMedia(ctx, "9715", "document", MediaRef{ID: "MEDIA1"}, "Brochure", "brochure.pdf")
	doc := (*got)["document"].(map[string]any)
	if doc["id"] != "MEDIA1" || doc["filename"] != "brochure.pdf" || doc["caption"] != "Brochure" || doc["link"] != nil {
		t.Errorf("document payload = %v", doc)
	}

	// Audio messages cannot carry a caption.
	s, got = fakeGraph(t, 200, okReply)
	_, _ = s.SendMedia(ctx, "9715", "audio", MediaRef{ID: "M2"}, "ignored", "")
	if _, has := (*got)["audio"].(map[string]any)["caption"]; has {
		t.Error("caption sent with an audio message")
	}
}

// media_type is client-controlled and used as a JSON key, so only real media types are allowed.
func TestSendMedia_RejectsBadArguments(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&calls, 1) }))
	defer srv.Close()
	s := NewSender(&SenderConfig{BaseURL: srv.URL, PhoneNumberID: "P", AccessToken: "t"})

	for _, bad := range []string{"template", "messaging_product", "to", "text", "sticker", ""} {
		if _, err := s.SendMedia(context.Background(), "9715", bad, MediaRef{Link: "https://e.com/a"}, "", ""); !errors.Is(err, ErrInvalidMedia) {
			t.Errorf("media_type %q: err = %v, want ErrInvalidMedia", bad, err)
		}
	}
	for _, ref := range []MediaRef{{}, {ID: "a", Link: "https://e.com/a"}} {
		if _, err := s.SendMedia(context.Background(), "9715", "image", ref, "", ""); !errors.Is(err, ErrInvalidMedia) {
			t.Errorf("ref %+v: err = %v, want ErrInvalidMedia", ref, err)
		}
	}
	if calls != 0 {
		t.Errorf("%d invalid requests reached Meta", calls)
	}
}

// Callers need Meta's error code to tell "outside the 24h window" from an outage.
func TestSend_ExposesMetaErrorCode(t *testing.T) {
	s, _ := fakeGraph(t, 400, `{"error":{"message":"Re-engagement message","type":"OAuthException","code":131047}}`)
	_, err := s.SendMessage(context.Background(), "9715", "hi")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 131047 || apiErr.HTTPStatus != 400 {
		t.Fatalf("error = %#v, want APIError{Code:131047, HTTPStatus:400}", err)
	}
	if !strings.Contains(err.Error(), "131047") {
		t.Errorf("message does not mention the code: %v", err)
	}
}
