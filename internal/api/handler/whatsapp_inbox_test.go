package handler

import (
	"context"
	"strings"
	"testing"
)

// Replying STOP (or إيقاف) records a WhatsApp opt-out and confirms it; START opts back in.
func TestWAInbox_StopAndStart(t *testing.T) {
	e := newWAEnv(t)
	number := "971501119901"
	e.post(t, payload(number, "Reem", textMsg(number, "wamid.STOP1", "STOP")))
	latest := func() string {
		var s string
		_ = e.pool.QueryRow(context.Background(), `SELECT cc.status FROM contact_consents cc JOIN contacts c ON c.id = cc.contact_id
			WHERE c.company_id = $1 AND c.phone_wa = $2 AND cc.channel = 'whatsapp' ORDER BY cc.created_at DESC, cc.id DESC LIMIT 1`,
			e.companyID, "+"+number).Scan(&s)
		return s
	}
	if got := latest(); got != "withdrawn" {
		t.Fatalf("after STOP, consent = %q", got)
	}
	e.meta.mu.Lock()
	n := len(e.meta.sentPayload)
	confirm, _ := e.meta.sentPayload[n-1]["text"].(map[string]any)
	e.meta.mu.Unlock()
	if body, _ := confirm["body"].(string); !strings.Contains(body, "no longer receive") {
		t.Fatalf("confirmation = %v", confirm)
	}
	e.post(t, payload(number, "Reem", textMsg(number, "wamid.STOP2", "اشتراك")))
	if got := latest(); got != "granted" {
		t.Fatalf("after اشتراك, consent = %q", got)
	}
	// An ordinary message that mentions "stop" is not an opt-out.
	e.post(t, payload(number, "Reem", textMsg(number, "wamid.STOP3", "please stop by at 5")))
	if got := latest(); got != "granted" {
		t.Fatalf("a normal message changed consent to %q", got)
	}
}

// Outside business hours a customer gets the auto-reply once per conversation every 12 hours.
func TestWAInbox_AutoReplyOutsideHours(t *testing.T) {
	e := newWAEnv(t)
	// Closed every day: always outside hours.
	if _, err := e.pool.Exec(context.Background(), `UPDATE company_settings SET wa_auto_reply_enabled = TRUE,
		wa_auto_reply_en = 'Thanks! We reply from 9:00.', wa_business_hours = '{"week":[{},{},{},{},{},{},{}],"specials":[]}'
		WHERE company_id = $1`, e.companyID); err != nil {
		t.Fatal(err)
	}
	number := "971501119902"
	sent := func() int {
		e.meta.mu.Lock()
		defer e.meta.mu.Unlock()
		c := 0
		for _, p := range e.meta.sentPayload {
			if txt, ok := p["text"].(map[string]any); ok && txt["body"] == "Thanks! We reply from 9:00." {
				c++
			}
		}
		return c
	}
	e.post(t, payload(number, "Omar", textMsg(number, "wamid.AR1", "hello")))
	e.post(t, payload(number, "Omar", textMsg(number, "wamid.AR2", "anyone there?")))
	if got := sent(); got != 1 {
		t.Fatalf("auto-replies sent: %d, want 1", got)
	}
	if n := e.count(t, `SELECT count(*) FROM whatsapp_messages m JOIN whatsapp_threads th ON th.id = m.thread_id
		WHERE th.company_id = $1 AND m.direction = 'outbound' AND m.body = 'Thanks! We reply from 9:00.'`, e.companyID); n != 1 {
		t.Fatalf("auto-reply recorded %d times in the conversation", n)
	}
}
