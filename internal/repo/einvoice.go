package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/einvoice"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// EInvoiceRepo stores the e-invoicing settings and each invoice's submission.
type EInvoiceRepo struct {
	db *pgxpool.Pool
}

func NewEInvoiceRepo(db *pgxpool.Pool) *EInvoiceRepo { return &EInvoiceRepo{db: db} }

// EInvoiceSettings are a company's e-invoicing settings.
type EInvoiceSettings struct {
	Provider              string `json:"provider"` // off | mock | an installed connector
	TIN                   string `json:"tin"`
	LegalIDType           string `json:"legal_id_type"`
	TradeLicenseAuthority string `json:"trade_license_authority"`
	City                  string `json:"city"`
	ServiceCode           string `json:"service_code"`
	ExemptionCode         string `json:"exemption_code"`
}

// Settings returns the company's e-invoicing settings.
func (r *EInvoiceRepo) Settings(ctx context.Context) (*EInvoiceSettings, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	s := &EInvoiceSettings{Provider: "off", LegalIDType: einvoice.LegalTradeLicence}
	err = r.db.QueryRow(ctx, `SELECT einvoice_provider, einvoice_tin, legal_id_type, trade_license_authority, business_city,
		einvoice_service_code, einvoice_exemption_code FROM company_settings WHERE company_id = $1`, cid).Scan(
		&s.Provider, &s.TIN, &s.LegalIDType, &s.TradeLicenseAuthority, &s.City, &s.ServiceCode, &s.ExemptionCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, nil
	}
	return s, err
}

// UpdateSettings saves the company's e-invoicing settings.
func (r *EInvoiceRepo) UpdateSettings(ctx context.Context, s *EInvoiceSettings) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE company_settings SET einvoice_provider = $1, einvoice_tin = $2, legal_id_type = $3,
		trade_license_authority = $4, business_city = $5, einvoice_service_code = $6, einvoice_exemption_code = $7
		WHERE company_id = $8`, s.Provider, s.TIN, s.LegalIDType, s.TradeLicenseAuthority, s.City, s.ServiceCode, s.ExemptionCode, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Seller returns the company as the e-invoice seller, with its bank account.
func (r *EInvoiceRepo) Seller(ctx context.Context) (einvoice.Party, string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return einvoice.Party{}, "", err
	}
	var p einvoice.Party
	var region, iban string
	err = r.db.QueryRow(ctx, `SELECT name, einvoice_tin, vat_number, trade_license_no, legal_id_type, trade_license_authority,
		business_address, business_city, region, country, COALESCE(business_email,''), COALESCE(bank_iban,'')
		FROM company_settings WHERE company_id = $1`, cid).Scan(&p.Name, &p.TIN, &p.TRN, &p.LegalID, &p.LegalIDType,
		&p.LegalAuthority, &p.Street, &p.City, &region, &p.Country, &p.Email, &iban)
	if err != nil {
		return p, "", err
	}
	p.Emirate = einvoice.Emirates[region]
	if p.LegalIDType != einvoice.LegalTradeLicence && p.LegalIDType != einvoice.LegalPassport {
		p.LegalAuthority = ""
	}
	return p, strings.ReplaceAll(iban, " ", ""), nil
}

// EInvoice is an invoice's submission to the provider.
type EInvoice struct {
	InvoiceID    uuid.UUID      `json:"invoice_id"`
	Provider     string         `json:"provider"`
	Status       string         `json:"status"`
	ProviderRef  string         `json:"provider_ref"`
	Reason       string         `json:"reason,omitempty"`
	Buyer        einvoice.Party `json:"buyer"`
	CreditReason string         `json:"credit_reason,omitempty"`
	DueDate      *time.Time     `json:"due_date,omitempty"`
	SubmittedBy  *uuid.UUID     `json:"submitted_by,omitempty"`
	SubmittedAt  time.Time      `json:"submitted_at"`
	DecidedAt    *time.Time     `json:"decided_at,omitempty"`
	XML          string         `json:"-"`
}

const einvoiceCols = `invoice_id, provider, status, provider_ref, reason, buyer, credit_reason, due_date, submitted_by, submitted_at, decided_at, xml`

func scanEInvoice(row pgx.Row, e *EInvoice) error {
	var buyer []byte
	if err := row.Scan(&e.InvoiceID, &e.Provider, &e.Status, &e.ProviderRef, &e.Reason, &buyer, &e.CreditReason, &e.DueDate,
		&e.SubmittedBy, &e.SubmittedAt, &e.DecidedAt, &e.XML); err != nil {
		return err
	}
	return json.Unmarshal(buyer, &e.Buyer)
}

// Get returns an invoice's e-invoice record.
func (r *EInvoiceRepo) Get(ctx context.Context, invoiceID uuid.UUID) (*EInvoice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	e := &EInvoice{}
	if err := scanEInvoice(r.db.QueryRow(ctx, `SELECT `+einvoiceCols+` FROM einvoices WHERE invoice_id = $1 AND company_id = $2`, invoiceID, cid), e); err != nil {
		return nil, err
	}
	return e, nil
}

// ErrAlreadySubmitted is returned when an invoice is already submitted or accepted.
var ErrAlreadySubmitted = errors.New("this invoice has already been submitted")

// Save records a submission. An invoice that was rejected, or could not be submitted, may
// be submitted again; one that is in progress or accepted may not.
func (r *EInvoiceRepo) Save(ctx context.Context, e *EInvoice) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	buyer, _ := json.Marshal(e.Buyer)
	tag, err := r.db.Exec(ctx, `
		INSERT INTO einvoices (invoice_id, company_id, provider, status, provider_ref, reason, buyer, credit_reason, due_date, xml, submitted_by)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11 WHERE EXISTS (SELECT 1 FROM vat_invoices WHERE id = $1 AND company_id = $2)
		ON CONFLICT (invoice_id) DO UPDATE SET provider = EXCLUDED.provider, status = EXCLUDED.status, provider_ref = EXCLUDED.provider_ref,
		       reason = EXCLUDED.reason, buyer = EXCLUDED.buyer, credit_reason = EXCLUDED.credit_reason, due_date = EXCLUDED.due_date,
		       xml = EXCLUDED.xml, submitted_by = EXCLUDED.submitted_by, submitted_at = NOW(), decided_at = NULL, updated_at = NOW()
		 WHERE einvoices.status IN ('rejected','error')`,
		e.InvoiceID, cid, e.Provider, e.Status, e.ProviderRef, e.Reason, buyer, e.CreditReason, e.DueDate, e.XML, e.SubmittedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		_ = r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM einvoices WHERE invoice_id = $1 AND company_id = $2)`, e.InvoiceID, cid).Scan(&exists)
		if exists {
			return ErrAlreadySubmitted
		}
		return ErrForeignReference
	}
	return nil
}

// PendingEInvoice is a submission awaiting the provider's decision.
type PendingEInvoice struct {
	InvoiceID   uuid.UUID
	InvoiceNo   string
	Provider    string
	ProviderRef string
	Issuer      *uuid.UUID // who issued the invoice
	SubmittedBy *uuid.UUID
}

// Pending lists the company's submissions still awaiting a decision.
func (r *EInvoiceRepo) Pending(ctx context.Context) ([]PendingEInvoice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT e.invoice_id, v.invoice_no, e.provider, e.provider_ref, v.created_by, e.submitted_by
		FROM einvoices e JOIN vat_invoices v ON v.id = e.invoice_id WHERE e.company_id = $1 AND e.status = 'submitted' LIMIT 500`, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (PendingEInvoice, error) {
		var p PendingEInvoice
		err := row.Scan(&p.InvoiceID, &p.InvoiceNo, &p.Provider, &p.ProviderRef, &p.Issuer, &p.SubmittedBy)
		return p, err
	})
}

// Decide records the provider's decision on a submitted e-invoice. It reports false when
// the record was no longer awaiting a decision.
func (r *EInvoiceRepo) Decide(ctx context.Context, invoiceID uuid.UUID, status, reason string) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.db.Exec(ctx, `UPDATE einvoices SET status = $1, reason = $2, decided_at = NOW(), updated_at = NOW()
		WHERE invoice_id = $3 AND company_id = $4 AND status = 'submitted'`, status, reason, invoiceID, cid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
