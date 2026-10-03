package handler

// End-to-end tests of the inbound WhatsApp webhook against a real database, driven with
// payloads shaped like Meta's WhatsApp Cloud API. Skipped unless TEST_DATABASE_URL is set.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/testdb"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

const waTestSecret = "test-app-secret"

type waEnv struct {
	app       *fiber.App
	pool      *pgxpool.Pool
	companyID uuid.UUID
}

func newWAEnv(t *testing.T) *waEnv {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	testdb.Migrate(t, url)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	id := uuid.New()
	for _, q := range []string{
		`INSERT INTO companies (id, name, subdomain) VALUES ($1::uuid, 'WA test', 'wa-' || substr($1::uuid::text,1,8))`,
		`INSERT INTO company_settings (company_id, name, vat_number, business_address) VALUES ($1, 'WA test', '', '')`,
	} {
		if _, err := pool.Exec(ctx, q, id); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		c := context.Background()
		for _, q := range []string{
			`DELETE FROM whatsapp_outbound WHERE company_id = $1`,
			`DELETE FROM whatsapp_messages WHERE thread_id IN (SELECT id FROM whatsapp_threads WHERE company_id = $1)`,
			`DELETE FROM whatsapp_threads WHERE company_id = $1`,
			`DELETE FROM contacts WHERE company_id = $1`,
			`DELETE FROM company_settings WHERE company_id = $1`,
			`DELETE FROM companies WHERE id = $1`,
		} {
			_, _ = pool.Exec(c, q, id)
		}
	})

	cfg := &config.Config{WAAppSecret: waTestSecret, AppCompanyID: id.String(), WAVerifyToken: "verify-me"}
	h := NewWhatsAppHandler(repo.NewWhatsAppRepo(pool), repo.NewContactRepo(pool), repo.NewWhatsAppOutboundRepo(pool), repo.NewCommunicationHistoryRepo(pool), nil, ws.NewHub(), cfg)
	app := fiber.New()
	app.Get("/webhooks/whatsapp", h.Verify)
	app.Post("/webhooks/whatsapp", h.Receive)
	return &waEnv{app: app, pool: pool, companyID: id}
}

func (e *waEnv) post(t *testing.T, body string) int {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(waTestSecret))
	mac.Write([]byte(body))
	req := httptest.NewRequest("POST", "/webhooks/whatsapp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func (e *waEnv) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// payload builds a Meta-style webhook body with the given raw "messages" JSON objects.
func payload(from, name string, messages ...string) string {
	return fmt.Sprintf(`{"object":"whatsapp_business_account","entry":[{"id":"WABA","changes":[{"field":"messages","value":{
		"messaging_product":"whatsapp","metadata":{"display_phone_number":"15550001111","phone_number_id":"PNID1"},
		"contacts":[{"profile":{"name":%q},"wa_id":%q}],"messages":[%s]}}]}]}`, name, from, strings.Join(messages, ","))
}

func textMsg(from, id, body string) string {
	return fmt.Sprintf(`{"from":%q,"id":%q,"timestamp":"1700000000","type":"text","text":{"body":%q}}`, from, id, body)
}

// Meta redelivers webhooks until it gets a 2xx. A redelivered message must be acknowledged,
// not turned into a 500, or Meta retries it forever.
func TestWAWebhook_RedeliveredMessageIsAcknowledged(t *testing.T) {
	e := newWAEnv(t)
	body := payload("971501110001", "Sara", textMsg("971501110001", "wamid.DUP1", "hello"))
	if got := e.post(t, body); got != 200 {
		t.Fatalf("first delivery = %d, want 200", got)
	}
	if got := e.post(t, body); got != 200 {
		t.Errorf("redelivery of the same message = %d, want 200 (Meta would retry forever on 5xx)", got)
	}
	if n := e.count(t, `SELECT count(*) FROM whatsapp_messages WHERE wa_message_id = 'wamid.DUP1'`); n != 1 {
		t.Errorf("stored %d copies, want exactly 1", n)
	}
}

// One bad/duplicate message must not make the rest of the same payload vanish.
func TestWAWebhook_BatchContinuesPastDuplicate(t *testing.T) {
	e := newWAEnv(t)
	e.post(t, payload("971501110002", "Omar", textMsg("971501110002", "wamid.B1", "one")))
	status := e.post(t, payload("971501110002", "Omar",
		textMsg("971501110002", "wamid.B1", "one"), // already stored
		textMsg("971501110002", "wamid.B2", "two"), // new
	))
	if n := e.count(t, `SELECT count(*) FROM whatsapp_messages WHERE wa_message_id = 'wamid.B2'`); n != 1 {
		t.Errorf("message after a duplicate in the same payload was lost (status %d)", status)
	}
}

// Contacts entered in the CRM are E.164 with a leading "+", while Meta sends wa_id without
// it. The same person must resolve to one contact.
func TestWAWebhook_MatchesCRMContactEnteredWithPlus(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `INSERT INTO contacts (id, company_id, phone_wa, full_name, email) VALUES (uuid_generate_v4(), $1, '+971501110003', 'Layla (agent-entered)', '')`, e.companyID); err != nil {
		t.Fatal(err)
	}
	e.post(t, payload("971501110003", "Layla WhatsApp Name", textMsg("971501110003", "wamid.P1", "hi")))
	if n := e.count(t, `SELECT count(*) FROM contacts WHERE company_id = $1 AND regexp_replace(phone_wa, '\D', '', 'g') = '971501110003'`, e.companyID); n != 1 {
		t.Errorf("same person became %d contacts (duplicate created for the WhatsApp number)", n)
	}
}

// An inbound message must not overwrite the name an agent saved for the contact.
func TestWAWebhook_DoesNotOverwriteAgentEditedName(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `INSERT INTO contacts (id, company_id, phone_wa, full_name, email) VALUES (uuid_generate_v4(), $1, '971501110004', 'Khalid Al Mansoori (VIP buyer)', '')`, e.companyID); err != nil {
		t.Fatal(err)
	}
	e.post(t, payload("971501110004", "k", textMsg("971501110004", "wamid.N1", "hi")))
	var name string
	_ = e.pool.QueryRow(ctx, `SELECT full_name FROM contacts WHERE company_id=$1 AND phone_wa='971501110004'`, e.companyID).Scan(&name)
	if name != "Khalid Al Mansoori (VIP buyer)" {
		t.Errorf("contact name was overwritten by the WhatsApp profile name: %q", name)
	}
}

// Meta sends media as {"image":{"id":"<media id>","mime_type":...,"caption":...}} — there is
// no "url" field. Such messages must still appear in the inbox.
func TestWAWebhook_StoresMediaMessagesAsSentByMeta(t *testing.T) {
	e := newWAEnv(t)
	for i, m := range []string{
		`{"from":"971501110005","id":"wamid.M1","timestamp":"1","type":"image","image":{"id":"MEDIA1","mime_type":"image/jpeg","sha256":"x","caption":"the villa"}}`,
		`{"from":"971501110005","id":"wamid.M2","timestamp":"1","type":"document","document":{"id":"MEDIA2","filename":"passport.pdf","mime_type":"application/pdf"}}`,
		`{"from":"971501110005","id":"wamid.M3","timestamp":"1","type":"audio","audio":{"id":"MEDIA3","mime_type":"audio/ogg; codecs=opus","voice":true}}`,
		`{"from":"971501110005","id":"wamid.M4","timestamp":"1","type":"location","location":{"latitude":25.2,"longitude":55.27,"name":"Marina Gate"}}`,
		`{"from":"971501110005","id":"wamid.M5","timestamp":"1","type":"interactive","interactive":{"type":"button_reply","button_reply":{"id":"b1","title":"Book viewing"}}}`,
		`{"from":"971501110005","id":"wamid.M6","timestamp":"1","type":"button","button":{"text":"Yes, interested","payload":"p1"}}`,
	} {
		e.post(t, payload("971501110005", "Noor", m))
		id := fmt.Sprintf("wamid.M%d", i+1)
		if n := e.count(t, `SELECT count(*) FROM whatsapp_messages WHERE wa_message_id = $1`, id); n != 1 {
			t.Errorf("message %s (%s) was dropped", id, strings.Split(strings.Split(m, `"type":"`)[1], `"`)[0])
		}
	}
}

// Delivery receipts arrive as value.statuses[] and must update the outbound message.
func TestWAWebhook_AppliesDeliveryStatusUpdates(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	e.post(t, payload("971501110006", "Huda", textMsg("971501110006", "wamid.S0", "hi"))) // creates contact + thread
	var threadID uuid.UUID
	if err := e.pool.QueryRow(ctx, `SELECT id FROM whatsapp_threads WHERE company_id=$1`, e.companyID).Scan(&threadID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO whatsapp_outbound (thread_id, to_number, message_body, status, wa_message_id, company_id) VALUES ($1,'971501110006','reply','sent','wamid.OUT1',$2)`, threadID, e.companyID); err != nil {
		t.Fatal(err)
	}
	status := func(s string, extra string) string {
		return fmt.Sprintf(`{"object":"whatsapp_business_account","entry":[{"id":"WABA","changes":[{"field":"messages","value":{"messaging_product":"whatsapp","metadata":{"phone_number_id":"PNID1"},"statuses":[{"id":"wamid.OUT1","status":%q,"timestamp":"1700000100","recipient_id":"971501110006"%s}]}}]}]}`, s, extra)
	}
	for _, s := range []string{"delivered", "read"} {
		if got := e.post(t, status(s, "")); got != 200 {
			t.Fatalf("status webhook = %d", got)
		}
		var cur string
		_ = e.pool.QueryRow(ctx, `SELECT status FROM whatsapp_outbound WHERE wa_message_id='wamid.OUT1'`).Scan(&cur)
		if cur != s {
			t.Errorf("after %q receipt outbound status = %q", s, cur)
		}
	}
	// A message that fails after being accepted by Meta must record the reason.
	if _, err := e.pool.Exec(ctx, `INSERT INTO whatsapp_outbound (thread_id, to_number, message_body, status, wa_message_id, company_id) VALUES ($1,'971501110006','reply 2','sent','wamid.OUT1F',$2)`, threadID, e.companyID); err != nil {
		t.Fatal(err)
	}
	failed := strings.Replace(status("failed", `,"errors":[{"code":131047,"title":"Re-engagement message"}]`), "wamid.OUT1", "wamid.OUT1F", 1)
	e.post(t, failed)
	var cur, msg string
	_ = e.pool.QueryRow(ctx, `SELECT status, COALESCE(error_message,'') FROM whatsapp_outbound WHERE wa_message_id='wamid.OUT1F'`).Scan(&cur, &msg)
	if cur != "failed" || !strings.Contains(msg, "131047") {
		t.Errorf("failed receipt not recorded: status=%q error=%q", cur, msg)
	}
	// ...but a failure report cannot undo a message already delivered.
	e.post(t, status("failed", `,"errors":[{"code":1,"title":"late"}]`))
	_ = e.pool.QueryRow(ctx, `SELECT status FROM whatsapp_outbound WHERE wa_message_id='wamid.OUT1'`).Scan(&cur)
	if cur != "read" {
		t.Errorf("a late failure receipt overwrote status 'read': %q", cur)
	}
}

// Receipts can arrive out of order (read before delivered); status must never go backwards.
func TestWAWebhook_StatusNeverRegresses(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	e.post(t, payload("971501110007", "Ali", textMsg("971501110007", "wamid.R0", "hi")))
	var threadID uuid.UUID
	_ = e.pool.QueryRow(ctx, `SELECT id FROM whatsapp_threads WHERE company_id=$1`, e.companyID).Scan(&threadID)
	_, _ = e.pool.Exec(ctx, `INSERT INTO whatsapp_outbound (thread_id, to_number, message_body, status, wa_message_id, company_id) VALUES ($1,'971501110007','reply','read','wamid.OUT2',$2)`, threadID, e.companyID)
	e.post(t, fmt.Sprintf(`{"entry":[{"changes":[{"field":"messages","value":{"statuses":[{"id":"wamid.OUT2","status":"delivered","timestamp":"1","recipient_id":"971501110007"}]}}]}]}`))
	var cur string
	_ = e.pool.QueryRow(ctx, `SELECT status FROM whatsapp_outbound WHERE wa_message_id='wamid.OUT2'`).Scan(&cur)
	if cur != "read" {
		t.Errorf("late 'delivered' receipt regressed status to %q", cur)
	}
}

func TestWAVerify_RejectsEmptyAndWrongToken(t *testing.T) {
	e := newWAEnv(t)
	for _, tc := range []struct {
		query string
		want  int
	}{
		{"hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=abc", 200},
		{"hub.mode=subscribe&hub.verify_token=nope&hub.challenge=abc", 403},
		{"hub.mode=subscribe&hub.challenge=abc", 403},
	} {
		resp, _ := e.app.Test(httptest.NewRequest("GET", "/webhooks/whatsapp?"+tc.query, nil))
		if resp.StatusCode != tc.want {
			t.Errorf("%s = %d, want %d", tc.query, resp.StatusCode, tc.want)
		}
	}
}

// A lead's timeline shows WhatsApp traffic: inbound messages are added, and delivery receipts
// update the matching entry.
func TestWAWebhook_WritesLeadTimeline(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	contactID := uuid.New()
	if _, err := e.pool.Exec(ctx, `INSERT INTO contacts (id, company_id, phone_wa, full_name, email) VALUES ($1,$2,'+971501110008','Mona','')`, contactID, e.companyID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO leads (id, company_id, contact_id, stage, source, currency) VALUES (uuid_generate_v4(),$1,$2,'new','whatsapp','AED')`, e.companyID, contactID); err != nil {
		t.Fatal(err)
	}
	e.post(t, payload("971501110008", "Mona", textMsg("971501110008", "wamid.T1", "is it available?")))
	if n := e.count(t, `SELECT count(*) FROM communication_history WHERE company_id=$1 AND external_id='wamid.T1' AND communication_type='whatsapp_inbound' AND body='is it available?'`, e.companyID); n != 1 {
		t.Fatalf("inbound message missing from the lead timeline (%d entries)", n)
	}
	// Redelivery must not add a second entry.
	e.post(t, payload("971501110008", "Mona", textMsg("971501110008", "wamid.T1", "is it available?")))
	if n := e.count(t, `SELECT count(*) FROM communication_history WHERE company_id=$1 AND external_id='wamid.T1'`, e.companyID); n != 1 {
		t.Errorf("redelivery duplicated the timeline entry (%d)", n)
	}
	// A contact with no lead simply has no timeline entry, but the thread still has the message.
	e.post(t, payload("971501110009", "Nobody", textMsg("971501110009", "wamid.T2", "hello")))
	if n := e.count(t, `SELECT count(*) FROM whatsapp_messages WHERE wa_message_id='wamid.T2'`); n != 1 {
		t.Error("message from a contact without a lead was not stored")
	}
}

// A redelivered message must not reopen a thread an agent has since closed, nor bump counters.
func TestWAWebhook_RedeliveryDoesNotReopenClosedThread(t *testing.T) {
	e := newWAEnv(t)
	ctx := context.Background()
	body := payload("971501110010", "Sami", textMsg("971501110010", "wamid.C1", "hello"))
	e.post(t, body)
	if _, err := e.pool.Exec(ctx, `UPDATE whatsapp_threads SET thread_status='closed' WHERE company_id=$1`, e.companyID); err != nil {
		t.Fatal(err)
	}
	e.post(t, body)
	var status string
	var count int
	_ = e.pool.QueryRow(ctx, `SELECT thread_status, message_count FROM whatsapp_threads WHERE company_id=$1`, e.companyID).Scan(&status, &count)
	if status != "closed" || count != 1 {
		t.Errorf("redelivery changed the thread: status=%q message_count=%d", status, count)
	}
	// A genuinely new message does reopen it.
	e.post(t, payload("971501110010", "Sami", textMsg("971501110010", "wamid.C2", "are you there?")))
	_ = e.pool.QueryRow(ctx, `SELECT thread_status FROM whatsapp_threads WHERE company_id=$1`, e.companyID).Scan(&status)
	if status != "open" {
		t.Errorf("new message left the thread %q, want open", status)
	}
}

func tenantCtx(id uuid.UUID) context.Context { return tenant.With(context.Background(), id) }
