package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// ExpiryAlertDays are the alert points before an expiry date; 0 means "has expired".
var ExpiryAlertDays = []int{30, 7, 0}

type expiryStore interface {
	ListExpiring(ctx context.Context, until time.Time) ([]repo.Expiring, error)
	ClaimAlert(ctx context.Context, entityType string, entityID uuid.UUID, expiry time.Time, daysBefore int) (bool, error)
	AdminIDs(ctx context.Context) ([]uuid.UUID, error)
}

var expiryLabels = map[string]struct{ en, ar string }{
	"listing_permit": {"Advertising permit", "تصريح الإعلان"},
	"brn":            {"Broker registration (BRN)", "تسجيل الوسيط (BRN)"},
	"trade_license":  {"Trade licence", "الرخصة التجارية"},
	"kyc_document":   {"Client identity document", "وثيقة هوية العميل"},
}

// alertPoint returns the alert threshold that applies when daysLeft days remain, or -1
// when it is too early. A missed threshold (server down that day) is caught up: the
// smallest threshold at or above daysLeft applies, and each is sent once.
func alertPoint(daysLeft int) int {
	if daysLeft < 0 {
		return 0
	}
	point := -1
	for _, d := range ExpiryAlertDays {
		if d > 0 && daysLeft <= d && (point == -1 || d < point) {
			point = d
		}
	}
	return point
}

// ExpiryAlerts warns about listing permits, broker registrations and the trade licence
// 30 and 7 days before they expire, and once when they have expired.
func ExpiryAlerts(store expiryStore, n *Notifier) Job {
	return Job{
		Name:     "expiry_alerts",
		Interval: time.Hour,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			loc := location(co.Timezone)
			y, m, d := time.Now().In(loc).Date()
			today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
			items, err := store.ListExpiring(ctx, today.AddDate(0, 0, ExpiryAlertDays[0]))
			if err != nil {
				return 0, err
			}
			var admins []uuid.UUID
			sent := 0
			for _, it := range items {
				ey, em, ed := it.Expiry.Date()
				expiry := time.Date(ey, em, ed, 0, 0, 0, 0, time.UTC)
				daysLeft := int(expiry.Sub(today).Hours() / 24)
				point := alertPoint(daysLeft)
				if point < 0 {
					continue
				}
				claimed, err := store.ClaimAlert(ctx, it.EntityType, it.EntityID, expiry, point)
				if err != nil {
					return sent, err
				}
				if !claimed {
					continue
				}
				if admins == nil {
					if admins, err = store.AdminIDs(ctx); err != nil {
						return sent, err
					}
				}
				label := expiryLabels[it.EntityType]
				date := expiry.Format("2 Jan 2006")
				msg := Message{
					Type:    "expiry_alert",
					TitleEN: fmt.Sprintf("%s expires in %d days", label.en, daysLeft),
					TitleAR: fmt.Sprintf("%s ينتهي خلال %d يوماً", label.ar, daysLeft),
					BodyEN:  fmt.Sprintf("%s — %s expires on %s.", it.Label, label.en, date),
					BodyAR:  fmt.Sprintf("%s — ينتهي %s في %s.", it.Label, label.ar, date),
					Data:    fmt.Sprintf(`{"entity_type":%q,"entity_id":%q}`, it.EntityType, it.EntityID),
					Email:   true,
				}
				if daysLeft < 0 {
					msg.TitleEN = label.en + " has expired"
					msg.TitleAR = "انتهت صلاحية " + label.ar
					msg.BodyEN = fmt.Sprintf("%s — %s expired on %s.", it.Label, label.en, date)
					msg.BodyAR = fmt.Sprintf("%s — انتهت صلاحية %s في %s.", it.Label, label.ar, date)
				}
				for _, uid := range unique(append(append([]uuid.UUID{}, it.Notify...), admins...)) {
					if err := n.Notify(ctx, uid, msg); err != nil {
						return sent, err
					}
				}
				sent++
			}
			return sent, nil
		},
	}
}

func unique(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	out := ids[:0]
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
