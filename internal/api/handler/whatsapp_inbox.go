package handler

import (
	"context"
	"log"
	"time"

	"github.com/maidulcu/masaar-crm/internal/bizhours"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/whatsapp"
)

// Inbox behaviour on inbound WhatsApp messages (#87): the 24-hour service window, opt-out
// and opt-in by reply ("STOP" / "إيقاف", "START" / "اشتراك"), and the business-hours
// auto-reply.

// AutoReplyEvery is the least time between two auto-replies on one conversation.
const AutoReplyEvery = 12 * time.Hour

type inboxDeps struct {
	broadcasts *repo.BroadcastRepo
	privacy    *repo.PrivacyRepo
	sender     interface {
		IsConfigured() bool
		SendMessage(ctx context.Context, to, body string) (string, error)
	}
	locale func(ctx context.Context) (*domain.CompanyLocale, error)
}

// UseInbox turns on the inbox behaviour above.
func (h *WhatsAppHandler) UseInbox(broadcasts *repo.BroadcastRepo, privacy *repo.PrivacyRepo, sender *whatsapp.Sender,
	locale func(ctx context.Context) (*domain.CompanyLocale, error)) *WhatsAppHandler {
	h.inbox = &inboxDeps{broadcasts: broadcasts, privacy: privacy, sender: sender, locale: locale}
	return h
}

// afterInbound runs once for each new inbound message.
func (h *WhatsAppHandler) afterInbound(ctx context.Context, thread *domain.WhatsAppThread, contact *domain.Contact, body string, isText bool) {
	in := h.inbox
	if in == nil {
		return
	}
	if isText {
		if out, optIn := whatsapp.OptKeyword(body); out || optIn {
			h.recordOptChoice(ctx, thread, contact, out)
			return
		}
	}
	h.maybeAutoReply(ctx, thread, contact)
}

func (h *WhatsAppHandler) recordOptChoice(ctx context.Context, thread *domain.WhatsAppThread, contact *domain.Contact, optOut bool) {
	in := h.inbox
	status, note := domain.ConsentGranted, "Replied START on WhatsApp"
	en := "You will receive our WhatsApp messages again. Reply STOP to stop them."
	ar := "ستصلك رسائلنا عبر واتساب مجدداً. أرسل «إيقاف» لإيقافها."
	if optOut {
		status, note = domain.ConsentWithdrawn, "Replied STOP on WhatsApp"
		en = "You will no longer receive our WhatsApp messages. Reply START to receive them again."
		ar = "لن تصلك رسائلنا عبر واتساب بعد الآن. أرسل «اشتراك» لاستلامها مجدداً."
	}
	if err := in.privacy.RecordConsent(ctx, &domain.Consent{ContactID: contact.ID, Channel: "whatsapp", Status: status,
		Source: "inbound_message", Note: note}); err != nil {
		log.Printf("whatsapp: record %s for %s: %v", status, contact.ID, err)
		return
	}
	// The customer just wrote, so the 24-hour window is open for this confirmation.
	h.sendInboxText(ctx, thread, contact, en, ar)
}

func (h *WhatsAppHandler) maybeAutoReply(ctx context.Context, thread *domain.WhatsAppThread, contact *domain.Contact) {
	in := h.inbox
	settings, err := in.broadcasts.InboxSettings(ctx)
	if err != nil || !settings.AutoReplyEnabled || (settings.AutoReplyEN == "" && settings.AutoReplyAR == "") {
		return
	}
	loc, schedule := time.UTC, bizhours.Default(nil)
	if l, err := in.locale(ctx); err == nil {
		rules, _ := jurisdiction.For(l.Country)
		schedule = bizhours.Default(rules.Weekend)
		if tz, err := time.LoadLocation(l.Timezone); err == nil {
			loc = tz
		}
	}
	if settings.Hours != nil {
		schedule = *settings.Hours
	}
	if schedule.OpenAt(time.Now(), loc) {
		return
	}
	ok, err := in.broadcasts.ClaimAutoReply(ctx, thread.ID, AutoReplyEvery)
	if err != nil || !ok {
		return
	}
	h.sendInboxText(ctx, thread, contact, settings.AutoReplyEN, settings.AutoReplyAR)
}

// sendInboxText sends a reply in the contact's language (falling back to the other) and
// records it on the thread.
func (h *WhatsAppHandler) sendInboxText(ctx context.Context, thread *domain.WhatsAppThread, contact *domain.Contact, en, ar string) {
	in := h.inbox
	if in.sender == nil || !in.sender.IsConfigured() {
		return
	}
	body := en
	if (contact.Language == "ar" && ar != "") || en == "" {
		body = ar
	}
	waid, err := in.sender.SendMessage(ctx, contact.PhoneWA, body)
	if err != nil {
		log.Printf("whatsapp: automatic reply to %s: %v", contact.ID, err)
		return
	}
	if _, err := h.wa.SaveMessage(ctx, &domain.WhatsAppMessage{ThreadID: thread.ID, Direction: domain.DirectionOutbound, Body: body, WAMessageID: waid}); err != nil {
		log.Printf("whatsapp: record automatic reply: %v", err)
	}
	if err := h.wa.UpdateThreadMeta(ctx, thread.ID); err != nil {
		log.Printf("whatsapp: thread meta for %s: %v", thread.ID, err)
	}
}

// applyBroadcastStatus forwards a delivery receipt to the broadcast that sent the message.
func (h *WhatsAppHandler) applyBroadcastStatus(ctx context.Context, waMessageID, status, errMsg string) {
	if h.inbox == nil {
		return
	}
	if _, err := h.inbox.broadcasts.ApplyStatus(ctx, waMessageID, status, errMsg); err != nil {
		log.Printf("whatsapp: broadcast receipt %s: %v", waMessageID, err)
	}
}
