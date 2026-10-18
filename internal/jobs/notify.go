package jobs

import (
	"context"
	"fmt"
	"html"
	"log"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// Message is a bilingual notification. The recipient's language preference picks the
// text; English is the fallback.
type Message struct {
	Type    string // notification type, e.g. "viewing_reminder"
	TitleEN string
	TitleAR string
	BodyEN  string
	BodyAR  string
	Data    string // JSON for in-app navigation, optional
	// Email also sends the message by email when the mail service is configured.
	Email bool
}

// The dependencies are interfaces so jobs can be tested without SMTP or websockets.
type (
	notificationStore interface {
		Create(ctx context.Context, n *domain.Notification) error
	}
	userFinder interface {
		FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	}
	mailer interface {
		IsConfigured() bool
		Send(e *domain.EmailHistory) error
	}
	pusher interface {
		SendToUser(userID string, e ws.Event)
	}
)

// Notifier tells a user about something a job did: an in-app notification, a live
// websocket event and, optionally, an email.
type Notifier struct {
	notifications notificationStore
	users         userFinder
	mail          mailer
	hub           pusher
}

// NewNotifier builds a Notifier. mail and hub may be nil.
func NewNotifier(notifications notificationStore, users userFinder, mail mailer, hub pusher) *Notifier {
	return &Notifier{notifications: notifications, users: users, mail: mail, hub: hub}
}

// Notify delivers m to userID, who must belong to the company bound to ctx. Delivery
// problems after the in-app notification is stored are logged, not returned, so one
// bad email address does not fail a whole job run.
func (n *Notifier) Notify(ctx context.Context, userID uuid.UUID, m Message) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	u, err := n.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("notify: find user: %w", err)
	}
	if u.CompanyID != cid || !u.IsActive {
		return nil // not this company's user, or deactivated: nothing to deliver
	}
	title, body, dir := m.TitleEN, m.BodyEN, "ltr"
	if u.LangPref == "ar" && m.TitleAR != "" {
		title, body, dir = m.TitleAR, m.BodyAR, "rtl"
	}
	note := &domain.Notification{UserID: userID, Type: m.Type, Title: title, Body: body, Data: m.Data}
	if err := n.notifications.Create(ctx, note); err != nil {
		return fmt.Errorf("notify: store: %w", err)
	}
	if n.hub != nil {
		// The web app prepends the payload to its notification list, so send the stored row.
		n.hub.SendToUser(userID.String(), ws.Event{Type: "notification", Payload: note})
	}
	if m.Email && n.mail != nil && n.mail.IsConfigured() && u.Email != "" {
		if err := n.mail.Send(&domain.EmailHistory{
			ToEmail:   u.Email,
			Subject:   title,
			Body:      body,
			HTMLBody:  `<p dir="` + dir + `">` + html.EscapeString(body) + "</p>",
			RelatedTo: m.Type,
		}); err != nil {
			log.Printf("jobs: email %s to user %s: %v", m.Type, userID, err)
		}
	}
	return nil
}
