package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// LeaseOpsRepo holds the day-to-day property management data around leases: notices
// served on tenants, lease alerts and rent reminders.
type LeaseOpsRepo struct {
	db *pgxpool.Pool
}

func NewLeaseOpsRepo(db *pgxpool.Pool) *LeaseOpsRepo { return &LeaseOpsRepo{db: db} }

// ── Settings ──────────────────────────────────────────────────────────────────

// PropertySettings are the company's lease alert and rent reminder settings.
type PropertySettings struct {
	LeaseAlertDays         []int  `json:"lease_alert_days"`
	RentNoticeDays         int    `json:"rent_notice_days"`
	RentRemindersEnabled   bool   `json:"rent_reminders_enabled"`
	RentReminderDaysBefore int    `json:"rent_reminder_days_before"`
	RentReminderWATemplate string `json:"rent_reminder_wa_template"`
	RentReminderWALanguage string `json:"rent_reminder_wa_language"`
}

// Settings returns the company's property management settings.
func (r *LeaseOpsRepo) Settings(ctx context.Context) (*PropertySettings, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	s := &PropertySettings{}
	err = r.db.QueryRow(ctx, `
		SELECT lease_alert_days, rent_notice_days, rent_reminders_enabled, rent_reminder_days_before,
		       rent_reminder_wa_template, rent_reminder_wa_language
		  FROM company_settings WHERE company_id = $1`, cid).Scan(
		&s.LeaseAlertDays, &s.RentNoticeDays, &s.RentRemindersEnabled, &s.RentReminderDaysBefore,
		&s.RentReminderWATemplate, &s.RentReminderWALanguage)
	if errors.Is(err, pgx.ErrNoRows) {
		return &PropertySettings{LeaseAlertDays: []int{120, 90, 60, 30}, RentNoticeDays: 90, RentReminderDaysBefore: 3,
			RentReminderWALanguage: "en"}, nil
	}
	return s, err
}

// UpdateSettings saves the company's property management settings.
func (r *LeaseOpsRepo) UpdateSettings(ctx context.Context, s *PropertySettings) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE company_settings SET lease_alert_days = $1, rent_notice_days = $2, rent_reminders_enabled = $3,
		       rent_reminder_days_before = $4, rent_reminder_wa_template = $5, rent_reminder_wa_language = $6
		 WHERE company_id = $7`,
		s.LeaseAlertDays, s.RentNoticeDays, s.RentRemindersEnabled, s.RentReminderDaysBefore,
		s.RentReminderWATemplate, s.RentReminderWALanguage, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── Lease agent and Ejari expiry ──────────────────────────────────────────────

// SetLeaseAgent assigns the agent who looks after a lease (nil clears it). The agent
// must be one of the company's users.
func (r *LeaseOpsRepo) SetLeaseAgent(ctx context.Context, leaseID uuid.UUID, agentID *uuid.UUID, ejariExpiry *time.Time) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if agentID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND company_id = $2)`, *agentID, cid).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrForeignReference
		}
	}
	tag, err := r.db.Exec(ctx, `UPDATE leases SET agent_id = $1, ejari_expiry = $2, updated_at = NOW() WHERE id = $3 AND company_id = $4`,
		agentID, ejariExpiry, leaseID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// LeaseExtras are the lease fields added for property management.
type LeaseExtras struct {
	AgentID     *uuid.UUID `json:"agent_id"`
	AgentName   string     `json:"agent_name"`
	EjariExpiry *time.Time `json:"ejari_expiry"`
	// RentReminders is the tenant's choice to receive rent reminders.
	RentReminders bool `json:"rent_reminders"`
}

// Extras returns a lease's agent and Ejari expiry.
func (r *LeaseOpsRepo) Extras(ctx context.Context, leaseID uuid.UUID) (*LeaseExtras, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	e := &LeaseExtras{}
	err = r.db.QueryRow(ctx, `
		SELECT l.agent_id, COALESCE(u.name, ''), l.ejari_expiry, COALESCE(t.rent_reminders, TRUE) FROM leases l
		  LEFT JOIN users u ON u.id = l.agent_id AND u.company_id = l.company_id
		  LEFT JOIN tenants t ON t.id = l.tenant_id AND t.company_id = l.company_id
		 WHERE l.id = $1 AND l.company_id = $2`, leaseID, cid).Scan(&e.AgentID, &e.AgentName, &e.EjariExpiry, &e.RentReminders)
	return e, err
}

// SetRentReminders records whether the lease's tenant receives rent reminders.
func (r *LeaseOpsRepo) SetRentReminders(ctx context.Context, leaseID uuid.UUID, enabled bool) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE tenants t SET rent_reminders = $1, updated_at = NOW() FROM leases l
		WHERE l.id = $2 AND l.company_id = $3 AND t.id = l.tenant_id AND t.company_id = l.company_id`, enabled, leaseID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── Notices ───────────────────────────────────────────────────────────────────

const noticeCols = `id, lease_id, kind, current_rent::float8, new_rent::float8, effective_date, body_en, body_ar,
	sent_at, sent_via, created_by, created_at`

func scanNotice(row pgx.Row, n *domain.LeaseNotice) error {
	return row.Scan(&n.ID, &n.LeaseID, &n.Kind, &n.CurrentRent, &n.NewRent, &n.EffectiveDate, &n.BodyEN, &n.BodyAR,
		&n.SentAt, &n.SentVia, &n.CreatedBy, &n.CreatedAt)
}

// NoticeContext is what a notice is written from.
type NoticeContext struct {
	TenantName   string
	PropertyName string
	UnitNumber   string
	Emirate      string
	StartDate    time.Time
	EndDate      time.Time
	MonthlyRent  float64
	Currency     string
	EjariNumber  string
	CompanyName  string
}

// NoticeContext gathers the lease details a notice needs.
func (r *LeaseOpsRepo) NoticeContext(ctx context.Context, leaseID uuid.UUID) (*NoticeContext, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	n := &NoticeContext{}
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(t.full_name_en,''), COALESCE(p.name,''), COALESCE(p.unit_number,''), COALESCE(p.emirate,''),
		       l.start_date, l.end_date, l.monthly_rent::float8, COALESCE(NULLIF(TRIM(l.currency),''),'AED'),
		       COALESCE(l.ejari_number,''), COALESCE(s.name,'')
		  FROM leases l
		  LEFT JOIN tenants t ON t.id = l.tenant_id
		  LEFT JOIN rental_properties p ON p.id = l.property_id
		  LEFT JOIN company_settings s ON s.company_id = l.company_id
		 WHERE l.id = $1 AND l.company_id = $2`, leaseID, cid).Scan(
		&n.TenantName, &n.PropertyName, &n.UnitNumber, &n.Emirate, &n.StartDate, &n.EndDate, &n.MonthlyRent, &n.Currency,
		&n.EjariNumber, &n.CompanyName)
	if err != nil {
		return nil, notFoundRef(err)
	}
	return n, nil
}

// CreateNotice stores a notice (not yet sent).
func (r *LeaseOpsRepo) CreateNotice(ctx context.Context, n *domain.LeaseNotice) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	n.ID = uuid.New()
	tag, err := r.db.Exec(ctx, `
		INSERT INTO lease_notices (id, company_id, lease_id, kind, current_rent, new_rent, effective_date, body_en, body_ar, created_by)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10 WHERE EXISTS (SELECT 1 FROM leases WHERE id = $3 AND company_id = $2)`,
		n.ID, cid, n.LeaseID, n.Kind, n.CurrentRent, n.NewRent, n.EffectiveDate, n.BodyEN, n.BodyAR, n.CreatedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrForeignReference
	}
	got, err := r.GetNotice(ctx, n.ID)
	if err == nil {
		*n = *got
	}
	return err
}

// GetNotice returns one notice.
func (r *LeaseOpsRepo) GetNotice(ctx context.Context, id uuid.UUID) (*domain.LeaseNotice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	n := &domain.LeaseNotice{}
	if err := scanNotice(r.db.QueryRow(ctx, `SELECT `+noticeCols+` FROM lease_notices WHERE id = $1 AND company_id = $2`, id, cid), n); err != nil {
		return nil, err
	}
	return n, nil
}

// ListNotices returns a lease's notices, newest first.
func (r *LeaseOpsRepo) ListNotices(ctx context.Context, leaseID uuid.UUID) ([]domain.LeaseNotice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+noticeCols+` FROM lease_notices WHERE lease_id = $1 AND company_id = $2
		ORDER BY created_at DESC`, leaseID, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LeaseNotice, error) {
		var n domain.LeaseNotice
		err := scanNotice(row, &n)
		return n, err
	})
}

// MarkNoticeSent records when and how a notice was served. A notice is marked once.
func (r *LeaseOpsRepo) MarkNoticeSent(ctx context.Context, id uuid.UUID, via string, at time.Time) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE lease_notices SET sent_at = $1, sent_via = $2 WHERE id = $3 AND company_id = $4 AND sent_at IS NULL`,
		at, via, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		_ = r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM lease_notices WHERE id = $1 AND company_id = $2)`, id, cid).Scan(&exists)
		if exists {
			return ErrInvalidTransition
		}
		return pgx.ErrNoRows
	}
	return nil
}

// ── Alerts ────────────────────────────────────────────────────────────────────

// EndingLease is an active lease that ends (or whose Ejari registration expires) soon.
type EndingLease struct {
	ID               uuid.UUID
	EndDate          time.Time
	EjariExpiry      *time.Time
	TenantName       string
	PropertyName     string
	Agent            *uuid.UUID
	RentNoticeServed bool // a rent-change or non-renewal notice was sent during this term
}

// EndingLeases returns active leases ending, or with an Ejari registration expiring, on
// or before `until`, and not ended more than a week ago.
func (r *LeaseOpsRepo) EndingLeases(ctx context.Context, today, until time.Time) ([]EndingLease, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.end_date, l.ejari_expiry, COALESCE(t.full_name_en,''), COALESCE(p.name,''),
		       (SELECT u.id FROM users u WHERE u.id = COALESCE(l.agent_id, l.created_by) AND u.company_id = l.company_id AND u.is_active),
		       EXISTS (SELECT 1 FROM lease_notices n WHERE n.lease_id = l.id AND n.sent_at IS NOT NULL
		                  AND n.kind IN ('rent_change','non_renewal','renewal_terms') AND n.sent_at >= l.start_date)
		  FROM leases l
		  LEFT JOIN tenants t ON t.id = l.tenant_id
		  LEFT JOIN rental_properties p ON p.id = l.property_id
		 WHERE l.company_id = $1 AND l.status = 'active'
		   AND ((l.end_date <= $2 AND l.end_date >= $3::date - 7)
		     OR (l.ejari_expiry IS NOT NULL AND l.ejari_expiry <= $2 AND l.ejari_expiry >= $3::date - 7))`,
		cid, until, today)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (EndingLease, error) {
		var e EndingLease
		err := row.Scan(&e.ID, &e.EndDate, &e.EjariExpiry, &e.TenantName, &e.PropertyName, &e.Agent, &e.RentNoticeServed)
		return e, err
	})
}

// ── Rent reminders ────────────────────────────────────────────────────────────

// Reminder kinds.
const (
	ReminderUpcoming = "upcoming"
	ReminderOverdue  = "overdue"
)

// DuePayment is a payment a tenant may be reminded about.
type DuePayment struct {
	PaymentID    uuid.UUID
	Amount       float64
	Currency     string
	DueDate      time.Time
	TenantName   string
	TenantEmail  string
	TenantPhone  string
	PropertyName string
	Kind         string // upcoming | overdue
}

// DuePayments returns unpaid payments of active leases whose tenants accept reminders:
// due within daysBefore days ("upcoming"), or 1 to 30 days late ("overdue"). Payments
// covered by a cheque in hand, in the bank or cleared are left out.
func (r *LeaseOpsRepo) DuePayments(ctx context.Context, today time.Time, daysBefore int) ([]DuePayment, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT pay.id, pay.amount::float8, COALESCE(NULLIF(TRIM(pay.currency),''),'AED'), pay.due_date,
		       COALESCE(t.full_name_en,''), COALESCE(t.email,''), COALESCE(NULLIF(t.phone_wa,''), t.phone, ''),
		       COALESCE(p.name,''),
		       CASE WHEN pay.due_date < $2::date THEN 'overdue' ELSE 'upcoming' END
		  FROM payments pay
		  JOIN leases l ON l.id = pay.lease_id AND l.company_id = pay.company_id
		  JOIN tenants t ON t.id = l.tenant_id
		  LEFT JOIN rental_properties p ON p.id = l.property_id
		 WHERE pay.company_id = $1 AND l.status = 'active' AND t.rent_reminders
		   AND pay.status IN ('pending','overdue','failed')
		   AND pay.due_date BETWEEN $2::date - 30 AND $2::date + $3::int
		   AND NOT EXISTS (SELECT 1 FROM cheques ch WHERE ch.payment_id = pay.id
		                    AND ch.status IN ('received','on_hold','deposited','cleared'))
		 ORDER BY pay.due_date`, cid, today, daysBefore)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (DuePayment, error) {
		var d DuePayment
		err := row.Scan(&d.PaymentID, &d.Amount, &d.Currency, &d.DueDate, &d.TenantName, &d.TenantEmail, &d.TenantPhone,
			&d.PropertyName, &d.Kind)
		return d, err
	})
}

// ClaimReminder records that a reminder of this kind goes out on this channel for the
// payment, and reports whether it was not already sent (or attempted).
func (r *LeaseOpsRepo) ClaimReminder(ctx context.Context, paymentID uuid.UUID, kind, channel string, on time.Time) (uuid.UUID, bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	var id uuid.UUID
	err = r.db.QueryRow(ctx, `
		INSERT INTO payment_reminders (company_id, payment_id, reminder_type, reminder_date, delivery_method)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (payment_id, reminder_type, delivery_method) DO NOTHING RETURNING id`,
		cid, paymentID, kind, on, channel).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	return id, err == nil, err
}

// FinishReminder records the outcome of a claimed reminder.
func (r *LeaseOpsRepo) FinishReminder(ctx context.Context, id uuid.UUID, status, errMsg string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE payment_reminders SET delivery_status = $1, delivery_error = $2, updated_at = NOW(),
		       sent_at = CASE WHEN $1 = 'sent' THEN NOW() ELSE sent_at END
		 WHERE id = $3 AND company_id = $4`, status, errMsg, id, cid)
	return err
}

// PaymentReminders lists the reminders sent for a lease's payments, newest first.
func (r *LeaseOpsRepo) PaymentReminders(ctx context.Context, leaseID uuid.UUID) ([]domain.PaymentReminder, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT pr.id, pr.company_id, pr.payment_id, pr.reminder_type, pr.reminder_date, pr.sent_at, pr.delivery_method,
		       pr.delivery_status, pr.delivery_error, pr.created_at, pr.updated_at
		  FROM payment_reminders pr JOIN payments pay ON pay.id = pr.payment_id
		 WHERE pay.lease_id = $1 AND pr.company_id = $2 ORDER BY pr.created_at DESC LIMIT 200`, leaseID, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.PaymentReminder, error) {
		var p domain.PaymentReminder
		err := row.Scan(&p.ID, &p.CompanyID, &p.PaymentID, &p.ReminderType, &p.ReminderDate, &p.SentAt, &p.DeliveryMethod,
			&p.DeliveryStatus, &p.DeliveryError, &p.CreatedAt, &p.UpdatedAt)
		return p, err
	})
}
