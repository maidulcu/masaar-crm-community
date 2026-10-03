package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// pngBytes is a minimal file that net/http recognises as image/png.
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

func imageMsg(from, id, mediaID, mime string) string {
	return fmt.Sprintf(`{"from":%q,"id":%q,"timestamp":"1","type":"image","image":{"id":%q,"mime_type":%q,"caption":"photo"}}`, from, id, mediaID, mime)
}

// messageID returns the id of the stored message with a Meta message id.
func (e *waEnv) messageID(t *testing.T, waID string) (thread, msg uuid.UUID) {
	t.Helper()
	if err := e.pool.QueryRow(context.Background(), `SELECT thread_id, id FROM whatsapp_messages WHERE wa_message_id=$1`, waID).Scan(&thread, &msg); err != nil {
		t.Fatalf("message %s: %v", waID, err)
	}
	return
}

func (e *waEnv) waitStored(t *testing.T, msg uuid.UUID) bool {
	t.Helper()
	for i := 0; i < 60; i++ {
		var p *string
		_ = e.pool.QueryRow(context.Background(), `SELECT media_path FROM whatsapp_messages WHERE id=$1`, msg).Scan(&p)
		if p != nil {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func (e *waEnv) getMedia(t *testing.T, prefix string, thread, msg uuid.UUID) (int, []byte, map[string]string) {
	t.Helper()
	resp, err := e.app.Test(httptest.NewRequest("GET", fmt.Sprintf("%s/threads/%s/messages/%s/media", prefix, thread, msg), nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	h := map[string]string{}
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	return resp.StatusCode, body, h
}

func (e *waEnv) storedFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(e.mediaDir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// A customer's photo is downloaded from Meta in the background and served from our own storage.
func TestWAMedia_InboundImageIsDownloadedAndServed(t *testing.T) {
	e := newWAEnv(t)
	e.meta.addFile("MEDIA1", "image/png", pngBytes)
	e.post(t, payload("971501110020", "Dana", imageMsg("971501110020", "wamid.IMG1", "MEDIA1", "image/png")))
	thread, msg := e.messageID(t, "wamid.IMG1")
	if !e.waitStored(t, msg) {
		t.Fatal("image was not downloaded in the background")
	}

	code, body, h := e.getMedia(t, "/api", thread, msg)
	if code != 200 || !bytes.Equal(body, pngBytes) {
		t.Fatalf("GET media = %d, %d bytes", code, len(body))
	}
	if h["Content-Type"] != "image/png" || !strings.HasPrefix(h["Content-Disposition"], "inline") {
		t.Errorf("content-type %q disposition %q", h["Content-Type"], h["Content-Disposition"])
	}
	if h["X-Content-Type-Options"] != "nosniff" || !strings.Contains(h["Content-Security-Policy"], "sandbox") {
		t.Errorf("customer file served without hardening headers: %v", h)
	}
	// Served from disk: no further download from Meta.
	_, before := e.meta.counts()
	e.getMedia(t, "/api", thread, msg)
	if _, after := e.meta.counts(); after != before {
		t.Error("stored file was downloaded again")
	}
}

// If the background download failed (Meta hiccup, restart), opening the message fetches it.
func TestWAMedia_FetchedOnDemandAfterFailedPrefetch(t *testing.T) {
	e := newWAEnv(t)
	e.meta.addFile("MEDIA2", "image/png", pngBytes)
	e.meta.setLookupFail(503)
	e.post(t, payload("971501110021", "Dana", imageMsg("971501110021", "wamid.IMG2", "MEDIA2", "image/png")))
	thread, msg := e.messageID(t, "wamid.IMG2")
	time.Sleep(300 * time.Millisecond) // let the (failing) prefetch finish

	if code, _, _ := e.getMedia(t, "/api", thread, msg); code != 502 {
		t.Fatalf("while Meta is failing: %d, want 502", code)
	}
	e.meta.setLookupFail(0)
	code, body, _ := e.getMedia(t, "/api", thread, msg)
	if code != 200 || !bytes.Equal(body, pngBytes) {
		t.Fatalf("after Meta recovered: %d (%d bytes)", code, len(body))
	}
}

// A file deleted from disk (lost volume) is fetched again instead of 404-ing forever.
func TestWAMedia_RefetchesWhenStoredFileIsMissing(t *testing.T) {
	e := newWAEnv(t)
	e.meta.addFile("MEDIA3", "image/png", pngBytes)
	e.post(t, payload("971501110022", "Dana", imageMsg("971501110022", "wamid.IMG3", "MEDIA3", "image/png")))
	thread, msg := e.messageID(t, "wamid.IMG3")
	if !e.waitStored(t, msg) {
		t.Fatal("not stored")
	}
	entries, _ := os.ReadDir(e.mediaDir)
	for _, f := range entries {
		_ = os.Remove(e.mediaDir + "/" + f.Name())
	}
	if code, body, _ := e.getMedia(t, "/api", thread, msg); code != 200 || !bytes.Equal(body, pngBytes) {
		t.Fatalf("after losing the file: %d (%d bytes)", code, len(body))
	}
}

// Many people opening the same message at once cause one download, not one each.
func TestWAMedia_ConcurrentRequestsDownloadOnce(t *testing.T) {
	e := newWAEnv(t)
	e.meta.addFile("MEDIA4", "image/png", pngBytes)
	e.meta.setLookupFail(503)
	e.post(t, payload("971501110023", "Dana", imageMsg("971501110023", "wamid.IMG4", "MEDIA4", "image/png")))
	thread, msg := e.messageID(t, "wamid.IMG4")
	time.Sleep(300 * time.Millisecond)
	e.meta.setLookupFail(0)
	_, dlBefore := e.meta.counts()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if code, _, _ := e.getMedia(t, "/api", thread, msg); code != 200 {
				t.Errorf("concurrent GET = %d", code)
			}
		}()
	}
	wg.Wait()
	if _, dl := e.meta.counts(); dl-dlBefore != 1 {
		t.Errorf("%d downloads for 8 concurrent requests, want 1", dl-dlBefore)
	}
	if n := e.storedFiles(t); n != 1 {
		t.Errorf("%d files stored, want 1", n)
	}
}

// What a customer sends is untrusted: a "text/html" file must never be served as HTML, and its
// file name cannot inject header content.
func TestWAMedia_HostileFilesAreServedAsDownloads(t *testing.T) {
	e := newWAEnv(t)
	page := []byte(`<html><script>fetch('/api/v1/users')</script></html>`)
	e.meta.addFile("EVIL1", "text/html", page)
	doc := fmt.Sprintf(`{"from":"971501110024","id":"wamid.EVIL1","timestamp":"1","type":"document","document":{"id":"EVIL1","mime_type":"text/html","filename":"..\/..\/x\"; filename*=UTF-8''evil.html\r\nX-Injected: 1"}}`)
	e.post(t, payload("971501110024", "Mallory", doc))
	thread, msg := e.messageID(t, "wamid.EVIL1")
	if !e.waitStored(t, msg) {
		t.Fatal("not stored")
	}
	code, body, h := e.getMedia(t, "/api", thread, msg)
	if code != 200 || !bytes.Equal(body, page) {
		t.Fatalf("GET = %d", code)
	}
	if h["Content-Type"] != "application/octet-stream" || !strings.HasPrefix(h["Content-Disposition"], "attachment") {
		t.Errorf("html served as %q / %q", h["Content-Type"], h["Content-Disposition"])
	}
	if h["X-Content-Type-Options"] != "nosniff" || h["X-Injected"] != "" {
		t.Errorf("headers: %v", h)
	}
	if strings.ContainsAny(h["Content-Disposition"], "\r\n/\\") {
		t.Errorf("unsafe filename in disposition: %q", h["Content-Disposition"])
	}
}

func TestWAMedia_OversizeIsRejectedAndNothingKept(t *testing.T) {
	e := newWAEnv(t) // cap is 1 MB
	e.meta.addFile("BIG1", "image/png", bytes.Repeat([]byte{1}, 2<<20))
	e.post(t, payload("971501110025", "Dana", imageMsg("971501110025", "wamid.BIG1", "BIG1", "image/png")))
	thread, msg := e.messageID(t, "wamid.BIG1")
	time.Sleep(300 * time.Millisecond)
	if code, _, _ := e.getMedia(t, "/api", thread, msg); code != 413 {
		t.Errorf("oversize file: %d, want 413", code)
	}
	if n := e.storedFiles(t); n != 0 {
		t.Errorf("%d files kept for a rejected download", n)
	}
}

func TestWAMedia_ExpiredAtMetaIsGone(t *testing.T) {
	e := newWAEnv(t) // MEDIA id not registered at the fake: Meta answers 404 / code 100
	e.post(t, payload("971501110026", "Dana", imageMsg("971501110026", "wamid.OLD1", "OLDMEDIA", "image/jpeg")))
	thread, msg := e.messageID(t, "wamid.OLD1")
	time.Sleep(300 * time.Millisecond)
	if code, _, _ := e.getMedia(t, "/api", thread, msg); code != 410 {
		t.Errorf("expired media: %d, want 410", code)
	}
}

func TestWAMedia_TextMessageHasNoAttachment(t *testing.T) {
	e := newWAEnv(t)
	e.post(t, payload("971501110027", "Dana", textMsg("971501110027", "wamid.TXT1", "hello")))
	thread, msg := e.messageID(t, "wamid.TXT1")
	if code, _, _ := e.getMedia(t, "/api", thread, msg); code != 404 {
		t.Errorf("text message media: %d, want 404", code)
	}
}

// Another company cannot read this company's customer files, even knowing both ids.
func TestWAMedia_OtherCompanyCannotRead(t *testing.T) {
	e := newWAEnv(t)
	e.meta.addFile("MEDIA5", "image/png", pngBytes)
	e.post(t, payload("971501110028", "Dana", imageMsg("971501110028", "wamid.IMG5", "MEDIA5", "image/png")))
	thread, msg := e.messageID(t, "wamid.IMG5")
	if !e.waitStored(t, msg) {
		t.Fatal("not stored")
	}
	if code, body, _ := e.getMedia(t, "/apib", thread, msg); code != 404 || bytes.Contains(body, pngBytes) {
		t.Errorf("other company got %d (%d bytes)", code, len(body))
	}
}

// ── outbound ─────────────────────────────────────────────────────────────────

func (e *waEnv) sendMultipart(t *testing.T, path string, fields map[string]string, filename, fileMime string, data []byte) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", fileMime)
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestWAOutbound_SendMediaLinkWithCaption(t *testing.T) {
	e := newWAEnv(t)
	thread := e.openThread(t, "971501110030", "wamid.IN30")
	path := "/api/threads/" + thread.String() + "/send-media"

	code, _ := e.sendJSON(t, path, `{"media_url":"https://cdn.example.com/villa.jpg","media_type":"image","caption":"Sea view, 12th floor"}`)
	if code != 201 {
		t.Fatalf("send = %d", code)
	}
	p := e.meta.lastPayload(t)
	img, _ := p["image"].(map[string]any)
	if p["type"] != "image" || img["link"] != "https://cdn.example.com/villa.jpg" || img["caption"] != "Sea view, 12th floor" {
		t.Errorf("payload to Meta = %v", p)
	}
	var body string
	_ = e.pool.QueryRow(context.Background(), `SELECT body FROM whatsapp_messages WHERE wa_message_id='wamid.SENT1' OR direction='outbound' ORDER BY sent_at DESC LIMIT 1`).Scan(&body)
	if body != "[Image] Sea view, 12th floor" {
		t.Errorf("stored body = %q", body)
	}
}

func TestWAOutbound_SendMediaRejectsBadRequests(t *testing.T) {
	e := newWAEnv(t)
	thread := e.openThread(t, "971501110031", "wamid.IN31")
	path := "/api/threads/" + thread.String() + "/send-media"
	before := e.meta.sentCount()
	for name, body := range map[string]string{
		"unknown type":     `{"media_url":"https://e.com/a.jpg","media_type":"template"}`,
		"json-key type":    `{"media_url":"https://e.com/a.jpg","media_type":"messaging_product"}`,
		"http link":        `{"media_url":"http://e.com/a.jpg","media_type":"image"}`,
		"relative link":    `{"media_url":"/uploads/a.jpg","media_type":"image"}`,
		"userinfo link":    `{"media_url":"https://user:pw@e.com/a.jpg","media_type":"image"}`,
		"audio caption":    `{"media_url":"https://e.com/a.ogg","media_type":"audio","caption":"hi"}`,
		"caption too long": `{"media_url":"https://e.com/a.jpg","media_type":"image","caption":"` + strings.Repeat("x", 1025) + `"}`,
		"missing url":      `{"media_type":"image"}`,
	} {
		if code, _ := e.sendJSON(t, path, body); code != 400 {
			t.Errorf("%s: status %d, want 400", name, code)
		}
	}
	if e.meta.sentCount() != before {
		t.Error("an invalid request reached Meta")
	}
}

func TestWAOutbound_UploadSendsFileAndKeepsCopy(t *testing.T) {
	e := newWAEnv(t)
	thread := e.openThread(t, "971501110032", "wamid.IN32")
	path := "/api/threads/" + thread.String() + "/send-media"

	code, _ := e.sendMultipart(t, path, map[string]string{"caption": "Floor plan"}, "plan.png", "image/png", pngBytes)
	if code != 201 {
		t.Fatalf("upload send = %d", code)
	}
	if len(e.meta.uploads) != 1 || !bytes.Equal(e.meta.uploads[0].Data, pngBytes) || e.meta.uploads[0].Mime != "image/png" {
		t.Fatalf("upload to Meta = %+v", e.meta.uploads)
	}
	p := e.meta.lastPayload(t)
	img := p["image"].(map[string]any)
	if img["id"] != "UPLOAD1" || img["caption"] != "Floor plan" || img["link"] != nil {
		t.Errorf("message payload = %v", p)
	}

	// The thread shows what was sent, from our own copy.
	var msg uuid.UUID
	_ = e.pool.QueryRow(context.Background(), `SELECT id FROM whatsapp_messages WHERE thread_id=$1 AND direction='outbound'`, thread).Scan(&msg)
	if code, body, _ := e.getMedia(t, "/api", thread, msg); code != 200 || !bytes.Equal(body, pngBytes) {
		t.Errorf("sent file not viewable: %d (%d bytes)", code, len(body))
	}
}

func TestWAOutbound_UploadValidation(t *testing.T) {
	e := newWAEnv(t)
	thread := e.openThread(t, "971501110033", "wamid.IN33")
	path := "/api/threads/" + thread.String() + "/send-media"
	before := e.meta.sentCount()

	cases := map[string]struct {
		fileMime string
		name     string
		data     []byte
		fields   map[string]string
	}{
		"html as document":      {"text/html", "page.html", []byte("<script>x</script>"), nil},
		"svg as image":          {"image/svg+xml", "a.svg", []byte("<svg/>"), nil},
		"exe":                   {"application/x-msdownload", "a.exe", []byte("MZ"), nil},
		"script renamed to png": {"image/png", "evil.png", []byte("#!/bin/sh\nrm -rf /"), nil},
		"fake pdf":              {"application/pdf", "a.pdf", []byte("not a pdf"), nil},
		"oversize image":        {"image/png", "big.png", append(append([]byte{}, pngBytes...), bytes.Repeat([]byte{0}, 6<<20)...), nil},
		"empty file":            {"image/png", "empty.png", nil, nil},
		"audio with caption":    {"audio/ogg", "a.ogg", []byte("OggS"), map[string]string{"caption": "hi"}},
		"type mismatch":         {"image/png", "a.png", pngBytes, map[string]string{"media_type": "audio"}},
	}
	for name, tc := range cases {
		if code, _ := e.sendMultipart(t, path, tc.fields, tc.name, tc.fileMime, tc.data); code != 400 {
			t.Errorf("%s: status %d, want 400", name, code)
		}
	}
	if e.meta.sentCount() != before || len(e.meta.uploads) != 0 {
		t.Error("a rejected file reached Meta")
	}
	if n := e.storedFiles(t); n != 0 {
		t.Errorf("%d files stored for rejected uploads", n)
	}
}
