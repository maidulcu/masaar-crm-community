package jobs

import (
	"context"
	"fmt"
	"html"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// Property management jobs (#86): the daily list of cheques to deposit, lease end, Ejari
// and rent-notice alerts, and rent reminders to tenants.

type (
	alertClaimer interface {
		ClaimAlert(ctx context.Context, entityType string, entityID uuid.UUID, expiry time.Time, daysBefore int) (bool, error)
		AdminIDs(ctx context.Context) ([]uuid.UUID, error)
	}
	chequeLister interface {
		ToDeposit(ctx context.Context, until time.Time) ([]domain.Cheque, error)
		LeaseAgent(ctx context.Context, leaseID uuid.UUID) (*uuid.UUID, error)
	}
	leaseAlertStore interface {
		Settings(ctx context.Context) (*repo.PropertySettings, error)
		EndingLeases(ctx context.Context, today, until time.Time) ([]repo.EndingLease, error)
	}
	reminderStore interface {
		Settings(ctx context.Context) (*repo.PropertySettings, error)
		DuePayments(ctx context.Context, today time.Time, daysBefore int) ([]repo.DuePayment, error)
		ClaimReminder(ctx context.Context, paymentID uuid.UUID, kind, channel string, on time.Time) (uuid.UUID, bool, error)
		FinishReminder(ctx context.Context, id uuid.UUID, status, errMsg string) error
	}
	optOuts interface {
		EmailOptedOut(ctx context.Context, email string) (bool, error)
		PhoneOptedOut(ctx context.Context, phone, channel string) (bool, error)
	}
	templateSender interface {
		IsConfigured() bool
		SendTemplate(ctx context.Context, to, templateName, languageCode string, params []string) (string, error)
	}
)

// clock is the time source of the property jobs; tests replace it.
var clock = time.Now

// localDay returns today's date in the company's timezone (as a UTC midnight) and the
// local hour.
func localDay(co domain.CompanyLocale, now time.Time) (time.Time, int) {
	t := now.In(location(co.Timezone))
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), t.Hour()
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func daysUntil(today, t time.Time) int {
	return int(dateOnly(t).Sub(today).Hours() / 24)
}

// pointFor returns the smallest threshold at or above daysLeft, or -1 when daysLeft is
// beyond every threshold. A threshold missed while the server was down is caught up.
func pointFor(daysLeft int, points []int) int {
	p := -1
	for _, d := range points {
		if d >= 0 && daysLeft <= d && (p == -1 || d < p) {
			p = d
		}
	}
	return p
}

func recipients(ctx context.Context, store alertClaimer, agent *uuid.UUID, admins *[]uuid.UUID) ([]uuid.UUID, error) {
	if *admins == nil {
		ids, err := store.AdminIDs(ctx)
		if err != nil {
			return nil, err
		}
		*admins = append([]uuid.UUID{}, ids...)
	}
	out := append([]uuid.UUID{}, *admins...)
	if agent != nil {
		dup := false
		for _, id := range out {
			dup = dup || id == *agent
		}
		if !dup {
			out = append(out, *agent)
		}
	}
	return out, nil
}

// ChequeDeposits sends, once a day from 7:00 local time, the list of cheques in hand that
// are due to be banked: each lease's agent gets theirs, admins get all of them.
func ChequeDeposits(cheques chequeLister, alerts alertClaimer, n *Notifier) Job {
	return Job{
		Name:     "cheque_deposits",
		Interval: time.Hour,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			today, hour := localDay(co, clock())
			if hour < 7 {
				return 0, nil
			}
			due, err := cheques.ToDeposit(ctx, today)
			if err != nil || len(due) == 0 {
				return 0, err
			}
			// One digest per company and day.
			claimed, err := alerts.ClaimAlert(ctx, "cheque_deposits", co.CompanyID, today, 0)
			if err != nil || !claimed {
				return 0, err
			}
			byUser := map[uuid.UUID][]domain.Cheque{}
			admins, err := alerts.AdminIDs(ctx)
			if err != nil {
				return 0, err
			}
			for _, c := range due {
				agent, err := cheques.LeaseAgent(ctx, c.LeaseID)
				if err != nil {
					return 0, err
				}
				if agent != nil {
					byUser[*agent] = append(byUser[*agent], c)
				}
			}
			for _, a := range admins {
				byUser[a] = due
			}
			for user, list := range byUser {
				if err := n.Notify(ctx, user, depositDigest(list)); err != nil {
					return len(due), err
				}
			}
			return len(due), nil
		},
	}
}

func depositDigest(list []domain.Cheque) Message {
	totals := map[string]float64{}
	var lines []string
	for _, c := range list {
		totals[c.Currency] += c.Amount
		lines = append(lines, fmt.Sprintf("%s · %s %s · %s · %s", c.ChequeDate.Format("2 Jan"), c.Currency,
			fmtAmount(c.Amount, c.Currency), c.ChequeNumber, orDash(c.TenantName)))
	}
	var sums []string
	for cur, v := range totals {
		sums = append(sums, cur+" "+fmtAmount(v, cur))
	}
	sort.Strings(sums)
	return Message{
		Type:    "cheques_to_deposit",
		TitleEN: fmt.Sprintf("%d cheque(s) to deposit (%s)", len(list), strings.Join(sums, ", ")),
		TitleAR: fmt.Sprintf("%d شيك للإيداع (%s)", len(list), strings.Join(sums, "، ")),
		BodyEN:  strings.Join(lines, "\n"),
		BodyAR:  strings.Join(lines, "\n"),
		Data:    `{"page":"cheques","status":"received"}`,
		Email:   true,
	}
}

func fmtAmount(v float64, currency string) string {
	d, ok := jurisdiction.CurrencyDecimals(currency)
	if !ok {
		d = 2
	}
	return fmt.Sprintf("%.*f", d, v)
}

// EjariAlertDays are the alert points before an Ejari (tenancy registration) expires.
var EjariAlertDays = []int{30, 7, 0}

// LeaseAlerts warns the lease's agent and the admins, once per threshold, as a lease's
// end approaches (thresholds from the company settings, by default 120/90/60/30 days),
// when the last day to serve a rent-change notice nears without one being recorded, and
// before the Ejari registration expires.
func LeaseAlerts(store leaseAlertStore, alerts alertClaimer, n *Notifier) Job {
	return Job{
		Name:     "lease_alerts",
		Interval: time.Hour,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			today, _ := localDay(co, clock())
			settings, err := store.Settings(ctx)
			if err != nil {
				return 0, err
			}
			horizon := settings.RentNoticeDays + 14
			for _, d := range settings.LeaseAlertDays {
				if d > horizon {
					horizon = d
				}
			}
			leases, err := store.EndingLeases(ctx, today, today.AddDate(0, 0, horizon))
			if err != nil {
				return 0, err
			}
			var admins []uuid.UUID
			sent := 0
			notify := func(l repo.EndingLease, m Message) error {
				to, err := recipients(ctx, alerts, l.Agent, &admins)
				if err != nil {
					return err
				}
				m.Data = fmt.Sprintf(`{"lease_id":%q}`, l.ID.String())
				m.Email = true
				for _, u := range to {
					if err := n.Notify(ctx, u, m); err != nil {
						return err
					}
				}
				sent++
				return nil
			}
			for _, l := range leases {
				who := l.TenantName + " · " + l.PropertyName
				left := daysUntil(today, l.EndDate)
				end := l.EndDate.Format("2 Jan 2006")

				if left >= 0 {
					if p := pointFor(left, settings.LeaseAlertDays); p >= 0 {
						ok, err := alerts.ClaimAlert(ctx, "lease_end", l.ID, dateOnly(l.EndDate), p)
						if err != nil {
							return sent, err
						}
						if ok {
							if err := notify(l, Message{
								Type:    "lease_ending",
								TitleEN: fmt.Sprintf("Lease ends in %d days (%s)", left, end),
								TitleAR: fmt.Sprintf("ينتهي عقد الإيجار خلال %d يوماً (%s)", left, l.EndDate.Format("2006-01-02")),
								BodyEN:  who + "\nStart the renewal from the lease page.",
								BodyAR:  who + "\nابدأ التجديد من صفحة العقد.",
							}); err != nil {
								return sent, err
							}
						}
					}
				}

				// The rent-change notice deadline: alert two weeks ahead, once, unless a
				// notice has been served for this term.
				if !l.RentNoticeServed && settings.RentNoticeDays > 0 && left >= 0 && left <= settings.RentNoticeDays+14 {
					ok, err := alerts.ClaimAlert(ctx, "rent_notice", l.ID, dateOnly(l.EndDate), settings.RentNoticeDays)
					if err != nil {
						return sent, err
					}
					if ok {
						deadline := l.EndDate.AddDate(0, 0, -settings.RentNoticeDays)
						m := Message{
							Type:    "rent_notice_deadline",
							TitleEN: "Rent-change notice due by " + deadline.Format("2 Jan 2006"),
							TitleAR: "موعد إشعار تعديل الإيجار قبل " + deadline.Format("2006-01-02"),
							BodyEN: fmt.Sprintf("%s\nTo change the rent on renewal, serve notice at least %d days before the lease ends (%s). No notice has been recorded yet.",
								who, settings.RentNoticeDays, end),
							BodyAR: fmt.Sprintf("%s\nلتعديل الإيجار عند التجديد، يجب تقديم الإشعار قبل %d يوماً على الأقل من انتهاء العقد (%s). لم يُسجَّل أي إشعار بعد.",
								who, settings.RentNoticeDays, l.EndDate.Format("2006-01-02")),
						}
						if deadline.Before(today) {
							m.TitleEN = "Rent-change notice deadline has passed (" + deadline.Format("2 Jan 2006") + ")"
							m.TitleAR = "انقضى موعد إشعار تعديل الإيجار (" + deadline.Format("2006-01-02") + ")"
						}
						if err := notify(l, m); err != nil {
							return sent, err
						}
					}
				}

				if l.EjariExpiry != nil {
					eleft := daysUntil(today, *l.EjariExpiry)
					p := 0
					if eleft >= 0 {
						p = pointFor(eleft, EjariAlertDays)
					}
					if p >= 0 && eleft >= -7 {
						ok, err := alerts.ClaimAlert(ctx, "ejari", l.ID, dateOnly(*l.EjariExpiry), p)
						if err != nil {
							return sent, err
						}
						if ok {
							title, titleAR := fmt.Sprintf("Ejari registration expires in %d days", eleft), fmt.Sprintf("ينتهي تسجيل إيجاري خلال %d يوماً", eleft)
							if eleft < 0 {
								title, titleAR = "Ejari registration has expired", "انتهى تسجيل إيجاري"
							}
							if err := notify(l, Message{Type: "ejari_expiry", TitleEN: title, TitleAR: titleAR,
								BodyEN: who, BodyAR: who}); err != nil {
								return sent, err
							}
						}
					}
				}
			}
			return sent, nil
		},
	}
}

// RentReminders reminds tenants by email and WhatsApp a few days before rent is due and
// once when it is late, between 9:00 and 20:00 local time, when the company has switched
// reminders on. Payments covered by a cheque in hand are skipped, and so are tenants who
// opted out of the channel (recorded as "skipped"). WhatsApp uses the approved template
// named in the settings, with the parameters: tenant name, amount, due date, property.
func RentReminders(store reminderStore, consent optOuts, mail mailer, wa templateSender) Job {
	return Job{
		Name:     "rent_reminders",
		Interval: time.Hour,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			today, hour := localDay(co, clock())
			if hour < 9 || hour >= 20 {
				return 0, nil
			}
			settings, err := store.Settings(ctx)
			if err != nil || !settings.RentRemindersEnabled {
				return 0, err
			}
			due, err := store.DuePayments(ctx, today, settings.RentReminderDaysBefore)
			if err != nil {
				return 0, err
			}
			sent := 0
			for _, p := range due {
				for _, ch := range []string{"email", "whatsapp"} {
					addr := p.TenantEmail
					if ch == "whatsapp" {
						addr = p.TenantPhone
					}
					if addr == "" {
						continue
					}
					if ch == "email" && (mail == nil || !mail.IsConfigured()) {
						continue
					}
					if ch == "whatsapp" && (wa == nil || !wa.IsConfigured() || settings.RentReminderWATemplate == "") {
						continue
					}
					id, claimed, err := store.ClaimReminder(ctx, p.PaymentID, p.Kind, ch, today)
					if err != nil {
						return sent, err
					}
					if !claimed {
						continue
					}
					status, reason := "sent", ""
					if out, err := optedOut(ctx, consent, ch, p); err != nil {
						return sent, err
					} else if out {
						status, reason = "skipped", "the tenant opted out of "+ch
					} else if err := deliverReminder(ctx, ch, p, settings, mail, wa); err != nil {
						status, reason = "failed", err.Error()
						log.Printf("rent reminder %s for payment %s: %v", ch, p.PaymentID, err)
					} else {
						sent++
					}
					if err := store.FinishReminder(ctx, id, status, reason); err != nil {
						return sent, err
					}
				}
			}
			return sent, nil
		},
	}
}

func optedOut(ctx context.Context, consent optOuts, ch string, p repo.DuePayment) (bool, error) {
	if consent == nil {
		return false, nil
	}
	if p.TenantPhone != "" {
		out, err := consent.PhoneOptedOut(ctx, p.TenantPhone, ch)
		if err != nil || out {
			return out, err
		}
	}
	if ch == "email" {
		return consent.EmailOptedOut(ctx, p.TenantEmail)
	}
	return false, nil
}

func reminderText(p repo.DuePayment) (subject, en, ar string) {
	amount := p.Currency + " " + fmtAmount(p.Amount, p.Currency)
	due := p.DueDate.Format("2 January 2006")
	dueAR := p.DueDate.Format("2006-01-02")
	if p.Kind == repo.ReminderOverdue {
		subject = "Rent overdue: " + amount
		en = fmt.Sprintf("Dear %s,\n\nOur records show that the rent of %s for %s, due on %s, has not been received yet. Please arrange payment, or contact us if you have already paid.",
			p.TenantName, amount, p.PropertyName, due)
		ar = fmt.Sprintf("عزيزي/عزيزتي %s،\n\nتشير سجلاتنا إلى أن الإيجار البالغ %s عن %s والمستحق بتاريخ %s لم يُستلم بعد. يرجى ترتيب الدفع، أو التواصل معنا إذا كنتم قد دفعتم.",
			p.TenantName, amount, p.PropertyName, dueAR)
	} else {
		subject = "Rent due " + due + ": " + amount
		en = fmt.Sprintf("Dear %s,\n\nThis is a reminder that the rent of %s for %s is due on %s.",
			p.TenantName, amount, p.PropertyName, due)
		ar = fmt.Sprintf("عزيزي/عزيزتي %s،\n\nنذكّركم بأن الإيجار البالغ %s عن %s يستحق بتاريخ %s.",
			p.TenantName, amount, p.PropertyName, dueAR)
	}
	return subject, en, ar
}

func deliverReminder(ctx context.Context, ch string, p repo.DuePayment, s *repo.PropertySettings, mail mailer, wa templateSender) error {
	subject, en, ar := reminderText(p)
	if ch == "email" {
		return mail.Send(&domain.EmailHistory{
			ToEmail: p.TenantEmail,
			Subject: subject,
			Body:    en + "\n\n" + ar,
			HTMLBody: `<p dir="ltr">` + strings.ReplaceAll(html.EscapeString(en), "\n", "<br>") + `</p><hr><p dir="rtl">` +
				strings.ReplaceAll(html.EscapeString(ar), "\n", "<br>") + `</p>`,
			RelatedTo: "rent_reminder",
		})
	}
	lang := s.RentReminderWALanguage
	if lang == "" {
		lang = "en"
	}
	_, err := wa.SendTemplate(ctx, p.TenantPhone, s.RentReminderWATemplate, lang, []string{
		p.TenantName, p.Currency + " " + fmtAmount(p.Amount, p.Currency), p.DueDate.Format("2006-01-02"), p.PropertyName,
	})
	return err
}
