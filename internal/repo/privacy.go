package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// PrivacyRepo stores consent and implements data-subject requests (export, erasure) and
// retention clean-up for UAE PDPL.
type PrivacyRepo struct {
	db *pgxpool.Pool
}

func NewPrivacyRepo(db *pgxpool.Pool) *PrivacyRepo { return &PrivacyRepo{db: db} }

// ErrContactErased is returned for writes to a contact that has been erased.
var ErrContactErased = errors.New("contact has been erased")

// RecordConsent appends a consent change. The contact must belong to the caller's company.
func (r *PrivacyRepo) RecordConsent(ctx context.Context, c *domain.Consent) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if !domain.ValidConsentChannel(c.Channel) || (c.Status != domain.ConsentGranted && c.Status != domain.ConsentWithdrawn) {
		return fmt.Errorf("invalid consent %q/%q", c.Channel, c.Status)
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO contact_consents (company_id, contact_id, channel, status, source, captured_by, api_key_id, note)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8
		 WHERE EXISTS (SELECT 1 FROM contacts WHERE id = $2 AND company_id = $1)
		RETURNING id, created_at`,
		cid, c.ContactID, c.Channel, c.Status, c.Source, c.CapturedBy, c.APIKeyID, c.Note,
	).Scan(&c.ID, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForeignReference
	}
	return err
}

// ConsentHistory returns every consent change for a contact, newest first.
func (r *PrivacyRepo) ConsentHistory(ctx context.Context, contactID uuid.UUID) ([]domain.Consent, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, contact_id, channel, status, source, captured_by, api_key_id, note, created_at
		  FROM contact_consents WHERE contact_id = $1 AND company_id = $2
		 ORDER BY created_at DESC, id DESC`, contactID, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanConsent)
}

func scanConsent(row pgx.CollectableRow) (domain.Consent, error) {
	var c domain.Consent
	err := row.Scan(&c.ID, &c.ContactID, &c.Channel, &c.Status, &c.Source, &c.CapturedBy, &c.APIKeyID, &c.Note, &c.CreatedAt)
	return c, err
}

// CurrentConsent returns the latest consent per channel ("granted"/"withdrawn"); a
// channel with no record is absent from the map.
func (r *PrivacyRepo) CurrentConsent(ctx context.Context, contactID uuid.UUID) (map[string]domain.Consent, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (channel) id, contact_id, channel, status, source, captured_by, api_key_id, note, created_at
		  FROM contact_consents WHERE contact_id = $1 AND company_id = $2
		 ORDER BY channel, created_at DESC, id DESC`, contactID, cid)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, scanConsent)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Consent, len(list))
	for _, c := range list {
		out[c.Channel] = c
	}
	return out, nil
}

// MayContact reports whether the contact may be messaged on channel: false when their
// latest choice for it is "withdrawn" or the contact has been erased. A contact who never
// expressed a choice may be contacted (e.g. replying to their enquiry); bulk marketing
// should require an explicit grant instead (see RequireGranted).
func (r *PrivacyRepo) MayContact(ctx context.Context, contactID uuid.UUID, channel string) (bool, error) {
	status, erased, err := r.channelStatus(ctx, contactID, channel)
	if err != nil {
		return false, err
	}
	return !erased && status != domain.ConsentWithdrawn, nil
}

// HasGranted reports whether the contact's latest choice for channel is "granted".
func (r *PrivacyRepo) HasGranted(ctx context.Context, contactID uuid.UUID, channel string) (bool, error) {
	status, erased, err := r.channelStatus(ctx, contactID, channel)
	if err != nil {
		return false, err
	}
	return !erased && status == domain.ConsentGranted, nil
}

func (r *PrivacyRepo) channelStatus(ctx context.Context, contactID uuid.UUID, channel string) (string, bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", false, err
	}
	var status *string
	var erased bool
	err = r.db.QueryRow(ctx, `
		SELECT (SELECT status FROM contact_consents
		         WHERE contact_id = c.id AND channel = $3 ORDER BY created_at DESC, id DESC LIMIT 1),
		       c.erased_at IS NOT NULL
		  FROM contacts c WHERE c.id = $1 AND c.company_id = $2`, contactID, cid, channel).Scan(&status, &erased)
	if err != nil {
		return "", false, err
	}
	if status == nil {
		return "", erased, nil
	}
	return *status, erased, nil
}

// EmailOptedOut reports whether any contact of the company with this email
// address has withdrawn email consent or been erased.
func (r *PrivacyRepo) EmailOptedOut(ctx context.Context, email string) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	var out bool
	err = r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM contacts c
		   WHERE c.company_id = $1 AND lower(c.email) = lower($2)
		     AND (c.erased_at IS NOT NULL OR
		          (SELECT status FROM contact_consents
		            WHERE contact_id = c.id AND channel = 'email' ORDER BY created_at DESC, id DESC LIMIT 1) = 'withdrawn'))`,
		cid, email).Scan(&out)
	return out, err
}

// PhoneOptedOut reports whether a contact of the company with this WhatsApp or secondary
// phone number withdrew consent for the channel, or was erased. Used for people who are
// not contacts themselves, such as tenants.
func (r *PrivacyRepo) PhoneOptedOut(ctx context.Context, phone, channel string) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	var out bool
	err = r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM contacts c
		   WHERE c.company_id = $1 AND (c.phone_wa = $2 OR $2 = ANY(c.secondary_phones))
		     AND (c.erased_at IS NOT NULL OR
		          (SELECT status FROM contact_consents
		            WHERE contact_id = c.id AND channel = $3 ORDER BY created_at DESC, id DESC LIMIT 1) = 'withdrawn'))`,
		cid, phone, channel).Scan(&out)
	return out, err
}

// exportQueries lists what a data-subject export contains. Each query selects rows for
// ($1 company, $2 contact) and is returned as JSON.
var exportQueries = []struct{ key, q string }{
	{"contact", `SELECT * FROM contacts WHERE company_id = $1 AND id = $2`},
	{"consents", `SELECT channel, status, source, note, created_at FROM contact_consents WHERE company_id = $1 AND contact_id = $2 ORDER BY created_at`},
	{"leads", `SELECT * FROM leads WHERE company_id = $1 AND contact_id = $2`},
	{"deals", `SELECT d.* FROM deals d JOIN leads l ON l.id = d.lead_id WHERE d.company_id = $1 AND l.contact_id = $2`},
	{"offers", `SELECT * FROM offers WHERE company_id = $1 AND contact_id = $2`},
	{"kyc_documents", `SELECT doc_type, number, issuing_country, issue_date, expiry_date, file_name, created_at FROM kyc_documents WHERE company_id = $1 AND contact_id = $2`},
	{"viewings", `SELECT * FROM viewings WHERE company_id = $1 AND contact_id = $2`},
	{"whatsapp_threads", `SELECT id, thread_status, last_message_at, message_count, ai_summary, created_at FROM whatsapp_threads WHERE company_id = $1 AND contact_id = $2`},
	{"whatsapp_messages", `SELECT m.id, m.thread_id, m.direction, m.body, m.media_mime, m.media_filename, m.sent_at
	                         FROM whatsapp_messages m JOIN whatsapp_threads t ON t.id = m.thread_id
	                        WHERE t.company_id = $1 AND t.contact_id = $2 ORDER BY m.sent_at`},
	{"communications", `SELECT * FROM communication_history WHERE company_id = $1 AND contact_id = $2 ORDER BY created_at`},
	{"emails", `SELECT e.id, e.to_email, e.subject, e.body, e.status, e.sent_at, e.created_at FROM email_history e
	             JOIN contacts c ON c.company_id = e.company_id AND lower(c.email) = lower(e.to_email) AND c.email <> ''
	            WHERE e.company_id = $1 AND c.id = $2 ORDER BY e.created_at`},
	{"invoices", `SELECT v.invoice_no, v.kind, v.currency, v.subtotal, v.vat_amount, v.total, v.status, v.issued_at,
	                     v.customer_name, v.customer_address, v.customer_trn
	                FROM vat_invoices v JOIN deals d ON d.id = v.deal_id JOIN leads l ON l.id = d.lead_id
	               WHERE v.company_id = $1 AND l.contact_id = $2 ORDER BY v.issued_at`},
}

// Export gathers everything held about a contact, keyed by kind, for a data-subject
// access request.
func (r *PrivacyRepo) Export(ctx context.Context, contactID uuid.UUID) (map[string]json.RawMessage, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE id = $1 AND company_id = $2)`,
		contactID, cid).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, pgx.ErrNoRows
	}
	out := make(map[string]json.RawMessage, len(exportQueries))
	for _, eq := range exportQueries {
		var raw []byte
		if err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(t), '[]'::json) FROM (`+eq.q+`) t`, cid, contactID).Scan(&raw); err != nil {
			return nil, fmt.Errorf("export %s: %w", eq.key, err)
		}
		out[eq.key] = raw
	}
	return out, nil
}

// Erase anonymises a contact: personal data in the contact, its conversations, notes and
// messages is removed, consent is recorded as withdrawn on every channel, and the
// contact is marked erased. Records the law requires the business to keep — issued
// invoices with the customer details printed on them — are left unchanged. It returns
// the stored media paths of the contact's WhatsApp attachments so the caller can delete
// the files after the transaction commits.
func (r *PrivacyRepo) Erase(ctx context.Context, contactID uuid.UUID, by uuid.UUID) ([]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var email string
	var erasedAt *time.Time
	err = tx.QueryRow(ctx, `SELECT COALESCE(email,''), erased_at FROM contacts WHERE id = $1 AND company_id = $2 FOR UPDATE`,
		contactID, cid).Scan(&email, &erasedAt)
	if err != nil {
		return nil, err
	}
	if erasedAt != nil {
		return nil, ErrContactErased
	}

	rows, err := tx.Query(ctx, `
		SELECT m.media_path FROM whatsapp_messages m JOIN whatsapp_threads t ON t.id = m.thread_id
		 WHERE t.company_id = $1 AND t.contact_id = $2 AND m.media_path IS NOT NULL AND m.media_path <> ''`, cid, contactID)
	if err != nil {
		return nil, err
	}
	paths, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	stmts := []string{
		// A unique placeholder number keeps (company_id, phone_wa) unique.
		`UPDATE contacts SET full_name = 'Erased contact', email = NULL,
		        phone_wa = 'erased:' || left(replace(id::text, '-', ''), 12),
		        lead_score = 0, assigned_to = NULL, erased_at = NOW(), updated_at = NOW()
		  WHERE id = $2 AND company_id = $1`,
		`UPDATE whatsapp_messages m SET body = '[erased]', media_url = NULL, wa_media_id = NULL, media_mime = NULL,
		        media_path = NULL, media_size = NULL, media_filename = NULL
		   FROM whatsapp_threads t
		  WHERE t.id = m.thread_id AND t.company_id = $1 AND t.contact_id = $2`,
		`UPDATE whatsapp_outbound o SET to_number = 'erased', message_body = '[erased]', media_url = NULL, metadata = NULL
		   FROM whatsapp_threads t
		  WHERE t.id = o.thread_id AND t.company_id = $1 AND t.contact_id = $2`,
		`UPDATE whatsapp_threads SET ai_summary = NULL, embedding = NULL WHERE company_id = $1 AND contact_id = $2`,
		`UPDATE leads SET notes = '', closed_reason = NULL WHERE company_id = $1 AND contact_id = $2`,
		`UPDATE communication_history SET body = '[erased]', from_identifier = NULL, to_identifier = NULL, metadata = NULL
		  WHERE company_id = $1 AND contact_id = $2`,
		`UPDATE viewings SET notes = '' WHERE company_id = $1 AND contact_id = $2`,
		`UPDATE offers SET notes = '', terms = '' WHERE company_id = $1 AND contact_id = $2`,
		`INSERT INTO contact_consents (company_id, contact_id, channel, status, source, captured_by, note)
		 SELECT $1, $2, ch, 'withdrawn', 'erasure', $3, 'data-subject erasure'
		   FROM unnest(ARRAY['whatsapp','email','sms','phone']) AS ch`,
	}
	for _, q := range stmts {
		args := []any{cid, contactID}
		if containsParam3(q) {
			args = append(args, by)
		}
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return nil, fmt.Errorf("erase: %w", err)
		}
	}
	if email != "" {
		if _, err := tx.Exec(ctx, `UPDATE email_history SET to_email = 'erased', body = '[erased]', html_body = '[erased]', metadata = NULL
		                            WHERE company_id = $1 AND lower(to_email) = lower($2)`, cid, email); err != nil {
			return nil, fmt.Errorf("erase emails: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return paths, nil
}

func containsParam3(q string) bool {
	for i := 0; i+1 < len(q); i++ {
		if q[i] == '$' && q[i+1] == '3' {
			return true
		}
	}
	return false
}

// Retention clean-up: rows whose retention_until has passed.

// ExpiredBankStatements returns company bank statements past retention that still exist,
// with their stored file paths.
func (r *PrivacyRepo) ExpiredBankStatements(ctx context.Context, today time.Time) ([]RetainedFile, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(storage_key, '') FROM bank_statements
		 WHERE company_id = $1 AND deleted_at IS NULL AND retention_until IS NOT NULL AND retention_until < $2`, cid, today)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (RetainedFile, error) {
		var f RetainedFile
		err := row.Scan(&f.ID, &f.Path)
		return f, err
	})
}

// RetainedFile is a stored record whose file must be removed with it; Path is the
// mediastore key.
type RetainedFile struct {
	ID   uuid.UUID
	Path string
}

// MarkBankStatementDeleted soft-deletes a statement after its file was removed.
func (r *PrivacyRepo) MarkBankStatementDeleted(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE bank_statements SET deleted_at = NOW(), storage_key = NULL
		WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL`, id, cid)
	return err
}

// PurgeExpiredRecords soft-deletes documents and payment confirmations past retention
// and returns how many were removed.
func (r *PrivacyRepo) PurgeExpiredRecords(ctx context.Context, today time.Time) (int64, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, q := range []string{
		`UPDATE documents SET deleted_at = NOW() WHERE company_id = $1 AND deleted_at IS NULL
		   AND retention_until IS NOT NULL AND retention_until < $2`,
		`UPDATE payment_confirmations SET deleted_at = NOW() WHERE company_id = $1 AND deleted_at IS NULL
		   AND retention_until IS NOT NULL AND retention_until < $2`,
	} {
		tag, err := r.db.Exec(ctx, q, cid, today)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
	}
	return total, nil
}
