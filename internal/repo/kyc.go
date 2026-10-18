package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/kyc"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// KYCRepo stores customer-due-diligence records and the AML review of deals.
type KYCRepo struct {
	db *pgxpool.Pool
}

func NewKYCRepo(db *pgxpool.Pool) *KYCRepo { return &KYCRepo{db: db} }

// KYCProfile is a contact's due-diligence record.
type KYCProfile struct {
	ContactID       uuid.UUID     `json:"contact_id"`
	FullName        string        `json:"full_name"`
	Nationality     string        `json:"nationality"`
	DateOfBirth     *time.Time    `json:"date_of_birth"`
	Risk            string        `json:"risk"` // '' | low | medium | high
	RiskReason      string        `json:"risk_reason"`
	IsPEP           bool          `json:"is_pep"`
	Screening       string        `json:"screening"`
	ScreeningSource string        `json:"screening_source"`
	ScreenedAt      *time.Time    `json:"screened_at"`
	Notes           string        `json:"notes"`
	ReviewedBy      *uuid.UUID    `json:"reviewed_by"`
	ReviewedAt      *time.Time    `json:"reviewed_at"`
	Documents       []KYCDocument `json:"documents"`
	Missing         []string      `json:"missing"` // empty when the identity record is complete
}

// KYCDocument is an identity document on file.
type KYCDocument struct {
	ID             uuid.UUID  `json:"id"`
	ContactID      uuid.UUID  `json:"contact_id"`
	DocType        string     `json:"doc_type"`
	Number         string     `json:"number"`
	IssuingCountry string     `json:"issuing_country"`
	IssueDate      *time.Time `json:"issue_date"`
	ExpiryDate     *time.Time `json:"expiry_date"`
	HasFile        bool       `json:"has_file"`
	FileName       string     `json:"file_name"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Profile returns a contact's KYC record and documents, with what is missing.
func (r *KYCRepo) Profile(ctx context.Context, contactID uuid.UUID) (*KYCProfile, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	p := &KYCProfile{ContactID: contactID}
	err = r.db.QueryRow(ctx, `
		SELECT full_name, TRIM(nationality), date_of_birth, kyc_risk, kyc_risk_reason, kyc_is_pep, kyc_screening,
		       kyc_screening_source, kyc_screened_at, kyc_notes, kyc_reviewed_by, kyc_reviewed_at
		  FROM contacts WHERE id = $1 AND company_id = $2`, contactID, cid).Scan(
		&p.FullName, &p.Nationality, &p.DateOfBirth, &p.Risk, &p.RiskReason, &p.IsPEP, &p.Screening,
		&p.ScreeningSource, &p.ScreenedAt, &p.Notes, &p.ReviewedBy, &p.ReviewedAt)
	if err != nil {
		return nil, err
	}
	if p.Documents, err = r.Documents(ctx, contactID); err != nil {
		return nil, err
	}
	p.Missing = kyc.Check(docsOf(p.Documents), false, time.Now())
	if p.Missing == nil {
		p.Missing = []string{}
	}
	return p, nil
}

func docsOf(list []KYCDocument) []kyc.Doc {
	out := make([]kyc.Doc, len(list))
	for i, d := range list {
		out[i] = kyc.Doc{Type: d.DocType, Expiry: d.ExpiryDate}
	}
	return out
}

// UpdateProfile records the risk rating, PEP flag, screening result and notes, and who
// reviewed them.
func (r *KYCRepo) UpdateProfile(ctx context.Context, p *KYCProfile, reviewer uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE contacts SET date_of_birth = $1, kyc_risk = $2, kyc_risk_reason = $3, kyc_is_pep = $4,
		       kyc_screened_at = CASE WHEN kyc_screening IS DISTINCT FROM $5 OR kyc_screening_source IS DISTINCT FROM $6
		                              THEN NOW() ELSE kyc_screened_at END,
		       kyc_screening = $5, kyc_screening_source = $6, kyc_notes = $7,
		       kyc_reviewed_by = $8, kyc_reviewed_at = NOW(), updated_at = NOW()
		 WHERE id = $9 AND company_id = $10 AND erased_at IS NULL`,
		p.DateOfBirth, p.Risk, p.RiskReason, p.IsPEP, p.Screening, p.ScreeningSource, p.Notes, reviewer, p.ContactID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// RecordScreening stores a screening provider's result.
func (r *KYCRepo) RecordScreening(ctx context.Context, contactID uuid.UUID, res kyc.ScreeningResult, provider string, by uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	note := strings.TrimSpace(fmt.Sprintf("[%s %s] %s %s", time.Now().UTC().Format("2006-01-02"), provider, res.Reference, res.Details))
	tag, err := r.db.Exec(ctx, `
		UPDATE contacts SET kyc_screening = $1, kyc_screening_source = $2, kyc_screened_at = NOW(),
		       kyc_notes = CASE WHEN kyc_notes = '' THEN $3 ELSE kyc_notes || E'\n' || $3 END,
		       kyc_reviewed_by = $4, kyc_reviewed_at = NOW(), updated_at = NOW()
		 WHERE id = $5 AND company_id = $6 AND erased_at IS NULL`, res.Status, provider, note, by, contactID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

const kycDocCols = `id, contact_id, doc_type, number, TRIM(issuing_country), issue_date, expiry_date, storage_key <> '', file_name, created_at`

func scanKYCDoc(row pgx.Row, d *KYCDocument) error {
	return row.Scan(&d.ID, &d.ContactID, &d.DocType, &d.Number, &d.IssuingCountry, &d.IssueDate, &d.ExpiryDate, &d.HasFile, &d.FileName, &d.CreatedAt)
}

// Documents lists a contact's identity documents, latest expiry first.
func (r *KYCRepo) Documents(ctx context.Context, contactID uuid.UUID) ([]KYCDocument, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+kycDocCols+` FROM kyc_documents WHERE contact_id = $1 AND company_id = $2
		ORDER BY expiry_date DESC NULLS LAST, created_at DESC`, contactID, cid)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (KYCDocument, error) {
		var d KYCDocument
		err := scanKYCDoc(row, &d)
		return d, err
	})
	if out == nil {
		out = []KYCDocument{}
	}
	return out, err
}

// AddDocument records an identity document for one of the company's contacts.
func (r *KYCRepo) AddDocument(ctx context.Context, d *KYCDocument, storageKey string, by *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	d.ID = uuid.New()
	err = r.db.QueryRow(ctx, `
		INSERT INTO kyc_documents (id, company_id, contact_id, doc_type, number, issuing_country, issue_date, expiry_date,
		                           storage_key, file_name, created_by)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		 WHERE EXISTS (SELECT 1 FROM contacts WHERE id = $3 AND company_id = $2 AND erased_at IS NULL)
		RETURNING created_at`,
		d.ID, cid, d.ContactID, d.DocType, d.Number, d.IssuingCountry, d.IssueDate, d.ExpiryDate, storageKey, d.FileName, by).Scan(&d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForeignReference
	}
	d.HasFile = storageKey != ""
	return err
}

// DocumentFile returns the stored file of a document.
func (r *KYCRepo) DocumentFile(ctx context.Context, id uuid.UUID) (key, name string, contactID uuid.UUID, err error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", "", uuid.Nil, err
	}
	err = r.db.QueryRow(ctx, `SELECT storage_key, file_name, contact_id FROM kyc_documents WHERE id = $1 AND company_id = $2`,
		id, cid).Scan(&key, &name, &contactID)
	return key, name, contactID, err
}

// DeleteDocument removes a document and returns its stored file key.
func (r *KYCRepo) DeleteDocument(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", uuid.Nil, err
	}
	var key string
	var contactID uuid.UUID
	err = r.db.QueryRow(ctx, `DELETE FROM kyc_documents WHERE id = $1 AND company_id = $2 RETURNING storage_key, contact_id`,
		id, cid).Scan(&key, &contactID)
	return key, contactID, err
}

// ── Beneficial owners ─────────────────────────────────────────────────────────

// BeneficialOwner links a person to a corporate client.
type BeneficialOwner struct {
	ContactID    uuid.UUID `json:"contact_id"`
	FullName     string    `json:"full_name"`
	OwnershipPct *float64  `json:"ownership_pct"`
	Role         string    `json:"role"` // owner | controller | signatory
	KYCMissing   []string  `json:"kyc_missing"`
}

// Owners lists an organisation's beneficial owners with their KYC status.
func (r *KYCRepo) Owners(ctx context.Context, orgID uuid.UUID) ([]BeneficialOwner, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT b.contact_id, c.full_name, b.ownership_pct::float8, b.role
		  FROM beneficial_owners b JOIN contacts c ON c.id = b.contact_id
		 WHERE b.organisation_id = $1 AND b.company_id = $2 ORDER BY b.ownership_pct DESC NULLS LAST, c.full_name`, orgID, cid)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (BeneficialOwner, error) {
		var b BeneficialOwner
		err := row.Scan(&b.ContactID, &b.FullName, &b.OwnershipPct, &b.Role)
		return b, err
	})
	if err != nil {
		return nil, err
	}
	for i := range out {
		docs, err := r.Documents(ctx, out[i].ContactID)
		if err != nil {
			return nil, err
		}
		out[i].KYCMissing = kyc.Check(docsOf(docs), false, time.Now())
		if out[i].KYCMissing == nil {
			out[i].KYCMissing = []string{}
		}
	}
	if out == nil {
		out = []BeneficialOwner{}
	}
	return out, nil
}

// SetOwner adds or updates a beneficial owner. Both records must be the company's.
func (r *KYCRepo) SetOwner(ctx context.Context, orgID, contactID uuid.UUID, pct *float64, role string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		INSERT INTO beneficial_owners (company_id, organisation_id, contact_id, ownership_pct, role)
		SELECT $1, $2, $3, $4, $5
		 WHERE EXISTS (SELECT 1 FROM organisations WHERE id = $2 AND company_id = $1)
		   AND EXISTS (SELECT 1 FROM contacts WHERE id = $3 AND company_id = $1)
		ON CONFLICT (organisation_id, contact_id) DO UPDATE SET ownership_pct = EXCLUDED.ownership_pct, role = EXCLUDED.role`,
		cid, orgID, contactID, pct, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrForeignReference
	}
	return nil
}

// RemoveOwner unlinks a beneficial owner.
func (r *KYCRepo) RemoveOwner(ctx context.Context, orgID, contactID uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM beneficial_owners WHERE organisation_id = $1 AND contact_id = $2 AND company_id = $3`,
		orgID, contactID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── Deals ─────────────────────────────────────────────────────────────────────

// DealKYC is the KYC state of a deal's buyer.
type DealKYC struct {
	DealID         uuid.UUID  `json:"deal_id"`
	ContactID      uuid.UUID  `json:"contact_id"`
	ContactName    string     `json:"contact_name"`
	Missing        []string   `json:"missing"`
	OverrideReason string     `json:"override_reason,omitempty"`
	OverrideBy     *uuid.UUID `json:"override_by,omitempty"`
	OverrideAt     *time.Time `json:"override_at,omitempty"`
	// CanWin is true when the identity record is complete or an admin overrode the check.
	CanWin bool `json:"can_win"`
	// How the deal is paid, and its AML review state.
	PaymentMethod string   `json:"payment_method"`
	CashAmount    float64  `json:"cash_amount"`
	AMLStatus     string   `json:"aml_status"`
	AMLReasons    []string `json:"aml_reasons"`
}

// DealKYC checks the identity record of a deal's buyer (the lead's contact).
func (r *KYCRepo) DealKYC(ctx context.Context, dealID uuid.UUID) (*DealKYC, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	k := &DealKYC{DealID: dealID}
	err = r.db.QueryRow(ctx, `
		SELECT c.id, c.full_name, d.kyc_override_reason, d.kyc_override_by, d.kyc_override_at,
		       d.payment_method, d.cash_amount::float8, d.aml_status, d.aml_reasons
		  FROM deals d JOIN leads l ON l.id = d.lead_id JOIN contacts c ON c.id = l.contact_id
		 WHERE d.id = $1 AND d.company_id = $2`, dealID, cid).Scan(&k.ContactID, &k.ContactName, &k.OverrideReason, &k.OverrideBy, &k.OverrideAt,
		&k.PaymentMethod, &k.CashAmount, &k.AMLStatus, &k.AMLReasons)
	if err != nil {
		return nil, err
	}
	docs, err := r.Documents(ctx, k.ContactID)
	if err != nil {
		return nil, err
	}
	k.Missing = kyc.Check(docsOf(docs), false, time.Now())
	if k.Missing == nil {
		k.Missing = []string{}
	}
	k.CanWin = len(k.Missing) == 0 || k.OverrideAt != nil
	return k, nil
}

// LeadKYCMissing returns what is missing from the identity record of a lead's contact,
// for a deal created directly as won.
func (r *KYCRepo) LeadKYCMissing(ctx context.Context, leadID uuid.UUID) ([]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	var contactID uuid.UUID
	if err := r.db.QueryRow(ctx, `SELECT contact_id FROM leads WHERE id = $1 AND company_id = $2`, leadID, cid).Scan(&contactID); err != nil {
		return nil, notFoundRef(err)
	}
	docs, err := r.Documents(ctx, contactID)
	if err != nil {
		return nil, err
	}
	return kyc.Check(docsOf(docs), false, time.Now()), nil
}

// OverrideDealKYC lets a deal be won without a complete identity record.
func (r *KYCRepo) OverrideDealKYC(ctx context.Context, dealID uuid.UUID, reason string, by uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE deals SET kyc_override_reason = $1, kyc_override_by = $2, kyc_override_at = NOW(), updated_at = NOW()
		WHERE id = $3 AND company_id = $4`, reason, by, dealID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── AML ───────────────────────────────────────────────────────────────────────

// AMLSettings returns the company's review criteria.
func (r *KYCRepo) AMLSettings(ctx context.Context) (kyc.AMLSettings, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return kyc.AMLSettings{}, err
	}
	s := kyc.AMLSettings{CashThreshold: 55000, FlagVirtualAsset: true}
	err = r.db.QueryRow(ctx, `SELECT aml_cash_threshold::float8, aml_flag_virtual_asset FROM company_settings WHERE company_id = $1`,
		cid).Scan(&s.CashThreshold, &s.FlagVirtualAsset)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return s, err
}

// UpdateAMLSettings saves the company's review criteria.
func (r *KYCRepo) UpdateAMLSettings(ctx context.Context, s kyc.AMLSettings) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE company_settings SET aml_cash_threshold = $1, aml_flag_virtual_asset = $2 WHERE company_id = $3`,
		s.CashThreshold, s.FlagVirtualAsset, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetDealPayment records how a deal is paid and re-evaluates the review criteria. A deal
// that meets them is flagged for review; a flagged deal that no longer meets them is
// unflagged. A reviewed deal keeps its review. It returns the AML status.
func (r *KYCRepo) SetDealPayment(ctx context.Context, dealID uuid.UUID, method string, cashAmount float64) (string, []string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", nil, err
	}
	settings, err := r.AMLSettings(ctx)
	if err != nil {
		return "", nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var amount float64
	var status string
	if err := tx.QueryRow(ctx, `SELECT amount::float8, aml_status FROM deals WHERE id = $1 AND company_id = $2 FOR UPDATE`,
		dealID, cid).Scan(&amount, &status); err != nil {
		return "", nil, err
	}
	reasons := kyc.FlagReasons(method, amount, cashAmount, settings)
	if reasons == nil {
		reasons = []string{}
	}
	switch {
	case len(reasons) > 0 && status == "":
		status = "flagged"
	case len(reasons) == 0 && status == "flagged":
		status = ""
	}
	if _, err := tx.Exec(ctx, `
		UPDATE deals SET payment_method = $1, cash_amount = $2, aml_status = $3::text, aml_reasons = $4,
		       aml_flagged_at = CASE WHEN $3::text = 'flagged' AND aml_flagged_at IS NULL THEN NOW()
		                             WHEN $3::text = '' THEN NULL ELSE aml_flagged_at END,
		       updated_at = NOW()
		 WHERE id = $5`, method, cashAmount, status, reasons, dealID); err != nil {
		return "", nil, err
	}
	return status, reasons, tx.Commit(ctx)
}

// AMLCase is a deal in the review queue.
type AMLCase struct {
	DealID         uuid.UUID  `json:"deal_id"`
	Title          string     `json:"title"`
	Stage          string     `json:"stage"`
	Amount         float64    `json:"amount"`
	Currency       string     `json:"currency"`
	PaymentMethod  string     `json:"payment_method"`
	CashAmount     float64    `json:"cash_amount"`
	Status         string     `json:"status"`
	Reasons        []string   `json:"reasons"`
	FlaggedAt      *time.Time `json:"flagged_at"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	Notes          string     `json:"notes"`
	GoAMLReference string     `json:"goaml_reference"`
	ContactID      uuid.UUID  `json:"contact_id"`
	ContactName    string     `json:"contact_name"`
	ContactRisk    string     `json:"contact_risk"`
	ContactIsPEP   bool       `json:"contact_is_pep"`
	Screening      string     `json:"screening"`
	OwnerName      string     `json:"owner_name"`
}

// AMLCases lists deals by AML status ("" lists every flagged or reviewed deal).
func (r *KYCRepo) AMLCases(ctx context.Context, status string) ([]AMLCase, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.title, d.stage, d.amount::float8, d.currency, d.payment_method, d.cash_amount::float8, d.aml_status,
		       d.aml_reasons, d.aml_flagged_at, d.aml_reviewed_at, d.aml_notes, d.goaml_reference,
		       c.id, c.full_name, c.kyc_risk, c.kyc_is_pep, c.kyc_screening, COALESCE(u.name, '')
		  FROM deals d JOIN leads l ON l.id = d.lead_id JOIN contacts c ON c.id = l.contact_id
		  LEFT JOIN users u ON u.id = d.owner_id
		 WHERE d.company_id = $1 AND d.aml_status <> '' AND ($2 = '' OR d.aml_status = $2)
		 ORDER BY (d.aml_status = 'flagged') DESC, d.aml_flagged_at DESC NULLS LAST LIMIT 500`, cid, status)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (AMLCase, error) {
		var a AMLCase
		err := row.Scan(&a.DealID, &a.Title, &a.Stage, &a.Amount, &a.Currency, &a.PaymentMethod, &a.CashAmount, &a.Status,
			&a.Reasons, &a.FlaggedAt, &a.ReviewedAt, &a.Notes, &a.GoAMLReference,
			&a.ContactID, &a.ContactName, &a.ContactRisk, &a.ContactIsPEP, &a.Screening, &a.OwnerName)
		return a, err
	})
	if out == nil {
		out = []AMLCase{}
	}
	return out, err
}

// ReviewAML records a reviewer's decision on a flagged (or previously reviewed) deal:
// "no_report" or "reported" (with the goAML reference).
func (r *KYCRepo) ReviewAML(ctx context.Context, dealID uuid.UUID, decision, notes, ref string, by uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE deals SET aml_status = $1, aml_notes = $2, goaml_reference = $3, aml_reviewed_by = $4, aml_reviewed_at = NOW(), updated_at = NOW()
		 WHERE id = $5 AND company_id = $6 AND aml_status <> ''`, decision, notes, ref, by, dealID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// AMLExportRow is one line of the goAML preparation export.
type AMLExportRow struct {
	DealID, Title, Stage, Currency, PaymentMethod, Status, Reasons, GoAMLRef string
	Amount, CashAmount                                                       float64
	FlaggedAt, DealDate                                                      *time.Time
	ClientName, ClientNationality, ClientEmail, ClientPhone, ClientAddress   string
	ClientDOB                                                                *time.Time
	ClientRisk, ClientScreening                                              string
	ClientPEP                                                                bool
	IDType, IDNumber, IDCountry                                              string
	IDExpiry                                                                 *time.Time
	AgentName, AgentBRN                                                      string
	PropertyTitle, PropertyReference, PropertyEmirate, PropertyCommunity     string
}

// AMLExport returns the data a reporting officer needs to prepare goAML reports for the
// company's flagged and reviewed deals flagged in [from, to).
func (r *KYCRepo) AMLExport(ctx context.Context, status string, from, to time.Time) ([]AMLExportRow, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT d.id::text, d.title, d.stage, d.currency, d.payment_method, d.aml_status, array_to_string(d.aml_reasons, ';'),
		       d.goaml_reference, d.amount::float8, d.cash_amount::float8, d.aml_flagged_at, d.close_date,
		       c.full_name, TRIM(c.nationality), COALESCE(c.email,''), c.phone_wa, c.address, c.date_of_birth,
		       c.kyc_risk, c.kyc_screening, c.kyc_is_pep,
		       COALESCE(k.doc_type,''), COALESCE(k.number,''), COALESCE(TRIM(k.issuing_country),''), k.expiry_date,
		       COALESCE(u.name,''), COALESCE(u.brn,''),
		       COALESCE(li.title,''), COALESCE(li.reference_number,''), COALESCE(li.emirate,''), COALESCE(li.community,'')
		  FROM deals d
		  JOIN leads l ON l.id = d.lead_id
		  JOIN contacts c ON c.id = l.contact_id
		  LEFT JOIN LATERAL (SELECT doc_type, number, issuing_country, expiry_date FROM kyc_documents
		                      WHERE contact_id = c.id AND doc_type IN ('emirates_id','passport','trade_licence')
		                      ORDER BY expiry_date DESC NULLS LAST LIMIT 1) k ON TRUE
		  LEFT JOIN users u ON u.id = d.owner_id
		  LEFT JOIN LATERAL (SELECT li.title, li.reference_number, li.emirate, li.community FROM offers o
		                       JOIN listings li ON li.id = o.listing_id WHERE o.deal_id = d.id LIMIT 1) li ON TRUE
		 WHERE d.company_id = $1 AND d.aml_status <> '' AND ($2 = '' OR d.aml_status = $2)
		   AND COALESCE(d.aml_flagged_at, d.updated_at) >= $3 AND COALESCE(d.aml_flagged_at, d.updated_at) < $4
		 ORDER BY d.aml_flagged_at`, cid, status, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (AMLExportRow, error) {
		var e AMLExportRow
		err := row.Scan(&e.DealID, &e.Title, &e.Stage, &e.Currency, &e.PaymentMethod, &e.Status, &e.Reasons, &e.GoAMLRef,
			&e.Amount, &e.CashAmount, &e.FlaggedAt, &e.DealDate,
			&e.ClientName, &e.ClientNationality, &e.ClientEmail, &e.ClientPhone, &e.ClientAddress, &e.ClientDOB,
			&e.ClientRisk, &e.ClientScreening, &e.ClientPEP,
			&e.IDType, &e.IDNumber, &e.IDCountry, &e.IDExpiry, &e.AgentName, &e.AgentBRN,
			&e.PropertyTitle, &e.PropertyReference, &e.PropertyEmirate, &e.PropertyCommunity)
		return e, err
	})
}
