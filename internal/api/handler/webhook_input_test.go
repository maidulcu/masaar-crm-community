package handler

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookCreateInput(t *testing.T) {
	for raw, want := range map[string]string{
		`"lead.created, lead.won"`:              "lead.created,lead.won",
		`["lead.created","lead.stage_changed"]`: "lead.created,lead.stage_changed",
		`""`:                                    "",
	} {
		var e webhookEvents
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		got, msg := normalizeWebhookEvents(string(e))
		if msg != "" || got != want {
			t.Errorf("%s = %q %q, want %q", raw, got, msg, want)
		}
	}
	if _, msg := normalizeWebhookEvents("lead.created,lead.deleted"); msg == "" {
		t.Error("unknown event accepted")
	}
	for u, ok := range map[string]bool{
		"https://hooks.example.com/x":          true,
		"http://hooks.example.com/x":           false,
		"https://":                             false,
		"https://user:pw@hooks.example.com":    false,
		"https://" + strings.Repeat("a", 2050): false,
	} {
		if got := checkWebhookURL(u) == ""; got != ok {
			t.Errorf("checkWebhookURL(%.40q) ok=%v, want %v", u, got, ok)
		}
	}
}
