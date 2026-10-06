package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestSignByToken(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	r := NewDocumentRepo(e.pool)

	docID := uuid.New()
	must(t, e.pool, `INSERT INTO documents (id, company_id, document_title, signature_status, created_by)
		VALUES ($1, $2, 'Lease', 'pending', $3)`, docID, e.a.id, e.a.user)
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `DELETE FROM document_signatures WHERE document_id = $1`, docID)
		_, _ = e.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, docID)
	})

	newSig := func(envelope string) uuid.UUID {
		id := uuid.New()
		must(t, e.pool, `INSERT INTO document_signatures (id, document_id, signer_name, signer_email, signature_status, ip_address, user_agent, envelope_id)
			VALUES ($1, $2, 'S', 's@example.com', 'pending', '10.0.0.1', 'agent-ua', NULLIF($3, ''))`, id, docID, envelope)
		return id
	}
	docStatus := func() string {
		var s string
		if err := e.pool.QueryRow(ctx, `SELECT signature_status FROM documents WHERE id = $1`, docID).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	first, second, docusign := newSig(""), newSig(""), newSig("env-123")

	// unknown id
	if err := r.SignByToken(ctx, uuid.New(), time.Now(), "1.2.3.4", "ua"); !errors.Is(err, ErrNotSignable) {
		t.Fatalf("unknown signature: got %v, want ErrNotSignable", err)
	}
	// DocuSign-managed signatures cannot be completed through the public link
	if err := r.SignByToken(ctx, docusign, time.Now(), "1.2.3.4", "ua"); !errors.Is(err, ErrNotSignable) {
		t.Fatalf("docusign signature: got %v, want ErrNotSignable", err)
	}

	// first signer signs: document stays pending while others are outstanding
	if err := r.SignByToken(ctx, first, time.Now(), "203.0.113.9", "signer-ua"); err != nil {
		t.Fatalf("sign: %v", err)
	}
	var ip, ua, status string
	if err := e.pool.QueryRow(ctx, `SELECT ip_address, user_agent, signature_status FROM document_signatures WHERE id = $1`, first).Scan(&ip, &ua, &status); err != nil {
		t.Fatal(err)
	}
	if ip != "203.0.113.9" || ua != "signer-ua" || status != string(domain.SignatureSigned) {
		t.Fatalf("signature row = %q %q %q; want signer's ip/ua and signed", ip, ua, status)
	}
	if got := docStatus(); got != string(domain.SignaturePending) {
		t.Fatalf("document status = %q after first of three signatures, want pending", got)
	}

	// signing twice is refused and does not rewrite the audit trail
	if err := r.SignByToken(ctx, first, time.Now(), "198.51.100.1", "other"); !errors.Is(err, ErrNotSignable) {
		t.Fatalf("re-sign: got %v, want ErrNotSignable", err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT ip_address FROM document_signatures WHERE id = $1`, first).Scan(&ip); err != nil || ip != "203.0.113.9" {
		t.Fatalf("re-sign altered ip: %q %v", ip, err)
	}

	// remaining non-DocuSign signer signs; DocuSign one is still pending so doc is not complete
	if err := r.SignByToken(ctx, second, time.Now(), "203.0.113.10", "ua2"); err != nil {
		t.Fatalf("sign second: %v", err)
	}
	if got := docStatus(); got != string(domain.SignaturePending) {
		t.Fatalf("document status = %q with DocuSign signature outstanding, want pending", got)
	}
	must(t, e.pool, `UPDATE document_signatures SET signature_status = 'signed' WHERE id = $1`, docusign)
	third := newSig("")
	if err := r.SignByToken(ctx, third, time.Now(), "203.0.113.11", "ua3"); err != nil {
		t.Fatalf("sign third: %v", err)
	}
	if got := docStatus(); got != string(domain.SignatureSigned) {
		t.Fatalf("document status = %q after every signature signed, want signed", got)
	}

	// deleted documents cannot be signed
	late := newSig("")
	must(t, e.pool, `UPDATE documents SET deleted_at = NOW() WHERE id = $1`, docID)
	if err := r.SignByToken(ctx, late, time.Now(), "1.2.3.4", "ua"); !errors.Is(err, ErrNotSignable) {
		t.Fatalf("deleted document: got %v, want ErrNotSignable", err)
	}
}
