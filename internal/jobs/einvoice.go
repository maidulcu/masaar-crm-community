package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/einvoice"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type einvoiceStore interface {
	Pending(ctx context.Context) ([]repo.PendingEInvoice, error)
	Decide(ctx context.Context, invoiceID uuid.UUID, status, reason string) (bool, error)
}

// ApplyEInvoiceDecision records a provider's decision and tells the person who issued the
// invoice (and whoever submitted it) when it was rejected.
func ApplyEInvoiceDecision(ctx context.Context, store einvoiceStore, n *Notifier, p repo.PendingEInvoice, res einvoice.Result) {
	if res.Status != einvoice.StatusAccepted && res.Status != einvoice.StatusRejected {
		return // still processing
	}
	changed, err := store.Decide(ctx, p.InvoiceID, res.Status, res.Reason)
	if err != nil {
		log.Printf("einvoice: record decision for %s: %v", p.InvoiceNo, err)
		return
	}
	if !changed || res.Status != einvoice.StatusRejected || n == nil {
		return
	}
	m := Message{
		Type:    "einvoice_rejected",
		TitleEN: "E-invoice rejected: " + p.InvoiceNo,
		TitleAR: "رُفضت الفاتورة الإلكترونية: " + p.InvoiceNo,
		BodyEN:  res.Reason,
		BodyAR:  res.Reason,
		Data:    fmt.Sprintf(`{"invoice_id":%q}`, p.InvoiceID.String()),
		Email:   true,
	}
	sent := map[uuid.UUID]bool{}
	for _, u := range []*uuid.UUID{p.Issuer, p.SubmittedBy} {
		if u != nil && !sent[*u] {
			sent[*u] = true
			if err := n.Notify(ctx, *u, m); err != nil {
				log.Printf("einvoice: notify %s: %v", *u, err)
			}
		}
	}
}

// EInvoiceStatus asks each provider about submitted e-invoices every five minutes and
// records accepted and rejected ones.
func EInvoiceStatus(store einvoiceStore, n *Notifier) Job {
	return Job{
		Name:     "einvoice_status",
		Interval: 5 * time.Minute,
		PerCompany: func(ctx context.Context, _ domain.CompanyLocale) (int, error) {
			pending, err := store.Pending(ctx)
			if err != nil {
				return 0, err
			}
			done := 0
			for _, p := range pending {
				conn, ok := einvoice.Lookup(p.Provider)
				if !ok {
					continue
				}
				res, err := conn.Status(ctx, p.ProviderRef)
				if err != nil {
					log.Printf("einvoice: status of %s: %v", p.InvoiceNo, err)
					continue
				}
				if res.Status == einvoice.StatusAccepted || res.Status == einvoice.StatusRejected {
					ApplyEInvoiceDecision(ctx, store, n, p, res)
					done++
				}
			}
			return done, nil
		},
	}
}
