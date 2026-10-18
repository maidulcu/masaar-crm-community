package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// ContactProfileRepo stores contact details beyond the core fields, organisations,
// custom fields, and finds and merges duplicate contacts.
type ContactProfileRepo struct {
	db *pgxpool.Pool
}

func NewContactProfileRepo(db *pgxpool.Pool) *ContactProfileRepo { return &ContactProfileRepo{db: db} }

// ── Profile ───────────────────────────────────────────────────────────────────

// GetProfile returns a contact's profile.
func (r *ContactProfileRepo) GetProfile(ctx context.Context, contactID uuid.UUID) (*domain.ContactProfile, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	p := &domain.ContactProfile{}
	err = r.db.QueryRow(ctx, `
		SELECT c.secondary_phones, c.nationality, c.source, c.preferred_contact_time, c.address, c.notes,
		       c.job_title, c.organisation_id, COALESCE(o.name, '')
		  FROM contacts c LEFT JOIN organisations o ON o.id = c.organisation_id AND o.company_id = c.company_id
		 WHERE c.id = $1 AND c.company_id = $2`, contactID, cid).Scan(
		&p.SecondaryPhones, &p.Nationality, &p.Source, &p.PreferredContactTime, &p.Address, &p.Notes,
		&p.JobTitle, &p.OrganisationID, &p.OrganisationName)
	if err != nil {
		return nil, err
	}
	p.Nationality = strings.TrimSpace(p.Nationality)
	return p, nil
}

// UpdateProfile replaces a contact's profile. The organisation must be the company's.
func (r *ContactProfileRepo) UpdateProfile(ctx context.Context, contactID uuid.UUID, p *domain.ContactProfile) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if p.SecondaryPhones == nil {
		p.SecondaryPhones = []string{}
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE contacts SET secondary_phones = $1, nationality = $2, source = $3, preferred_contact_time = $4,
		       address = $5, notes = $6, job_title = $7, organisation_id = $8, updated_at = NOW()
		 WHERE id = $9 AND company_id = $10 AND erased_at IS NULL
		   AND ($8::uuid IS NULL OR EXISTS (SELECT 1 FROM organisations WHERE id = $8 AND company_id = $10))`,
		p.SecondaryPhones, p.Nationality, p.Source, p.PreferredContactTime, p.Address, p.Notes, p.JobTitle,
		p.OrganisationID, contactID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrForeignReference
	}
	return nil
}

// ── Organisations ─────────────────────────────────────────────────────────────

const orgCols = `o.id, o.name, o.trade_license_no, o.trn, o.address, o.phone, o.email, o.website, o.notes,
	(SELECT COUNT(*) FROM contacts c WHERE c.organisation_id = o.id), o.created_at, o.updated_at`

func scanOrg(row pgx.Row, o *domain.Organisation) error {
	return row.Scan(&o.ID, &o.Name, &o.TradeLicenseNo, &o.TRN, &o.Address, &o.Phone, &o.Email, &o.Website,
		&o.Notes, &o.ContactCount, &o.CreatedAt, &o.UpdatedAt)
}

// ListOrganisations returns the company's organisations, optionally filtered by name.
func (r *ContactProfileRepo) ListOrganisations(ctx context.Context, search string, limit, offset int) ([]domain.Organisation, int, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM organisations o WHERE o.company_id = $1 AND ($2 = '' OR o.name ILIKE '%' || $2 || '%')`,
		cid, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+orgCols+` FROM organisations o
		WHERE o.company_id = $1 AND ($2 = '' OR o.name ILIKE '%' || $2 || '%')
		ORDER BY lower(o.name) LIMIT $3 OFFSET $4`, cid, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Organisation
	for rows.Next() {
		var o domain.Organisation
		if err := scanOrg(rows, &o); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

// GetOrganisation returns one organisation.
func (r *ContactProfileRepo) GetOrganisation(ctx context.Context, id uuid.UUID) (*domain.Organisation, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	o := &domain.Organisation{}
	if err := scanOrg(r.db.QueryRow(ctx, `SELECT `+orgCols+` FROM organisations o WHERE o.id = $1 AND o.company_id = $2`, id, cid), o); err != nil {
		return nil, err
	}
	return o, nil
}

// SaveOrganisation creates (ID nil) or updates an organisation.
func (r *ContactProfileRepo) SaveOrganisation(ctx context.Context, o *domain.Organisation) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
		_, err = r.db.Exec(ctx, `INSERT INTO organisations (id, company_id, name, trade_license_no, trn, address, phone, email, website, notes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			o.ID, cid, o.Name, o.TradeLicenseNo, o.TRN, o.Address, o.Phone, o.Email, o.Website, o.Notes)
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE organisations SET name=$1, trade_license_no=$2, trn=$3, address=$4, phone=$5,
		email=$6, website=$7, notes=$8, updated_at=NOW() WHERE id=$9 AND company_id=$10`,
		o.Name, o.TradeLicenseNo, o.TRN, o.Address, o.Phone, o.Email, o.Website, o.Notes, o.ID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteOrganisation removes an organisation; its contacts keep existing without one.
func (r *ContactProfileRepo) DeleteOrganisation(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM organisations WHERE id = $1 AND company_id = $2`, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// OrganisationContacts lists an organisation's contacts.
func (r *ContactProfileRepo) OrganisationContacts(ctx context.Context, orgID uuid.UUID) ([]domain.Contact, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT id, phone_wa, full_name, COALESCE(email,''), language, lead_score, assigned_to, created_at, updated_at
		FROM contacts WHERE organisation_id = $1 AND company_id = $2 ORDER BY full_name`, orgID, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Contact, error) {
		var c domain.Contact
		err := row.Scan(&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore, &c.AssignedTo, &c.CreatedAt, &c.UpdatedAt)
		return c, err
	})
}

// ── Custom fields ─────────────────────────────────────────────────────────────

// ErrCustomFieldInvalid is returned for a value that does not fit its field.
var ErrCustomFieldInvalid = errors.New("invalid custom field value")

// ListCustomFields returns the company's field definitions, optionally for one entity type.
func (r *ContactProfileRepo) ListCustomFields(ctx context.Context, entityType string) ([]domain.CustomField, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, entity_type, field_name, COALESCE(field_label, field_name), field_type, options,
		       COALESCE(is_required, false), COALESCE(display_order, 0)
		  FROM custom_field_definitions WHERE company_id = $1 AND ($2 = '' OR entity_type = $2)
		 ORDER BY entity_type, COALESCE(display_order, 0), field_name`, cid, entityType)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.CustomField, error) {
		var f domain.CustomField
		err := row.Scan(&f.ID, &f.EntityType, &f.Name, &f.Label, &f.Type, &f.Options, &f.Required, &f.DisplayOrder)
		return f, err
	})
}

// SaveCustomField creates (ID nil) or updates a field definition. The machine name and
// entity type cannot change after creation.
func (r *ContactProfileRepo) SaveCustomField(ctx context.Context, f *domain.CustomField) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if f.Options == nil {
		f.Options = []string{}
	}
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
		_, err = r.db.Exec(ctx, `INSERT INTO custom_field_definitions
			(id, company_id, entity_type, field_name, field_label, field_type, options, is_required, display_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			f.ID, cid, f.EntityType, f.Name, f.Label, f.Type, f.Options, f.Required, f.DisplayOrder)
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE custom_field_definitions SET field_label=$1, field_type=$2, options=$3,
		is_required=$4, display_order=$5 WHERE id=$6 AND company_id=$7`,
		f.Label, f.Type, f.Options, f.Required, f.DisplayOrder, f.ID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteCustomField removes a field and its values.
func (r *ContactProfileRepo) DeleteCustomField(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM custom_field_definitions WHERE id = $1 AND company_id = $2`, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CustomValues returns a record's custom values keyed by field name.
func (r *ContactProfileRepo) CustomValues(ctx context.Context, entityType string, entityID uuid.UUID) (map[string]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT d.field_name, COALESCE(v.value, '')
		  FROM custom_field_values v JOIN custom_field_definitions d ON d.id = v.field_id
		 WHERE v.company_id = $1 AND d.entity_type = $2 AND v.entity_id = $3`, cid, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// tableFor maps a custom-field entity type to its table, to check the record exists.
var customEntityTables = map[string]string{"contact": "contacts", "lead": "leads", "deal": "deals", "listing": "listings"}

// SetCustomValues stores values (by field name) for a record; an empty value deletes it.
// Unknown field names and values that do not fit the field type are refused.
func (r *ContactProfileRepo) SetCustomValues(ctx context.Context, entityType string, entityID uuid.UUID, values map[string]string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	table, ok := customEntityTables[entityType]
	if !ok {
		return ErrCustomFieldInvalid
	}
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = $1 AND company_id = $2)`,
		entityID, cid).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrForeignReference
	}
	fields, err := r.ListCustomFields(ctx, entityType)
	if err != nil {
		return err
	}
	byName := make(map[string]domain.CustomField, len(fields))
	for _, f := range fields {
		byName[f.Name] = f
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for name, value := range values {
		f, ok := byName[name]
		if !ok {
			return fmt.Errorf("%w: unknown field %q", ErrCustomFieldInvalid, name)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			if _, err := tx.Exec(ctx, `DELETE FROM custom_field_values WHERE field_id = $1 AND entity_id = $2`, f.ID, entityID); err != nil {
				return err
			}
			continue
		}
		if err := ValidateCustomValue(f, value); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO custom_field_values (company_id, field_id, entity_type, entity_id, value)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (field_id, entity_id) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			cid, f.ID, entityType, entityID, value); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ── Duplicates and merge ──────────────────────────────────────────────────────

// FindDuplicates groups the company's (non-erased) contacts that share an email address,
// a full name, or whose secondary phone is another contact's WhatsApp number.
func (r *ContactProfileRepo) FindDuplicates(ctx context.Context, limit int) ([]domain.DuplicateGroup, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		WITH live AS (SELECT * FROM contacts WHERE company_id = $1 AND erased_at IS NULL),
		pairs AS (
			SELECT 'email' AS reason, lower(email) AS key, id FROM live
			 WHERE email IS NOT NULL AND email <> ''
			   AND lower(email) IN (SELECT lower(email) FROM live WHERE email IS NOT NULL AND email <> ''
			                         GROUP BY lower(email) HAVING COUNT(*) > 1)
			UNION ALL
			SELECT 'name', lower(trim(full_name)), id FROM live
			 WHERE trim(full_name) <> '' AND lower(trim(full_name)) IN (
			       SELECT lower(trim(full_name)) FROM live GROUP BY lower(trim(full_name)) HAVING COUNT(*) > 1)
			UNION ALL
			SELECT 'phone', a.phone_wa, x.id FROM live a
			  JOIN live b ON a.phone_wa = ANY(b.secondary_phones) AND a.id <> b.id
			  CROSS JOIN LATERAL (VALUES (a.id), (b.id)) AS x(id)
		)
		SELECT DISTINCT p.reason, p.key, c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language,
		       c.lead_score, c.assigned_to, c.created_at, c.updated_at
		  FROM pairs p JOIN live c ON c.id = p.id
		 ORDER BY p.reason, p.key, c.created_at`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []domain.DuplicateGroup
	idx := map[string]int{}
	for rows.Next() {
		var reason, key string
		var c domain.Contact
		if err := rows.Scan(&reason, &key, &c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language,
			&c.LeadScore, &c.AssignedTo, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		gk := reason + "\x00" + key
		i, ok := idx[gk]
		if !ok {
			if limit > 0 && len(groups) >= limit {
				continue
			}
			groups = append(groups, domain.DuplicateGroup{Reason: reason, Key: key})
			i = len(groups) - 1
			idx[gk] = i
		}
		groups[i].Contacts = append(groups[i].Contacts, c)
	}
	return groups, rows.Err()
}

// Merge folds the `merge` contacts into `keep`: their leads, offers, viewings,
// conversations, communications, consent history, tasks and custom values move to the
// survivor, empty fields of the survivor are filled from them, their WhatsApp numbers
// become the survivor's secondary phones, and they are deleted. Same rules as the
// one-time merge in migration 0061.
func (r *ContactProfileRepo) Merge(ctx context.Context, keep uuid.UUID, merge []uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ids := append([]uuid.UUID{keep}, merge...)
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM (SELECT id FROM contacts WHERE id = ANY($1) AND company_id = $2
		AND erased_at IS NULL FOR UPDATE) s`, ids, cid).Scan(&n); err != nil {
		return err
	}
	if n != len(ids) {
		return ErrForeignReference
	}
	for _, dup := range merge {
		if dup == keep {
			return ErrForeignReference
		}
		steps := []string{
			`UPDATE leads SET contact_id = $1 WHERE contact_id = $2 AND company_id = $3`,
			`UPDATE offers SET contact_id = $1 WHERE contact_id = $2 AND company_id = $3`,
			`UPDATE viewings SET contact_id = $1 WHERE contact_id = $2 AND company_id = $3`,
			`UPDATE communication_history SET contact_id = $1 WHERE contact_id = $2 AND company_id = $3`,
			`UPDATE contact_consents SET contact_id = $1 WHERE contact_id = $2 AND company_id = $3`,
			`UPDATE tasks SET related_id = $1 WHERE related_type = 'contact' AND related_id = $2 AND company_id = $3`,
			// Custom values: keep the survivor's where both have one.
			`UPDATE custom_field_values v SET entity_id = $1 WHERE entity_id = $2 AND company_id = $3
			   AND NOT EXISTS (SELECT 1 FROM custom_field_values k WHERE k.field_id = v.field_id AND k.entity_id = $1)`,
			// What is left are values the survivor already has ($1 typed as above).
			`DELETE FROM custom_field_values WHERE entity_id = $2 AND company_id = $3 AND $1::uuid IS NOT NULL`,
			// Threads are unique per (contact, account): fold into the survivor's when it has one.
			`UPDATE whatsapp_messages m SET thread_id = k.id
			   FROM whatsapp_threads d JOIN whatsapp_threads k ON k.contact_id = $1 AND k.wa_account_id = d.wa_account_id
			  WHERE m.thread_id = d.id AND d.contact_id = $2 AND d.company_id = $3`,
			`UPDATE whatsapp_outbound o SET thread_id = k.id
			   FROM whatsapp_threads d JOIN whatsapp_threads k ON k.contact_id = $1 AND k.wa_account_id = d.wa_account_id
			  WHERE o.thread_id = d.id AND d.contact_id = $2 AND d.company_id = $3`,
			`UPDATE whatsapp_threads k SET message_count = k.message_count + d.message_count,
			        last_message_at = GREATEST(k.last_message_at, d.last_message_at),
			        ai_summary = COALESCE(NULLIF(k.ai_summary, ''), d.ai_summary)
			   FROM whatsapp_threads d WHERE k.contact_id = $1 AND d.contact_id = $2 AND d.company_id = $3
			    AND k.wa_account_id = d.wa_account_id`,
			`DELETE FROM whatsapp_threads d WHERE d.contact_id = $2 AND d.company_id = $3
			   AND EXISTS (SELECT 1 FROM whatsapp_threads k WHERE k.contact_id = $1 AND k.wa_account_id = d.wa_account_id)`,
			`UPDATE whatsapp_threads SET contact_id = $1 WHERE contact_id = $2 AND company_id = $3`,
			// Fill the survivor's empty fields and keep the duplicate's numbers.
			`UPDATE contacts k SET
			        email = COALESCE(NULLIF(k.email, ''), d.email),
			        assigned_to = COALESCE(k.assigned_to, d.assigned_to),
			        lead_score = GREATEST(k.lead_score, d.lead_score),
			        nationality = COALESCE(NULLIF(k.nationality, ''), d.nationality),
			        source = COALESCE(NULLIF(k.source, ''), d.source),
			        preferred_contact_time = COALESCE(NULLIF(k.preferred_contact_time, ''), d.preferred_contact_time),
			        address = COALESCE(NULLIF(k.address, ''), d.address),
			        job_title = COALESCE(NULLIF(k.job_title, ''), d.job_title),
			        organisation_id = COALESCE(k.organisation_id, d.organisation_id),
			        notes = CASE WHEN d.notes = '' THEN k.notes WHEN k.notes = '' THEN d.notes ELSE k.notes || E'\n\n' || d.notes END,
			        secondary_phones = ARRAY(SELECT DISTINCT p FROM unnest(k.secondary_phones || d.secondary_phones || ARRAY[d.phone_wa]) p
			                                  WHERE p <> k.phone_wa),
			        updated_at = NOW()
			   FROM contacts d WHERE k.id = $1 AND d.id = $2 AND d.company_id = $3`,
			// $1 is unused here but every step gets the same arguments; the cast types it.
			`DELETE FROM contacts WHERE id = $2 AND company_id = $3 AND $1::uuid IS NOT NULL`,
		}
		for _, q := range steps {
			if _, err := tx.Exec(ctx, q, keep, dup, cid); err != nil {
				return fmt.Errorf("merge contact: %w", err)
			}
		}
	}
	return tx.Commit(ctx)
}

// AllCustomValues returns every custom value of an entity type in the company, keyed by
// record and then field name (one query, for exports).
func (r *ContactProfileRepo) AllCustomValues(ctx context.Context, entityType string) (map[uuid.UUID]map[string]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT v.entity_id, d.field_name, COALESCE(v.value, '')
		  FROM custom_field_values v JOIN custom_field_definitions d ON d.id = v.field_id
		 WHERE v.company_id = $1 AND d.entity_type = $2`, cid, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]map[string]string{}
	for rows.Next() {
		var id uuid.UUID
		var k, v string
		if err := rows.Scan(&id, &k, &v); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = map[string]string{}
		}
		out[id][k] = v
	}
	return out, rows.Err()
}
