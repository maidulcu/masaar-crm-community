package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/whatsapp"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

type PaymentReminderService struct{}

func NewPaymentReminderService(
	paymentRepo *repo.PaymentRepo,
	reminderRepo *repo.PaymentReminderRepo,
	leaseRepo *repo.LeaseRepo,
	tenantRepo *repo.TenantRepo,
	contactRepo *repo.ContactRepo,
	emailService *email.Service,
	whatsappSender *whatsapp.Sender,
	hub *ws.Hub,
) *PaymentReminderService {
	return &PaymentReminderService{}
}

func (s *PaymentReminderService) GenerateReminders(ctx context.Context, companyID uuid.UUID) error {
	return nil
}

func (s *PaymentReminderService) SendPendingReminders(ctx context.Context, companyID uuid.UUID) error {
	return nil
}
