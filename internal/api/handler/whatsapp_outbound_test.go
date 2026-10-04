package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// openThread has a customer write first (creating contact + thread) and returns the thread id.
func (e *waEnv) openThread(t *testing.T, number, msgID string) uuid.UUID {
	t.Helper()
	e.post(t, payload(number, "Customer", textMsg(number, msgID, "hi")))
	var threadID uuid.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT t.id FROM whatsapp_threads t JOIN contacts c ON c.id = t.contact_id WHERE t.company_id=$1 AND c.phone_wa=$2`,
		e.companyID, "+"+number).Scan(&threadID); err != nil {
		t.Fatal(err)
	}
	return threadID
}

func (e *waEnv) sendJSON(t *testing.T, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// Agent replies must become part of the conversation: stored in the thread, counted, shown
// with their delivery state, and kept out of the way of a failed send.
func TestWAOutbound_SendRecordsMessageInThread(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	threadID := e.openThread(t, "971501110011", "wamid.IN1")
	path := "/api/threads/" + threadID.String() + "/send-message"

	if code, _ := e.sendJSON(t, path, `{"message":"Hello Reem, the villa is available"}`); code != 201 {
		t.Fatalf("send = %d, want 201", code)
	}

	// Stored in the conversation, in order, with the thread's counters updated.
	waRepo := repo.NewWhatsAppRepo(e.pool)
	tctx := tenantCtx(e.companyID)
	msgs, err := waRepo.GetMessages(tctx, threadID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Direction != domain.DirectionInbound || msgs[1].Direction != domain.DirectionOutbound {
		t.Fatalf("conversation = %+v, want [inbound, outbound]", msgs)
	}
	if msgs[1].Body != "Hello Reem, the villa is available" || msgs[1].Status != "sent" {
		t.Errorf("outbound message = %q status %q", msgs[1].Body, msgs[1].Status)
	}
	var count int
	_ = e.pool.QueryRow(ctx, `SELECT message_count FROM whatsapp_threads WHERE id=$1`, threadID).Scan(&count)
	if count != 2 {
		t.Errorf("thread message_count = %d, want 2", count)
	}

	// A delivery receipt shows up on the message in the thread view.
	e.post(t, `{"entry":[{"changes":[{"field":"messages","value":{"statuses":[{"id":"wamid.SENT1","status":"delivered","recipient_id":"971501110011"}]}}]}]}`)
	msgs, _ = waRepo.GetMessages(tctx, threadID, 50)
	if msgs[1].Status != "delivered" {
		t.Errorf("after receipt status = %q, want delivered", msgs[1].Status)
	}

	// Over-long and blank messages are rejected before reaching Meta.
	before := e.meta.sentCount()
	if code, _ := e.sendJSON(t, path, `{"message":"`+strings.Repeat("a", 4097)+`"}`); code != 400 {
		t.Errorf("4097-char message = %d, want 400", code)
	}
	if code, _ := e.sendJSON(t, path, `{"message":"   "}`); code != 400 {
		t.Errorf("blank message = %d, want 400", code)
	}
	if e.meta.sentCount() != before {
		t.Error("invalid messages were forwarded to Meta")
	}
}
