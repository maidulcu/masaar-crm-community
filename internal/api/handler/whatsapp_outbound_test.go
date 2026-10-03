package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/whatsapp"
)

// Agent replies must become part of the conversation: stored in the thread, counted, shown
// with their delivery state, and kept out of the way of a failed send.
func TestWAOutbound_SendRecordsMessageInThread(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()

	var sent int
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent++
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.AGENT1"}]}`))
	}))
	defer meta.Close()

	// A customer writes first (creates contact + thread).
	e.post(t, payload("971501110011", "Reem", textMsg("971501110011", "wamid.IN1", "hi")))
	var threadID uuid.UUID
	if err := e.pool.QueryRow(ctx, `SELECT id FROM whatsapp_threads WHERE company_id=$1`, e.companyID).Scan(&threadID); err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	if _, err := e.pool.Exec(ctx, `INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1,$2,'Agent',$3,'x','agent')`, userID, e.companyID, userID.String()+"@test.local"); err != nil {
		t.Fatal(err)
	}

	waRepo := repo.NewWhatsAppRepo(e.pool)
	h := NewWhatsAppOutboundHandler(
		whatsapp.NewSender(&whatsapp.SenderConfig{BaseURL: meta.URL, PhoneNumberID: "PNID1", AccessToken: "tok"}),
		repo.NewWhatsAppOutboundRepo(e.pool), waRepo, repo.NewCommunicationHistoryRepo(e.pool))
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("company_id", e.companyID.String())
		return c.Next()
	})
	app.Post("/threads/:id/send-message", h.SendMessage)

	send := func(body string) (int, map[string]any) {
		req := httptest.NewRequest("POST", "/threads/"+threadID.String()+"/send-message", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	if code, _ := send(`{"message":"Hello Reem, the villa is available"}`); code != 201 {
		t.Fatalf("send = %d, want 201", code)
	}

	// Stored in the conversation, in order, with the thread's counters updated.
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
	e.post(t, `{"entry":[{"changes":[{"field":"messages","value":{"statuses":[{"id":"wamid.AGENT1","status":"delivered","recipient_id":"971501110011"}]}}]}]}`)
	msgs, _ = waRepo.GetMessages(tctx, threadID, 50)
	if msgs[1].Status != "delivered" {
		t.Errorf("after receipt status = %q, want delivered", msgs[1].Status)
	}

	// Over-long and blank messages are rejected before reaching Meta.
	before := sent
	if code, _ := send(`{"message":"` + strings.Repeat("a", 4097) + `"}`); code != 400 {
		t.Errorf("4097-char message = %d, want 400", code)
	}
	if code, _ := send(`{"message":"   "}`); code != 400 {
		t.Errorf("blank message = %d, want 400", code)
	}
	if sent != before {
		t.Error("invalid messages were forwarded to Meta")
	}
}
