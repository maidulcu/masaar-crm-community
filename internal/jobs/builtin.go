package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// ViewingReminderLead is how long before a viewing its agent is reminded.
const ViewingReminderLead = 60 * time.Minute

// RunLogRetention is how long job_runs rows are kept.
const RunLogRetention = 30 * 24 * time.Hour

type viewingStore interface {
	DueForReminder(ctx context.Context, withinMinutes int) ([]domain.Viewing, error)
	ClaimReminder(ctx context.Context, id uuid.UUID) (bool, error)
}

type offerStore interface {
	ExpireDue(ctx context.Context) ([]repo.ExpiredOffer, error)
}

type runPruner interface {
	PruneRuns(ctx context.Context, before time.Time) (int64, error)
}

// ViewingReminders reminds each viewing's agent, once, within ViewingReminderLead of
// the viewing.
func ViewingReminders(viewings viewingStore, n *Notifier) Job {
	return Job{
		Name:     "viewing_reminders",
		Interval: time.Minute,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			due, err := viewings.DueForReminder(ctx, int(ViewingReminderLead/time.Minute))
			if err != nil {
				return 0, err
			}
			loc := location(co.Timezone)
			sent := 0
			for _, v := range due {
				claimed, err := viewings.ClaimReminder(ctx, v.ID)
				if err != nil {
					return sent, err
				}
				if !claimed {
					continue // another worker got it
				}
				sent++
				if v.AgentID == nil {
					continue
				}
				who := ""
				if v.Contact != nil {
					who = v.Contact.FullName
				}
				at := v.ScheduledAt.In(loc).Format("15:04")
				what := v.ListingTitle
				if what == "" {
					what = v.Address
				}
				if err := n.Notify(ctx, *v.AgentID, Message{
					Type:    "viewing_reminder",
					TitleEN: "Upcoming viewing at " + at,
					TitleAR: "معاينة قادمة الساعة " + at,
					BodyEN:  fmt.Sprintf("Viewing with %s at %s: %s", orDash(who), at, orDash(what)),
					BodyAR:  fmt.Sprintf("معاينة مع %s الساعة %s: %s", orDash(who), at, orDash(what)),
					Data:    fmt.Sprintf(`{"viewing_id":%q}`, v.ID.String()),
					Email:   true,
				}); err != nil {
					return sent, err
				}
			}
			return sent, nil
		},
	}
}

// OfferExpiry moves offers past valid_until to 'expired' and tells their agents.
func OfferExpiry(offers offerStore, n *Notifier) Job {
	return Job{
		Name:     "offer_expiry",
		Interval: 5 * time.Minute,
		PerCompany: func(ctx context.Context, _ domain.CompanyLocale) (int, error) {
			expired, err := offers.ExpireDue(ctx)
			if err != nil {
				return 0, err
			}
			for _, o := range expired {
				if o.AgentID == nil {
					continue
				}
				if err := n.Notify(ctx, *o.AgentID, Message{
					Type:    "offer_expired",
					TitleEN: "Offer expired",
					TitleAR: "انتهت صلاحية العرض",
					BodyEN:  fmt.Sprintf("The offer from %s on %s has expired.", orDash(o.ContactName), orDash(o.ListingTitle)),
					BodyAR:  fmt.Sprintf("انتهت صلاحية عرض %s على %s.", orDash(o.ContactName), orDash(o.ListingTitle)),
					Data:    fmt.Sprintf(`{"offer_id":%q}`, o.ID.String()),
				}); err != nil {
					return len(expired), err
				}
			}
			return len(expired), nil
		},
	}
}

// PruneRunLog deletes job_runs rows older than RunLogRetention.
func PruneRunLog(store runPruner) Job {
	return Job{
		Name:     "job_runs_prune",
		Interval: 24 * time.Hour,
		Lease:    10 * time.Minute,
		System: func(ctx context.Context) (int, error) {
			n, err := store.PruneRuns(ctx, time.Now().Add(-RunLogRetention))
			return int(n), err
		},
	}
}

func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
