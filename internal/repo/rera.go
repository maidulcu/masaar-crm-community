package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// ReraRepo stores pre-filled RERA forms and gathers the data to pre-fill them.
type ReraRepo struct {
	db *pgxpool.Pool
}

func NewReraRepo(db *pgxpool.Pool) *ReraRepo { return &ReraRepo{db: db} }

// ReraForm is a stored form with its document's signature state.
type ReraForm struct {
	DocumentID      uuid.UUID         `json:"document_id"`
	FormType        string            `json:"form_type"`
	ListingID       *uuid.UUID        `json:"listing_id,omitempty"`
	ContactID       *uuid.UUID        `json:"contact_id,omitempty"`
	OfferID         *uuid.UUID        `json:"offer_id,omitempty"`
	Data            map[string]string `json:"data"`
	Title           string            `json:"title"`
	SignatureStatus string            `json:"signature_status"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Prefill gathers values for a form from the company, the listing (and its agent), the
// contact and the offer. Any id may be nil; each must belong to the caller's company.
func (r *ReraRepo) Prefill(ctx context.Context, listingID, contactID, offerID *uuid.UUID, agentID uuid.UUID) (map[string]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	v := map[string]string{}
	var name, orn, addr, phone string
	if err := r.db.QueryRow(ctx, `SELECT name, orn, business_address, COALESCE(business_phone,'') FROM company_settings WHERE company_id = $1`,
		cid).Scan(&name, &orn, &addr, &phone); err == nil {
		v["broker_company"], v["broker_orn"], v["broker_address"], v["broker_phone"] = name, orn, addr, phone
	}

	if offerID != nil {
		var amount float64
		var currency, terms string
		var lID, cID uuid.UUID
		err := r.db.QueryRow(ctx, `SELECT listing_id, contact_id, offer_amount, currency, terms FROM offers WHERE id = $1 AND company_id = $2`,
			*offerID, cid).Scan(&lID, &cID, &amount, &currency, &terms)
		if err != nil {
			return nil, notFoundRef(err)
		}
		v["sale_price"] = strconv.FormatFloat(amount, 'f', -1, 64)
		v["currency"] = strings.TrimSpace(currency)
		v["deposit_amount"] = strconv.FormatFloat(amount*0.10, 'f', 2, 64)
		v["special_conditions"] = terms
		if listingID == nil {
			listingID = &lID
		}
		if contactID == nil {
			contactID = &cID
		}
	}

	agent := agentID
	if listingID != nil {
		var title, ptype, ltype, emirate, community, area, ref, ownerName, ownerPhone, ownerEmail, permit, currency string
		var sqft, price float64
		var assigned *uuid.UUID
		err := r.db.QueryRow(ctx, `
			SELECT title, property_type, listing_type, COALESCE(emirate,''), COALESCE(community,''), COALESCE(area,''),
			       COALESCE(reference_number,''), COALESCE(owner_name,''), COALESCE(owner_phone,''), COALESCE(owner_email,''),
			       permit_number, COALESCE(total_sqft,0), price, COALESCE(currency,'AED'), assigned_to
			  FROM listings WHERE id = $1 AND company_id = $2`, *listingID, cid).Scan(
			&title, &ptype, &ltype, &emirate, &community, &area, &ref, &ownerName, &ownerPhone, &ownerEmail,
			&permit, &sqft, &price, &currency, &assigned)
		if err != nil {
			return nil, notFoundRef(err)
		}
		v["property_title"], v["property_type"], v["emirate"] = title, ptype, emirate
		v["community"] = strings.Trim(strings.Join([]string{community, area}, ", "), ", ")
		v["unit_reference"], v["permit_number"] = ref, permit
		if sqft > 0 {
			v["size_sqft"] = strconv.FormatFloat(sqft, 'f', -1, 64)
		}
		v["owner_name"], v["owner_phone"], v["owner_email"] = ownerName, ownerPhone, ownerEmail
		v["listing_type"] = ltype
		v["asking_price"] = fmt.Sprintf("%s %s", strings.TrimSpace(currency), strconv.FormatFloat(price, 'f', -1, 64))
		if assigned != nil {
			agent = *assigned
		}
	}

	if contactID != nil {
		var cname, cphone, cemail, nationality string
		err := r.db.QueryRow(ctx, `SELECT full_name, phone_wa, COALESCE(email,''), nationality FROM contacts
			WHERE id = $1 AND company_id = $2 AND erased_at IS NULL`, *contactID, cid).Scan(&cname, &cphone, &cemail, &nationality)
		if err != nil {
			return nil, notFoundRef(err)
		}
		v["buyer_name"], v["buyer_phone"], v["buyer_email"], v["buyer_nationality"] = cname, cphone, cemail, strings.TrimSpace(nationality)
	}

	var agentName, brn string
	if err := r.db.QueryRow(ctx, `SELECT name, brn FROM users WHERE id = $1 AND company_id = $2`, agent, cid).Scan(&agentName, &brn); err == nil {
		v["agent_name"], v["agent_brn"] = agentName, brn
	}
	today := time.Now()
	v["start_date"] = today.Format("2006-01-02")
	v["end_date"] = today.AddDate(0, 3, 0).Format("2006-01-02")
	return v, nil
}

func notFoundRef(err error) error {
	if err == pgx.ErrNoRows {
		return ErrForeignReference
	}
	return err
}

// Create stores a form as a pending document plus its data, in one transaction.
func (r *ReraRepo) Create(ctx context.Context, f *ReraForm, createdBy uuid.UUID, relatedType string, relatedID *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(f.Data)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	f.DocumentID = uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO documents (id, company_id, document_type, related_entity_type, related_entity_id, document_title,
		                       file_url, signature_status, created_by, data_classification)
		VALUES ($1, $2, $3, $4, $5, $6, '', 'pending', $7, 'confidential')`,
		f.DocumentID, cid, "rera_form_"+strings.ToLower(f.FormType), relatedType, relatedID, f.Title, createdBy); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO rera_forms (document_id, company_id, form_type, listing_id, contact_id, offer_id, data, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING created_at`,
		f.DocumentID, cid, f.FormType, f.ListingID, f.ContactID, f.OfferID, raw, createdBy).Scan(&f.CreatedAt); err != nil {
		return err
	}
	f.SignatureStatus = "pending"
	return tx.Commit(ctx)
}

const reraCols = `f.document_id, f.form_type, f.listing_id, f.contact_id, f.offer_id, f.data,
	COALESCE(d.document_title,''), COALESCE(d.signature_status,'pending'), f.created_at`

func scanRera(row pgx.Row, f *ReraForm) error {
	var raw []byte
	if err := row.Scan(&f.DocumentID, &f.FormType, &f.ListingID, &f.ContactID, &f.OfferID, &raw,
		&f.Title, &f.SignatureStatus, &f.CreatedAt); err != nil {
		return err
	}
	return json.Unmarshal(raw, &f.Data)
}

// Get returns one of the company's forms.
func (r *ReraRepo) Get(ctx context.Context, documentID uuid.UUID) (*ReraForm, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	f := &ReraForm{}
	err = scanRera(r.db.QueryRow(ctx, `SELECT `+reraCols+` FROM rera_forms f JOIN documents d ON d.id = f.document_id
		WHERE f.document_id = $1 AND f.company_id = $2 AND d.deleted_at IS NULL`, documentID, cid), f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// List returns forms for a listing and/or contact, newest first.
func (r *ReraRepo) List(ctx context.Context, listingID, contactID *uuid.UUID) ([]ReraForm, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+reraCols+` FROM rera_forms f JOIN documents d ON d.id = f.document_id
		WHERE f.company_id = $1 AND d.deleted_at IS NULL
		  AND ($2::uuid IS NULL OR f.listing_id = $2) AND ($3::uuid IS NULL OR f.contact_id = $3)
		ORDER BY f.created_at DESC LIMIT 100`, cid, listingID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReraForm
	for rows.Next() {
		var f ReraForm
		if err := scanRera(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ForDocument returns the form attached to a document, for the public signing page.
// The caller reached the document through a signature token, so no company is bound.
func (r *ReraRepo) ForDocument(ctx context.Context, documentID uuid.UUID) (*ReraForm, error) {
	f := &ReraForm{}
	err := scanRera(r.db.QueryRow(ctx, `SELECT `+reraCols+` FROM rera_forms f JOIN documents d ON d.id = f.document_id
		WHERE f.document_id = $1`, documentID), f)
	if err != nil {
		return nil, err
	}
	return f, nil
}
