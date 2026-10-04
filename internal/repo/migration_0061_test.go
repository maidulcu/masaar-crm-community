package repo

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The contact-merge block of migration 0061 is idempotent, so the test seeds the kind of
// duplicates real installs have (WhatsApp "971…" vs agent-typed "+971…", formatted numbers)
// into an already-migrated database and runs the block again.
func TestMigration0061MergesDuplicateContacts(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	raw, err := os.ReadFile("../../migrations/0061_whatsapp_fixes.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	start := strings.Index(sql, "-- +goose StatementBegin")
	end := strings.Index(sql, "-- +goose StatementEnd")
	if start < 0 || end < start {
		t.Fatal("merge block markers not found in migration")
	}
	block := sql[start+len("-- +goose StatementBegin") : end]

	// Company A: an agent-entered contact (older) and a duplicate created by an inbound message.
	keep, dup := uuid.New(), uuid.New()
	must(t, e.pool, `INSERT INTO contacts (id, company_id, phone_wa, full_name, email, lead_score, created_at)
		VALUES ($1, $2, '+971501110001', 'Khalid (VIP)', '', 10, now() - interval '2 days')`, keep, e.a.id)
	must(t, e.pool, `INSERT INTO contacts (id, company_id, phone_wa, full_name, email, lead_score, assigned_to, created_at)
		VALUES ($1, $2, '971501110001', '971501110001', 'k@example.com', 40, $3, now())`, dup, e.a.id, e.a.user)

	lead := uuid.New()
	must(t, e.pool, `INSERT INTO leads (id, company_id, contact_id, stage, source, currency) VALUES ($1,$2,$3,'new','whatsapp','AED')`, lead, e.a.id, dup)

	thKeep, thDup, thDup2 := uuid.New(), uuid.New(), uuid.New()
	must(t, e.pool, `INSERT INTO whatsapp_threads (id, company_id, contact_id, wa_account_id, message_count, last_message_at)
		VALUES ($1,$2,$3,'PN1',1, now() - interval '1 hour')`, thKeep, e.a.id, keep)
	must(t, e.pool, `INSERT INTO whatsapp_threads (id, company_id, contact_id, wa_account_id, message_count, last_message_at)
		VALUES ($1,$2,$3,'PN1',2, now())`, thDup, e.a.id, dup) // same account -> folded into thKeep
	must(t, e.pool, `INSERT INTO whatsapp_threads (id, company_id, contact_id, wa_account_id, message_count)
		VALUES ($1,$2,$3,'PN2',1)`, thDup2, e.a.id, dup) // different account -> re-parented
	must(t, e.pool, `INSERT INTO whatsapp_messages (thread_id, direction, body, wa_message_id) VALUES
		($1,'inbound','a','m-keep-1'), ($2,'inbound','b','m-dup-1'), ($2,'inbound','c','m-dup-2'), ($3,'inbound','d','m-dup-3')`, thKeep, thDup, thDup2)
	must(t, e.pool, `INSERT INTO whatsapp_outbound (thread_id, to_number, message_body, status, company_id) VALUES ($1,'971501110001','hi','sent',$2)`, thDup, e.a.id)

	// Formatting-only differences, plus a number we cannot interpret, plus another company.
	must(t, e.pool, `INSERT INTO contacts (id, company_id, phone_wa, full_name) VALUES (gen_random_uuid(), $1, '0097150 111 0002', 'Formatted')`, e.a.id)
	must(t, e.pool, `INSERT INTO contacts (id, company_id, phone_wa, full_name) VALUES (gen_random_uuid(), $1, '0501110003', 'National')`, e.a.id)
	must(t, e.pool, `INSERT INTO contacts (id, company_id, phone_wa, full_name) VALUES (gen_random_uuid(), $1, '971501110001', 'Other company')`, e.b.id)

	must(t, e.pool, block)

	count := func(q string, args ...any) int {
		var n int
		if err := e.pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(`SELECT count(*) FROM contacts WHERE company_id=$1 AND phone_wa='+971501110001'`, e.a.id); n != 1 {
		t.Fatalf("company A has %d contacts for the number, want 1", n)
	}
	var name, email string
	var score int
	var assigned *uuid.UUID
	if err := e.pool.QueryRow(ctx, `SELECT full_name, COALESCE(email,''), lead_score, assigned_to FROM contacts WHERE id=$1`, keep).Scan(&name, &email, &score, &assigned); err != nil {
		t.Fatalf("survivor missing: %v", err)
	}
	if name != "Khalid (VIP)" || email != "k@example.com" || score != 40 || assigned == nil {
		t.Errorf("survivor = name %q email %q score %d assigned %v; want agent's name kept and the duplicate's extra data merged", name, email, score, assigned)
	}
	if n := count(`SELECT count(*) FROM contacts WHERE id=$1`, dup); n != 0 {
		t.Error("duplicate contact still exists")
	}
	if n := count(`SELECT count(*) FROM leads WHERE id=$1 AND contact_id=$2`, lead, keep); n != 1 {
		t.Error("lead was not re-pointed to the surviving contact")
	}
	if n := count(`SELECT count(*) FROM whatsapp_threads WHERE contact_id=$1`, keep); n != 2 {
		t.Errorf("survivor has %d threads, want 2 (PN1 merged, PN2 re-parented)", n)
	}
	if n := count(`SELECT count(*) FROM whatsapp_messages WHERE thread_id=$1`, thKeep); n != 3 {
		t.Errorf("merged PN1 thread has %d messages, want 3", n)
	}
	if n := count(`SELECT message_count FROM whatsapp_threads WHERE id=$1`, thKeep); n != 3 {
		t.Errorf("merged thread message_count = %d, want 3", n)
	}
	if n := count(`SELECT count(*) FROM whatsapp_outbound WHERE thread_id=$1`, thKeep); n != 1 {
		t.Error("outbound message was not moved to the merged thread")
	}
	if n := count(`SELECT count(*) FROM contacts WHERE company_id=$1 AND phone_wa='+971501110002'`, e.a.id); n != 1 {
		t.Error("formatted number was not normalised")
	}
	if n := count(`SELECT count(*) FROM contacts WHERE company_id=$1 AND phone_wa='0501110003'`, e.a.id); n != 1 {
		t.Error("national-format number must be left untouched")
	}
	if n := count(`SELECT count(*) FROM contacts WHERE company_id=$1 AND phone_wa='+971501110001'`, e.b.id); n != 1 {
		t.Error("other company's contact must be normalised but never merged into company A")
	}

	// Idempotent: a second run changes nothing.
	must(t, e.pool, block)
	if n := count(`SELECT count(*) FROM contacts WHERE company_id=$1`, e.a.id); n != 3 {
		t.Errorf("second run changed the data: %d contacts in company A, want 3", n)
	}
}
