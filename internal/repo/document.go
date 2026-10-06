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

type DocumentRepo struct {
	db *pgxpool.Pool
}

func NewDocumentRepo(db *pgxpool.Pool) *DocumentRepo {
	return &DocumentRepo{db: db}
}

func (r *DocumentRepo) CreateTemplate(ctx context.Context, t *domain.DocumentTemplate) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	t.CompanyID = cid // never trust a company id supplied by the client
	const q = `
		INSERT INTO document_templates (id, company_id, template_name, document_type, template_content, language, signature_required, signature_fields, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at
	`
	sigFields, _ := json.Marshal(t.SignatureFields)
	t.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		t.ID, t.CompanyID, t.TemplateName, t.DocumentType, t.TemplateContent,
		t.Language, t.SignatureRequired, sigFields, t.CreatedBy,
	).Scan(&t.CreatedAt)
}

func (r *DocumentRepo) GetTemplate(ctx context.Context, id uuid.UUID) (*domain.DocumentTemplate, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT id, company_id, template_name, document_type, template_content, language,
		       signature_required, signature_fields, created_by, created_at
		FROM document_templates WHERE id = $1 AND company_id = $2
	`
	t := &domain.DocumentTemplate{}
	var sigFields []byte
	err = r.db.QueryRow(ctx, q, id, cid).Scan(
		&t.ID, &t.CompanyID, &t.TemplateName, &t.DocumentType, &t.TemplateContent,
		&t.Language, &t.SignatureRequired, &sigFields, &t.CreatedBy, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	if sigFields != nil {
		json.Unmarshal(sigFields, &t.SignatureFields)
	}
	return t, nil
}

func (r *DocumentRepo) ListTemplates(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.DocumentTemplate], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM document_templates WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count templates: %w", err)
	}

	const q = `
		SELECT id, company_id, template_name, document_type, template_content, language,
		       signature_required, signature_fields, created_by, created_at
		FROM document_templates WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []domain.DocumentTemplate
	for rows.Next() {
		var t domain.DocumentTemplate
		var sigFields []byte
		if err := rows.Scan(
			&t.ID, &t.CompanyID, &t.TemplateName, &t.DocumentType, &t.TemplateContent,
			&t.Language, &t.SignatureRequired, &sigFields, &t.CreatedBy, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		if sigFields != nil {
			json.Unmarshal(sigFields, &t.SignatureFields)
		}
		templates = append(templates, t)
	}
	return &domain.PaginatedResult[domain.DocumentTemplate]{
		Data: templates, Total: total, Page: page, Limit: limit,
	}, nil
}

func (r *DocumentRepo) UpdateTemplate(ctx context.Context, id uuid.UUID, name string, content string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE document_templates SET template_name=$1, template_content=$2 WHERE id=$3 AND company_id=$4`,
		name, content, id, cid,
	)
	return err
}

func (r *DocumentRepo) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM document_templates WHERE id=$1 AND company_id=$2`, id, cid)
	return err
}

func (r *DocumentRepo) CreateDocument(ctx context.Context, d *domain.Document) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	d.CompanyID = cid // never trust a company id supplied by the client
	if d.OriginalTemplateID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM document_templates WHERE id = $1 AND company_id = $2)`, *d.OriginalTemplateID, cid).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrForeignReference
		}
	}
	const q = `
		INSERT INTO documents (id, company_id, document_type, original_template_id, related_entity_type, related_entity_id,
		                       document_title, file_url, file_size_bytes, content_hash, signature_status, created_by, data_classification, retention_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING created_at, updated_at
	`
	d.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		d.ID, d.CompanyID, d.DocumentType, d.OriginalTemplateID, d.RelatedEntityType, d.RelatedEntityID,
		d.DocumentTitle, d.FileURL, d.FileSizeBytes, d.ContentHash, d.SignatureStatus, d.CreatedBy, d.DataClassification, d.RetentionUntil,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
}

func (r *DocumentRepo) GetDocument(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	return r.getDocument(ctx, id, &cid)
}

// GetDocumentForSigning returns a document without company scoping. It is ONLY for the
// public e-signature flow, where possession of the signature UUID is the credential and the
// caller has already resolved the document through that signature. Never expose it to
// authenticated endpoints that take a document id from the client.
func (r *DocumentRepo) GetDocumentForSigning(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	return r.getDocument(ctx, id, nil)
}

func (r *DocumentRepo) getDocument(ctx context.Context, id uuid.UUID, companyID *uuid.UUID) (*domain.Document, error) {
	const q = `
		SELECT id, company_id, document_type, original_template_id, related_entity_type, related_entity_id,
		       document_title, file_url, file_size_bytes, content_hash, signature_status, created_by, created_at, updated_at, data_classification, retention_until, deleted_at
		FROM documents WHERE id = $1 AND ($2::uuid IS NULL OR company_id = $2)
	`
	d := &domain.Document{}
	err := r.db.QueryRow(ctx, q, id, companyID).Scan(
		&d.ID, &d.CompanyID, &d.DocumentType, &d.OriginalTemplateID, &d.RelatedEntityType, &d.RelatedEntityID,
		&d.DocumentTitle, &d.FileURL, &d.FileSizeBytes, &d.ContentHash, &d.SignatureStatus, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.DataClassification, &d.RetentionUntil, &d.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return d, nil
}

func (r *DocumentRepo) ListDocumentsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.Document, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT id, company_id, document_type, original_template_id, related_entity_type, related_entity_id,
		       document_title, file_url, file_size_bytes, content_hash, signature_status, created_by, created_at, updated_at, data_classification, retention_until, deleted_at
		FROM documents WHERE related_entity_type = $1 AND related_entity_id = $2 AND company_id = $3 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, q, entityType, entityID, cid)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var docs []domain.Document
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(
			&d.ID, &d.CompanyID, &d.DocumentType, &d.OriginalTemplateID, &d.RelatedEntityType, &d.RelatedEntityID,
			&d.DocumentTitle, &d.FileURL, &d.FileSizeBytes, &d.ContentHash, &d.SignatureStatus, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.DataClassification, &d.RetentionUntil, &d.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *DocumentRepo) UpdateSignatureStatus(ctx context.Context, id uuid.UUID, status domain.SignatureStatus) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE documents SET signature_status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3`,
		status, id, cid,
	)
	return err
}

func (r *DocumentRepo) CreateSignature(ctx context.Context, s *domain.DocumentSignature) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// The document must belong to the caller's company.
	const q = `
		INSERT INTO document_signatures (id, document_id, signer_name, signer_email, signature_field_name, signature_status, signed_at, signature_image_url, ip_address, user_agent, envelope_id)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
		WHERE EXISTS (SELECT 1 FROM documents WHERE id = $2 AND company_id = $12)
		RETURNING created_at
	`
	s.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		s.ID, s.DocumentID, s.SignerName, s.SignerEmail, s.SignatureFieldName, s.SignatureStatus, s.SignedAt, s.SignatureImageURL, s.IPAddress, s.UserAgent, nullIfEmpty(s.EnvelopeID), cid,
	).Scan(&s.CreatedAt)
}

func (r *DocumentRepo) GetSignatures(ctx context.Context, documentID uuid.UUID) ([]domain.DocumentSignature, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT s.id, s.document_id, s.signer_name, s.signer_email, s.signature_field_name, s.signature_status, s.signed_at, s.signature_image_url, s.ip_address, s.user_agent, s.envelope_id, s.created_at
		FROM document_signatures s JOIN documents d ON d.id = s.document_id
		WHERE s.document_id = $1 AND d.company_id = $2
		ORDER BY s.created_at ASC
	`
	rows, err := r.db.Query(ctx, q, documentID, cid)
	if err != nil {
		return nil, fmt.Errorf("get signatures: %w", err)
	}
	defer rows.Close()

	var sigs []domain.DocumentSignature
	for rows.Next() {
		var s domain.DocumentSignature
		if err := rows.Scan(
			&s.ID, &s.DocumentID, &s.SignerName, &s.SignerEmail, &s.SignatureFieldName, &s.SignatureStatus, &s.SignedAt, &s.SignatureImageURL, &s.IPAddress, &s.UserAgent, &s.EnvelopeID, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan signature: %w", err)
		}
		sigs = append(sigs, s)
	}
	return sigs, nil
}

// GetSignatureByToken looks a signature up by its UUID without company scoping. It backs the
// public e-signature page, where the signature UUID itself is the access credential.
func (r *DocumentRepo) GetSignatureByToken(ctx context.Context, sigID uuid.UUID) (*domain.DocumentSignature, error) {
	const q = `
		SELECT id, document_id, signer_name, signer_email, signature_field_name, signature_status, signed_at, signature_image_url, ip_address, user_agent, envelope_id, created_at
		FROM document_signatures WHERE id = $1
	`
	s := &domain.DocumentSignature{}
	err := r.db.QueryRow(ctx, q, sigID).Scan(
		&s.ID, &s.DocumentID, &s.SignerName, &s.SignerEmail, &s.SignatureFieldName, &s.SignatureStatus,
		&s.SignedAt, &s.SignatureImageURL, &s.IPAddress, &s.UserAgent, &s.EnvelopeID, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get signature: %w", err)
	}
	return s, nil
}

// MarkSigned records a signature for an authenticated user; the document must be in their company.
func (r *DocumentRepo) MarkSigned(ctx context.Context, sigID uuid.UUID, signedAt time.Time) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE document_signatures s SET signature_status=$1, signed_at=$2
		 WHERE s.id=$3 AND EXISTS (SELECT 1 FROM documents d WHERE d.id = s.document_id AND d.company_id = $4)`,
		domain.SignatureSigned, signedAt, sigID, cid,
	)
	return err
}

// ErrNotSignable is returned by SignByToken when the signature does not exist, was already
// signed, is handled by DocuSign, or belongs to a deleted document.
var ErrNotSignable = errors.New("signature is not signable")

// SignByToken records a signature from the public e-signature page, where the signature UUID
// is the credential. Deliberately not company scoped. Only a pending, non-DocuSign signature
// on a live document can be signed, exactly once; the signer's own IP and user agent replace
// the ones captured when the request was created (the audit trail must describe the signer).
// The document is marked signed once every one of its signatures is.
func (r *DocumentRepo) SignByToken(ctx context.Context, sigID uuid.UUID, signedAt time.Time, ip, userAgent string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var docID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE document_signatures s
		SET signature_status = $1, signed_at = $2, ip_address = $3, user_agent = $4
		WHERE s.id = $5
		  AND s.signature_status = $6
		  AND (s.envelope_id IS NULL OR s.envelope_id = '')
		  AND EXISTS (SELECT 1 FROM documents d WHERE d.id = s.document_id AND d.deleted_at IS NULL)
		RETURNING s.document_id`,
		domain.SignatureSigned, signedAt, ip, userAgent, sigID, domain.SignaturePending,
	).Scan(&docID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotSignable
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE documents SET signature_status = $1, updated_at = NOW()
		WHERE id = $2
		  AND NOT EXISTS (
		        SELECT 1 FROM document_signatures o
		        WHERE o.document_id = $2 AND o.signature_status IS DISTINCT FROM $1)`,
		domain.SignatureSigned, docID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// MarkSignedByEnvelope is called by the DocuSign webhook (authenticated by its HMAC secret),
// which has no company context; the envelope id identifies the signature.
func (r *DocumentRepo) MarkSignedByEnvelope(ctx context.Context, envelopeID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE document_signatures SET signature_status=$1, signed_at=NOW() WHERE envelope_id=$2`,
		domain.SignatureSigned, envelopeID,
	)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE documents SET signature_status=$1, updated_at=NOW()
		WHERE id = (SELECT document_id FROM document_signatures WHERE envelope_id=$2 LIMIT 1)
	`, domain.SignatureSigned, envelopeID)
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *DocumentRepo) SoftDeleteDocument(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE documents SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND company_id=$2`,
		id, cid,
	)
	return err
}
