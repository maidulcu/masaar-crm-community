package repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/cheque"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// ChequeRepo stores post-dated rent cheques and keeps their payments in step.
type ChequeRepo struct {
	db *pgxpool.Pool
}

func NewChequeRepo(db *pgxpool.Pool) *ChequeRepo { return &ChequeRepo{db: db} }

const chequeCols = `ch.id, ch.lease_id, ch.payment_id, ch.cheque_number, ch.bank_name, ch.drawer_name, ch.amount::float8,
	ch.currency, ch.cheque_date, ch.status, ch.scan_key <> '', ch.deposited_at, ch.cleared_at, ch.bounced_at,
	ch.bounce_reason, ch.replaced_by, ch.notes, ch.created_at, ch.updated_at,
	COALESCE(t.full_name_en, ''), COALESCE(p.name, '')`

const chequeFrom = ` FROM cheques ch
	JOIN leases l ON l.id = ch.lease_id
	LEFT JOIN tenants t ON t.id = l.tenant_id
	LEFT JOIN rental_properties p ON p.id = l.property_id`

func scanCheque(row pgx.Row, c *domain.Cheque) error {
	return row.Scan(&c.ID, &c.LeaseID, &c.PaymentID, &c.ChequeNumber, &c.BankName, &c.DrawerName, &c.Amount,
		&c.Currency, &c.ChequeDate, &c.Status, &c.HasScan, &c.DepositedAt, &c.ClearedAt, &c.BouncedAt,
		&c.BounceReason, &c.ReplacedBy, &c.Notes, &c.CreatedAt, &c.UpdatedAt, &c.TenantName, &c.PropertyName)
}

func collectCheques(rows pgx.Rows) ([]domain.Cheque, error) {
	defer rows.Close()
	var out []domain.Cheque
	for rows.Next() {
		var c domain.Cheque
		if err := scanCheque(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get returns one of the company's cheques.
func (r *ChequeRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Cheque, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	c := &domain.Cheque{}
	if err := scanCheque(r.db.QueryRow(ctx, `SELECT `+chequeCols+chequeFrom+` WHERE ch.id = $1 AND ch.company_id = $2`, id, cid), c); err != nil {
		return nil, err
	}
	return c, nil
}

// ChequeFilter narrows the register.
type ChequeFilter struct {
	LeaseID *uuid.UUID
	Status  string
	From    *time.Time // cheque date, inclusive
	To      *time.Time // cheque date, inclusive
	Limit   int
}

// List returns cheques by cheque date.
func (r *ChequeRepo) List(ctx context.Context, f ChequeFilter) ([]domain.Cheque, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	where := []string{"ch.company_id = $1"}
	args := []any{cid}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.LeaseID != nil {
		add("ch.lease_id = $%d", *f.LeaseID)
	}
	if f.Status != "" {
		add("ch.status = $%d", f.Status)
	}
	if f.From != nil {
		add("ch.cheque_date >= $%d", *f.From)
	}
	if f.To != nil {
		add("ch.cheque_date <= $%d", *f.To)
	}
	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := r.db.Query(ctx, `SELECT `+chequeCols+chequeFrom+` WHERE `+strings.Join(where, " AND ")+
		fmt.Sprintf(` ORDER BY ch.cheque_date, ch.cheque_number LIMIT %d`, limit), args...)
	if err != nil {
		return nil, err
	}
	return collectCheques(rows)
}

// ScheduleInput describes a book of cheques for a lease.
type ScheduleInput struct {
	Count       int
	FirstDate   time.Time // zero: the lease start
	FirstNumber string    // numbered upwards from here, keeping leading zeros
	BankName    string
	DrawerName  string
}

// ErrScheduleExists is returned when a lease already has open cheques.
var ErrScheduleExists = errors.New("this lease already has cheques; add or replace them one by one")

// CreateSchedule splits the lease's rent into Count cheques and creates, for each, a
// pending payment by cheque. Everything happens in one transaction.
func (r *ChequeRepo) CreateSchedule(ctx context.Context, leaseID uuid.UUID, in ScheduleInput, createdBy *uuid.UUID) ([]domain.Cheque, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var start, end time.Time
	var rent float64
	var currency string
	err = tx.QueryRow(ctx, `SELECT start_date, end_date, monthly_rent::float8, COALESCE(NULLIF(TRIM(currency),''),'AED')
		FROM leases WHERE id = $1 AND company_id = $2 FOR UPDATE`, leaseID, cid).Scan(&start, &end, &rent, &currency)
	if err != nil {
		return nil, notFoundRef(err)
	}
	var existing int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM cheques WHERE lease_id = $1 AND status NOT IN ('replaced','returned')`,
		leaseID).Scan(&existing); err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrScheduleExists
	}
	decimals, ok := jurisdiction.CurrencyDecimals(currency)
	if !ok {
		decimals = 2
	}
	plan, err := cheque.Schedule(start, end, in.FirstDate, rent, in.Count, decimals)
	if err != nil {
		return nil, err
	}
	numbers := chequeNumbers(in.FirstNumber, len(plan))
	ids := make([]uuid.UUID, len(plan))
	for i, p := range plan {
		paymentID := uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO payments (id, company_id, lease_id, amount, currency, due_date, payment_method, payment_reference,
			                      status, notes, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, 'check', $7, 'pending', $8, $9, $9)`,
			paymentID, cid, leaseID, p.Amount, currency, p.Date, numbers[i],
			fmt.Sprintf("Cheque %d of %d", i+1, len(plan)), createdBy); err != nil {
			return nil, err
		}
		ids[i] = uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO cheques (id, company_id, lease_id, payment_id, cheque_number, bank_name, drawer_name, amount,
			                     currency, cheque_date, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			ids[i], cid, leaseID, paymentID, numbers[i], in.BankName, in.DrawerName, p.Amount, currency, p.Date, createdBy); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	lid := leaseID
	return r.List(ctx, ChequeFilter{LeaseID: &lid})
}

// chequeNumbers numbers n cheques from first upwards. A non-numeric first number is
// used for the first cheque only; the rest are left for the user to fill in.
func chequeNumbers(first string, n int) []string {
	out := make([]string, n)
	first = strings.TrimSpace(first)
	v, err := strconv.ParseUint(first, 10, 64)
	for i := range out {
		switch {
		case first == "":
			out[i] = fmt.Sprintf("TBC-%d", i+1)
		case err != nil:
			if i == 0 {
				out[i] = first
			} else {
				out[i] = fmt.Sprintf("TBC-%d", i+1)
			}
		default:
			out[i] = fmt.Sprintf("%0*d", len(first), v+uint64(i))
		}
	}
	return out
}

// Create records a single cheque. A payment, when given, must be one of the lease's.
func (r *ChequeRepo) Create(ctx context.Context, c *domain.Cheque, createdBy *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var currency string
	if err := r.db.QueryRow(ctx, `SELECT COALESCE(NULLIF(TRIM(currency),''),'AED') FROM leases WHERE id = $1 AND company_id = $2`,
		c.LeaseID, cid).Scan(&currency); err != nil {
		return notFoundRef(err)
	}
	if c.PaymentID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM payments WHERE id = $1 AND lease_id = $2 AND company_id = $3)`,
			*c.PaymentID, c.LeaseID, cid).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrForeignReference
		}
	}
	if c.Currency == "" {
		c.Currency = currency
	}
	c.ID = uuid.New()
	c.Status = cheque.Received
	return r.db.QueryRow(ctx, `
		INSERT INTO cheques (id, company_id, lease_id, payment_id, cheque_number, bank_name, drawer_name, amount, currency,
		                     cheque_date, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING created_at, updated_at`,
		c.ID, cid, c.LeaseID, c.PaymentID, c.ChequeNumber, c.BankName, c.DrawerName, c.Amount, c.Currency,
		c.ChequeDate, c.Notes, createdBy).Scan(&c.CreatedAt, &c.UpdatedAt)
}

// UpdateDetails corrects a cheque's number, bank, drawer, date, amount or notes while it
// has not been banked.
func (r *ChequeRepo) UpdateDetails(ctx context.Context, c *domain.Cheque) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var status string
	var paymentID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT status, payment_id FROM cheques WHERE id = $1 AND company_id = $2 FOR UPDATE`,
		c.ID, cid).Scan(&status, &paymentID); err != nil {
		return err
	}
	if status != cheque.Received && status != cheque.OnHold {
		return ErrInvalidTransition
	}
	if _, err := tx.Exec(ctx, `
		UPDATE cheques SET cheque_number = $1, bank_name = $2, drawer_name = $3, cheque_date = $4, amount = $5,
		       notes = $6, updated_at = NOW()
		 WHERE id = $7`, c.ChequeNumber, c.BankName, c.DrawerName, c.ChequeDate, c.Amount, c.Notes, c.ID); err != nil {
		return err
	}
	if paymentID != nil {
		if _, err := tx.Exec(ctx, `UPDATE payments SET payment_reference = $1, due_date = $2, amount = $3, updated_at = NOW()
			WHERE id = $4 AND status IN ('pending','overdue')`, c.ChequeNumber, c.ChequeDate, c.Amount, *paymentID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Transition moves a cheque to another status on the given date and keeps its payment in
// step: cleared marks the payment received, bounced marks it failed, re-presenting a
// bounced cheque makes it pending again. It returns the previous status.
func (r *ChequeRepo) Transition(ctx context.Context, id uuid.UUID, to string, on time.Time, reason string) (string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", err
	}
	if !cheque.ValidStatus(to) || to == cheque.Replaced {
		return "", ErrInvalidTransition // replacing goes through Replace
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var from, number string
	var paymentID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT status, cheque_number, payment_id FROM cheques WHERE id = $1 AND company_id = $2 FOR UPDATE`,
		id, cid).Scan(&from, &number, &paymentID); err != nil {
		return "", err
	}
	if !cheque.CanTransition(from, to) {
		return from, ErrInvalidTransition
	}
	if _, err := tx.Exec(ctx, `
		UPDATE cheques SET status = $1::text,
		       deposited_at  = CASE WHEN $1::text = 'deposited' THEN $2::date ELSE deposited_at END,
		       cleared_at    = CASE WHEN $1::text = 'cleared'   THEN $2::date ELSE cleared_at END,
		       bounced_at    = CASE WHEN $1::text = 'bounced'   THEN $2::date ELSE bounced_at END,
		       bounce_reason = CASE WHEN $1::text = 'bounced'   THEN $3 ELSE bounce_reason END,
		       updated_at = NOW()
		 WHERE id = $4`, to, on, reason, id); err != nil {
		return from, err
	}
	if paymentID != nil {
		var err error
		switch {
		case to == cheque.Cleared:
			_, err = tx.Exec(ctx, `UPDATE payments SET status = 'received', paid_date = $2, payment_method = 'check',
				payment_reference = $3, updated_at = NOW() WHERE id = $1`, *paymentID, on, number)
		case to == cheque.Bounced:
			_, err = tx.Exec(ctx, `UPDATE payments SET status = 'failed', updated_at = NOW() WHERE id = $1`, *paymentID)
		case from == cheque.Bounced && to == cheque.Deposited:
			_, err = tx.Exec(ctx, `UPDATE payments SET status = 'pending', updated_at = NOW() WHERE id = $1 AND status = 'failed'`, *paymentID)
		}
		if err != nil {
			return from, err
		}
	}
	return from, tx.Commit(ctx)
}

// Replace swaps a cheque (bounced, or not yet banked) for a new one that pays the same
// payment. The old cheque becomes "replaced" and points at the new one.
func (r *ChequeRepo) Replace(ctx context.Context, oldID uuid.UUID, nc *domain.Cheque, createdBy *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var status, currency string
	var leaseID uuid.UUID
	var paymentID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT status, lease_id, payment_id, currency FROM cheques WHERE id = $1 AND company_id = $2 FOR UPDATE`,
		oldID, cid).Scan(&status, &leaseID, &paymentID, &currency); err != nil {
		return err
	}
	if !cheque.CanTransition(status, cheque.Replaced) {
		return ErrInvalidTransition
	}
	nc.ID, nc.LeaseID, nc.PaymentID, nc.Currency, nc.Status = uuid.New(), leaseID, paymentID, currency, cheque.Received
	if err := tx.QueryRow(ctx, `
		INSERT INTO cheques (id, company_id, lease_id, payment_id, cheque_number, bank_name, drawer_name, amount, currency,
		                     cheque_date, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING created_at, updated_at`,
		nc.ID, cid, leaseID, paymentID, nc.ChequeNumber, nc.BankName, nc.DrawerName, nc.Amount, currency,
		nc.ChequeDate, nc.Notes, createdBy).Scan(&nc.CreatedAt, &nc.UpdatedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE cheques SET status = 'replaced', replaced_by = $1, updated_at = NOW() WHERE id = $2`,
		nc.ID, oldID); err != nil {
		return err
	}
	if paymentID != nil {
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'pending', due_date = $2, amount = $3, payment_reference = $4,
			updated_at = NOW() WHERE id = $1 AND status IN ('pending','overdue','failed')`,
			*paymentID, nc.ChequeDate, nc.Amount, nc.ChequeNumber); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// AddPenalty records a charge (for example a bounced-cheque fee) as a pending payment on
// the cheque's lease, due on the given date.
func (r *ChequeRepo) AddPenalty(ctx context.Context, chequeID uuid.UUID, amount float64, due time.Time, createdBy *uuid.UUID) (uuid.UUID, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	id := uuid.New()
	tag, err := r.db.Exec(ctx, `
		INSERT INTO payments (id, company_id, lease_id, amount, currency, due_date, payment_method, status, notes,
		                      created_by, updated_by)
		SELECT $1, ch.company_id, ch.lease_id, $2, ch.currency, $3, 'other', 'pending',
		       'Returned cheque charge (cheque ' || ch.cheque_number || ')', $4, $4
		  FROM cheques ch WHERE ch.id = $5 AND ch.company_id = $6`, id, amount, due, createdBy, chequeID, cid)
	if err != nil {
		return uuid.Nil, err
	}
	if tag.RowsAffected() == 0 {
		return uuid.Nil, pgx.ErrNoRows
	}
	return id, nil
}

// SetScan stores the key of a cheque's scan and returns the key it replaces.
func (r *ChequeRepo) SetScan(ctx context.Context, id uuid.UUID, key string) (string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", err
	}
	var old string
	err = r.db.QueryRow(ctx, `
		UPDATE cheques ch SET scan_key = $1, updated_at = NOW()
		  FROM (SELECT scan_key FROM cheques WHERE id = $2 AND company_id = $3 FOR UPDATE) prev
		 WHERE ch.id = $2 AND ch.company_id = $3 RETURNING prev.scan_key`, key, id, cid).Scan(&old)
	return old, err
}

// ScanKey returns the key of a cheque's scan ("" when there is none).
func (r *ChequeRepo) ScanKey(ctx context.Context, id uuid.UUID) (string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", err
	}
	var key string
	err = r.db.QueryRow(ctx, `SELECT scan_key FROM cheques WHERE id = $1 AND company_id = $2`, id, cid).Scan(&key)
	return key, err
}

// LeaseAgent returns the active user who looks after a lease (its agent, else its
// creator), or nil.
func (r *ChequeRepo) LeaseAgent(ctx context.Context, leaseID uuid.UUID) (*uuid.UUID, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	var who uuid.UUID
	err = r.db.QueryRow(ctx, `
		SELECT u.id FROM leases l
		  JOIN users u ON u.id = COALESCE(l.agent_id, l.created_by) AND u.company_id = l.company_id AND u.is_active
		 WHERE l.id = $1 AND l.company_id = $2`, leaseID, cid).Scan(&who)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &who, nil
}

// ToDeposit returns cheques in hand (received) dated on or before `until`.
func (r *ChequeRepo) ToDeposit(ctx context.Context, until time.Time) ([]domain.Cheque, error) {
	u := until
	return r.List(ctx, ChequeFilter{Status: cheque.Received, To: &u})
}
