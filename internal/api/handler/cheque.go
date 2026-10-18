package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/cheque"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jobs"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/mediastore"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// ChequeHandler manages the post-dated cheque register.
type ChequeHandler struct {
	cheques  *repo.ChequeRepo
	audit    *repo.AuditLogRepo
	notifier *jobs.Notifier
	admins   func(ctx context.Context) ([]uuid.UUID, error)
	store    *mediastore.Store // scans; may be nil
	maxBytes int64
}

func NewChequeHandler(cheques *repo.ChequeRepo, audit *repo.AuditLogRepo, notifier *jobs.Notifier,
	admins func(ctx context.Context) ([]uuid.UUID, error), store *mediastore.Store, maxMB int64) *ChequeHandler {
	if maxMB <= 0 {
		maxMB = 10
	}
	return &ChequeHandler{cheques: cheques, audit: audit, notifier: notifier, admins: admins, store: store, maxBytes: maxMB << 20}
}

type chequeBody struct {
	ChequeNumber string     `json:"cheque_number"`
	BankName     string     `json:"bank_name"`
	DrawerName   string     `json:"drawer_name"`
	Amount       float64    `json:"amount"`
	ChequeDate   string     `json:"cheque_date"` // YYYY-MM-DD
	PaymentID    *uuid.UUID `json:"payment_id"`
	Notes        string     `json:"notes"`
}

func (b *chequeBody) cheque() (*domain.Cheque, error) {
	d, err := time.Parse("2006-01-02", b.ChequeDate)
	if err != nil {
		return nil, errors.New("cheque_date must be YYYY-MM-DD")
	}
	if strings.TrimSpace(b.ChequeNumber) == "" || b.Amount <= 0 {
		return nil, errors.New("cheque_number and a positive amount are required")
	}
	return &domain.Cheque{ChequeNumber: strings.TrimSpace(b.ChequeNumber), BankName: strings.TrimSpace(b.BankName),
		DrawerName: strings.TrimSpace(b.DrawerName), Amount: b.Amount, ChequeDate: d, PaymentID: b.PaymentID, Notes: b.Notes}, nil
}

func parseDay(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (h *ChequeHandler) log(c *fiber.Ctx, action string, id uuid.UUID, diff fiber.Map) {
	if actor := localsUserPtr(c); actor != nil && h.audit != nil {
		h.audit.Log(c.Context(), *actor, action, "cheque", id, diff)
	}
}

// List godoc
// @Summary      Cheque register
// @Description  Filters: status, lease_id, from and to (cheque date, YYYY-MM-DD, inclusive).
// @Tags         Cheques
// @Produce      json
// @Success      200  {array}  domain.Cheque
// @Security     BearerAuth
// @Router       /cheques [get]
func (h *ChequeHandler) List(c *fiber.Ctx) error {
	f := repo.ChequeFilter{Status: c.Query("status")}
	if f.Status != "" && !cheque.ValidStatus(f.Status) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown status"})
	}
	var err error
	if f.From, err = parseDay(c.Query("from")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from must be YYYY-MM-DD"})
	}
	if f.To, err = parseDay(c.Query("to")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to must be YYYY-MM-DD"})
	}
	if s := c.Query("lease_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lease_id"})
		}
		f.LeaseID = &id
	}
	list, err := h.cheques.List(c.Context(), f)
	if err != nil {
		return serverError(c, err)
	}
	if list == nil {
		list = []domain.Cheque{}
	}
	return c.JSON(list)
}

// ForLease godoc
// @Summary      A lease's cheques
// @Tags         Cheques
// @Produce      json
// @Param        id  path  string  true  "Lease ID"
// @Success      200  {array}  domain.Cheque
// @Security     BearerAuth
// @Router       /leases/{id}/cheques [get]
func (h *ChequeHandler) ForLease(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	list, err := h.cheques.List(c.Context(), repo.ChequeFilter{LeaseID: &id})
	if err != nil {
		return serverError(c, err)
	}
	if list == nil {
		list = []domain.Cheque{}
	}
	return c.JSON(list)
}

// Schedule godoc
// @Summary      Create a lease's cheque schedule
// @Description  Splits the rent for the lease term into `count` cheques at equal month intervals from first_date (default: the lease start), numbered upwards from first_number, and creates a pending payment for each. Refused when the lease already has cheques.
// @Tags         Cheques
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Lease ID"
// @Param        body  body  object{count=int,first_date=string,first_number=string,bank_name=string,drawer_name=string}  true  "Schedule"
// @Success      201  {array}  domain.Cheque
// @Security     BearerAuth
// @Router       /leases/{id}/cheques/schedule [post]
func (h *ChequeHandler) Schedule(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Count       int    `json:"count"`
		FirstDate   string `json:"first_date"`
		FirstNumber string `json:"first_number"`
		BankName    string `json:"bank_name"`
		DrawerName  string `json:"drawer_name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	in := repo.ScheduleInput{Count: body.Count, FirstNumber: body.FirstNumber, BankName: strings.TrimSpace(body.BankName),
		DrawerName: strings.TrimSpace(body.DrawerName)}
	if d, err := parseDay(body.FirstDate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "first_date must be YYYY-MM-DD"})
	} else if d != nil {
		in.FirstDate = *d
	}
	list, err := h.cheques.CreateSchedule(c.Context(), id, in, localsUserPtr(c))
	if err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditCreate, id, fiber.Map{"cheque_schedule": len(list)})
	return c.Status(fiber.StatusCreated).JSON(list)
}

// Create godoc
// @Summary      Record a cheque for a lease
// @Tags         Cheques
// @Accept       json
// @Produce      json
// @Param        id    path  string      true  "Lease ID"
// @Param        body  body  chequeBody  true  "Cheque"
// @Success      201  {object}  domain.Cheque
// @Security     BearerAuth
// @Router       /leases/{id}/cheques [post]
func (h *ChequeHandler) Create(c *fiber.Ctx) error {
	leaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body chequeBody
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	ch, err := body.cheque()
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	ch.LeaseID = leaseID
	if err := h.cheques.Create(c.Context(), ch, localsUserPtr(c)); err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditCreate, ch.ID, fiber.Map{"cheque_number": ch.ChequeNumber, "amount": ch.Amount})
	got, err := h.cheques.Get(c.Context(), ch.ID)
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(got)
}

// Update godoc
// @Summary      Correct a cheque's details
// @Description  Only while the cheque has not been banked (received or on hold).
// @Tags         Cheques
// @Accept       json
// @Produce      json
// @Param        id    path  string      true  "Cheque ID"
// @Param        body  body  chequeBody  true  "Cheque"
// @Success      200  {object}  domain.Cheque
// @Security     BearerAuth
// @Router       /cheques/{id} [patch]
func (h *ChequeHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body chequeBody
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	ch, err := body.cheque()
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	ch.ID = id
	if err := h.cheques.UpdateDetails(c.Context(), ch); err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, id, fiber.Map{"cheque_number": ch.ChequeNumber, "amount": ch.Amount, "cheque_date": body.ChequeDate})
	got, err := h.cheques.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(got)
}

// SetStatus godoc
// @Summary      Move a cheque through its lifecycle
// @Description  received → deposited → cleared, or bounced; also on_hold and returned. A cleared cheque marks its payment received; a bounced cheque marks it failed, notifies the lease's agent and the admins and can add a returned-cheque charge (penalty_amount) as a new payment.
// @Tags         Cheques
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Cheque ID"
// @Param        body  body  object{status=string,date=string,reason=string,penalty_amount=number}  true  "Change"
// @Success      200  {object}  domain.Cheque
// @Failure      409  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /cheques/{id}/status [post]
func (h *ChequeHandler) SetStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status        string  `json:"status"`
		Date          string  `json:"date"`
		Reason        string  `json:"reason"`
		PenaltyAmount float64 `json:"penalty_amount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	on := time.Now().UTC().Truncate(24 * time.Hour)
	if d, err := parseDay(body.Date); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "date must be YYYY-MM-DD"})
	} else if d != nil {
		on = *d
	}
	if body.PenaltyAmount < 0 || (body.PenaltyAmount > 0 && body.Status != cheque.Bounced) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "a penalty can only be added when a cheque bounces"})
	}
	prev, err := h.cheques.Transition(c.Context(), id, body.Status, on, strings.TrimSpace(body.Reason))
	if err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, id, fiber.Map{"status": fiber.Map{"from": prev, "to": body.Status}, "date": on.Format("2006-01-02"), "reason": body.Reason})
	got, err := h.cheques.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if body.Status == cheque.Bounced {
		if body.PenaltyAmount > 0 {
			if pid, err := h.cheques.AddPenalty(c.Context(), id, body.PenaltyAmount, on, localsUserPtr(c)); err != nil {
				log.Printf("cheques: penalty for %s: %v", id, err)
			} else {
				h.log(c, repo.AuditCreate, id, fiber.Map{"penalty_payment": pid, "amount": body.PenaltyAmount})
			}
		}
		h.notifyBounce(c.Context(), got)
	}
	return c.JSON(got)
}

func (h *ChequeHandler) notifyBounce(ctx context.Context, ch *domain.Cheque) {
	if h.notifier == nil {
		return
	}
	to := map[uuid.UUID]bool{}
	if agent, err := h.cheques.LeaseAgent(ctx, ch.LeaseID); err == nil && agent != nil {
		to[*agent] = true
	}
	if h.admins != nil {
		if ids, err := h.admins(ctx); err == nil {
			for _, id := range ids {
				to[id] = true
			}
		}
	}
	amount := fmt.Sprintf("%s %.2f", ch.Currency, jurisdiction.RoundMoney(ch.Amount, ch.Currency))
	m := jobs.Message{
		Type:    "cheque_bounced",
		TitleEN: "Cheque bounced: " + ch.ChequeNumber + " (" + amount + ")",
		TitleAR: "شيك مرتجع: " + ch.ChequeNumber + " (" + amount + ")",
		BodyEN:  strings.TrimSpace(ch.TenantName + " · " + ch.PropertyName + "\n" + ch.BounceReason),
		BodyAR:  strings.TrimSpace(ch.TenantName + " · " + ch.PropertyName + "\n" + ch.BounceReason),
		Data:    fmt.Sprintf(`{"lease_id":%q,"cheque_id":%q}`, ch.LeaseID, ch.ID),
		Email:   true,
	}
	for id := range to {
		if err := h.notifier.Notify(ctx, id, m); err != nil {
			log.Printf("cheques: notify bounce to %s: %v", id, err)
		}
	}
}

// Replace godoc
// @Summary      Replace a cheque
// @Description  Records the tenant's new cheque for the same payment; the old one (bounced, or not yet banked) becomes "replaced".
// @Tags         Cheques
// @Accept       json
// @Produce      json
// @Param        id    path  string      true  "Cheque ID"
// @Param        body  body  chequeBody  true  "New cheque"
// @Success      201  {object}  domain.Cheque
// @Security     BearerAuth
// @Router       /cheques/{id}/replace [post]
func (h *ChequeHandler) Replace(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body chequeBody
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	nc, err := body.cheque()
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.cheques.Replace(c.Context(), id, nc, localsUserPtr(c)); err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, id, fiber.Map{"replaced_by": nc.ID, "cheque_number": nc.ChequeNumber})
	got, err := h.cheques.Get(c.Context(), nc.ID)
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(got)
}

var scanTypes = map[string]bool{"image/jpeg": true, "image/png": true, "application/pdf": true}

// UploadScan godoc
// @Summary      Attach a scan of a cheque (JPEG, PNG or PDF)
// @Tags         Cheques
// @Accept       multipart/form-data
// @Param        id    path      string  true  "Cheque ID"
// @Param        file  formData  file    true  "Scan"
// @Success      204
// @Security     BearerAuth
// @Router       /cheques/{id}/scan [post]
func (h *ChequeHandler) UploadScan(c *fiber.Ctx) error {
	if h.store == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "file storage is not available"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if _, err := h.cheques.Get(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}
	if file.Size > h.maxBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file is too large"})
	}
	f, err := file.Open()
	if err != nil {
		return serverError(c, err)
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if !scanTypes[http.DetectContentType(head)] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the scan must be a JPEG, PNG or PDF file"})
	}
	key, _, err := h.store.Save(io.MultiReader(bytes.NewReader(head), f), h.maxBytes)
	if err != nil {
		if errors.Is(err, mediastore.ErrTooLarge) {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file is too large"})
		}
		return serverError(c, err)
	}
	old, err := h.cheques.SetScan(c.Context(), id, key)
	if err != nil {
		_ = h.store.Remove(key)
		return serverError(c, err)
	}
	if old != "" {
		if err := h.store.Remove(old); err != nil {
			log.Printf("cheques: remove old scan: %v", err)
		}
	}
	h.log(c, repo.AuditUpdate, id, fiber.Map{"scan": "uploaded"})
	return c.SendStatus(fiber.StatusNoContent)
}

// Scan godoc
// @Summary      Download a cheque's scan
// @Tags         Cheques
// @Param        id  path  string  true  "Cheque ID"
// @Success      200
// @Security     BearerAuth
// @Router       /cheques/{id}/scan [get]
func (h *ChequeHandler) Scan(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	key, err := h.cheques.ScanKey(c.Context(), id)
	if err != nil || key == "" || h.store == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}
	f, err := h.store.Open(key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		return serverError(c, err)
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return serverError(c, err)
	}
	ctype := http.DetectContentType(head[:n])
	if !scanTypes[ctype] {
		ctype = fiber.MIMEOctetStream
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return serverError(c, err)
	}
	c.Set(fiber.HeaderContentType, ctype)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.SendStream(f, int(st.Size()))
}
