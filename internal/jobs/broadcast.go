package jobs

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type broadcastStore interface {
	InboxSettings(ctx context.Context) (*repo.InboxSettings, error)
	ClaimBatch(ctx context.Context, n int) ([]repo.BroadcastSend, error)
	StillConsents(ctx context.Context, contactID uuid.UUID) (bool, error)
	FinishSend(ctx context.Context, s repo.BroadcastSend, status, waMessageID, errMsg string) error
	CompleteFinished(ctx context.Context) error
}

// renderParams fills {{name}} and {{first_name}} in a broadcast's template parameters.
func renderParams(params []string, name string) []string {
	first := strings.TrimSpace(name)
	if i := strings.IndexByte(first, ' '); i > 0 {
		first = first[:i]
	}
	out := make([]string, len(params))
	for i, p := range params {
		p = strings.ReplaceAll(p, "{{name}}", strings.TrimSpace(name))
		out[i] = strings.ReplaceAll(p, "{{first_name}}", first)
	}
	return out
}

// BroadcastSender sends queued broadcast messages, up to the company's per-minute rate.
// Each recipient's consent is checked again just before sending, so a "STOP" that
// arrives while a broadcast runs is respected.
func BroadcastSender(store broadcastStore, wa templateSender) Job {
	return Job{
		Name:     "whatsapp_broadcasts",
		Interval: time.Minute,
		PerCompany: func(ctx context.Context, _ domain.CompanyLocale) (int, error) {
			if wa == nil || !wa.IsConfigured() {
				return 0, nil
			}
			settings, err := store.InboxSettings(ctx)
			if err != nil {
				return 0, err
			}
			batch, err := store.ClaimBatch(ctx, settings.BroadcastPerMinute)
			if err != nil {
				return 0, err
			}
			sent := 0
			for _, s := range batch {
				ok, err := store.StillConsents(ctx, s.ContactID)
				if err != nil {
					return sent, err
				}
				if !ok {
					if err := store.FinishSend(ctx, s, "skipped", "", "opted_out"); err != nil {
						return sent, err
					}
					continue
				}
				waid, err := wa.SendTemplate(ctx, s.Phone, s.TemplateName, s.Language, renderParams(s.Params, s.Name))
				status, msg := "sent", ""
				if err != nil {
					status, msg = "failed", err.Error()
				} else {
					sent++
				}
				if err := store.FinishSend(ctx, s, status, waid, msg); err != nil {
					return sent, err
				}
			}
			return sent, store.CompleteFinished(ctx)
		},
	}
}
