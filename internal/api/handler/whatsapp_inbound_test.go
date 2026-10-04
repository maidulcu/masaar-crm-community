package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func parse(t *testing.T, raw string) inboundContent {
	t.Helper()
	var m waMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	return parseInbound(m)
}

func TestParseInbound(t *testing.T) {
	cases := []struct {
		name, raw     string
		wantBody      string // substring
		wantMediaID   string
		wantMediaMime string
		skip          bool
	}{
		{"text", `{"type":"text","text":{"body":"مرحبا, interested in the villa"}}`, "مرحبا, interested in the villa", "", "", false},
		{"image with caption", `{"type":"image","image":{"id":"M1","mime_type":"image/jpeg","caption":"the villa"}}`, "[Image] the villa", "M1", "image/jpeg", false},
		{"image without caption", `{"type":"image","image":{"id":"M1","mime_type":"image/jpeg"}}`, "[Image]", "M1", "image/jpeg", false},
		{"document", `{"type":"document","document":{"id":"M2","filename":"passport.pdf","mime_type":"application/pdf"}}`, "[Document: passport.pdf]", "M2", "application/pdf", false},
		{"voice note", `{"type":"audio","audio":{"id":"M3","mime_type":"audio/ogg","voice":true}}`, "[Voice message]", "M3", "audio/ogg", false},
		{"video", `{"type":"video","video":{"id":"M4","mime_type":"video/mp4","caption":"tour"}}`, "[Video] tour", "M4", "video/mp4", false},
		{"sticker", `{"type":"sticker","sticker":{"id":"M5","mime_type":"image/webp"}}`, "[Sticker]", "M5", "image/webp", false},
		{"location", `{"type":"location","location":{"latitude":25.2,"longitude":55.27,"name":"Marina Gate"}}`, "Marina Gate (25.200000,55.270000)", "", "", false},
		{"location without name", `{"type":"location","location":{"latitude":25.2,"longitude":55.27}}`, "[Location] 25.200000,55.270000", "", "", false},
		{"button reply", `{"type":"interactive","interactive":{"type":"button_reply","button_reply":{"id":"b1","title":"Book viewing"}}}`, "Book viewing", "", "", false},
		{"list reply", `{"type":"interactive","interactive":{"type":"list_reply","list_reply":{"id":"l1","title":"2 bedroom","description":"Marina"}}}`, "2 bedroom — Marina", "", "", false},
		{"template quick reply", `{"type":"button","button":{"text":"Yes, interested","payload":"p"}}`, "Yes, interested", "", "", false},
		{"contact card", `{"type":"contacts","contacts":[{"name":{"formatted_name":"Omar Ali"}}]}`, "[Contact card] Omar Ali", "", "", false},
		{"reaction is not a message", `{"type":"reaction","reaction":{"emoji":"👍"}}`, "", "", "", true},
		{"unsupported", `{"type":"unsupported"}`, "[Unsupported message]", "", "", false},
		{"unknown future type", `{"type":"order"}`, "[order]", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parse(t, tc.raw)
			if got.Skip != tc.skip {
				t.Fatalf("skip = %v, want %v", got.Skip, tc.skip)
			}
			if !strings.Contains(got.Body, tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", got.Body, tc.wantBody)
			}
			if got.MediaID != tc.wantMediaID || got.MediaMime != tc.wantMediaMime {
				t.Errorf("media = %q/%q, want %q/%q", got.MediaID, got.MediaMime, tc.wantMediaID, tc.wantMediaMime)
			}
		})
	}
}

// PostgreSQL cannot store NUL bytes or invalid UTF-8: such a message must be cleaned, not left
// to fail (and be redelivered by Meta) forever.
func TestParseInboundMakesBodyStorable(t *testing.T) {
	got := parse(t, `{"type":"text","text":{"body":"hi\u0000there"}}`)
	if strings.ContainsRune(got.Body, 0) {
		t.Error("NUL byte kept in body")
	}
	if got := cleanBody("bad\xffbytes"); !utf8.ValidString(got) {
		t.Errorf("invalid UTF-8 kept: %q", got)
	}
	long := parse(t, `{"type":"text","text":{"body":"`+strings.Repeat("x", 20000)+`"}}`)
	if n := utf8.RuneCountInString(long.Body); n != maxInboundBody {
		t.Errorf("long body not truncated: %d runes", n)
	}
}

func TestDescribeStatusErrors(t *testing.T) {
	var st waStatus
	_ = json.Unmarshal([]byte(`{"errors":[{"code":131047,"title":"Re-engagement message","message":"More than 24 hours have passed"}]}`), &st)
	if got := describeStatusErrors(st); !strings.HasPrefix(got, "131047: Re-engagement message") {
		t.Errorf("got %q", got)
	}
	if got := describeStatusErrors(waStatus{}); got != "delivery failed" {
		t.Errorf("no-error fallback = %q", got)
	}
}
