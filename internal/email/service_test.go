package email

import (
	"errors"
	"mime"
	"net/mail"
	"strings"
	"testing"
)

func TestParseAddressRejectsHeaderInjection(t *testing.T) {
	bad := []string{
		"",
		"victim@example.com\r\nBcc: attacker@example.com",
		"victim@example.com\nBcc: attacker@example.com",
		"not an address",
	}
	for _, in := range bad {
		if _, err := parseAddress(in); !errors.Is(err, ErrInvalidHeader) {
			t.Errorf("parseAddress(%q) err = %v, want ErrInvalidHeader", in, err)
		}
	}
	addr, err := parseAddress("Jane Doe <jane@example.com>")
	if err != nil || addr.Address != "jane@example.com" {
		t.Fatalf("valid address rejected: %v %v", addr, err)
	}
}

func TestBuildMessage(t *testing.T) {
	from := &mail.Address{Name: "Masaar", Address: "noreply@example.com"}
	to := &mail.Address{Address: "jane@example.com"}

	if _, err := buildMessage(from, to, "Hi\r\nBcc: evil@example.com", false, "x"); !errors.Is(err, ErrInvalidHeader) {
		t.Fatalf("subject with CRLF must be rejected, got %v", err)
	}

	raw, err := buildMessage(from, to, "عرض عقار", true, "<p>مرحبا</p>")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("message does not parse: %v", err)
	}
	if got := msg.Header.Get("Bcc"); got != "" {
		t.Errorf("unexpected Bcc header %q", got)
	}
	subj, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || subj != "عرض عقار" {
		t.Errorf("subject round trip = %q, %v", subj, err)
	}
	if ct := msg.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content type = %q", ct)
	}
	if msg.Header.Get("Date") == "" {
		t.Error("missing Date header")
	}
}
