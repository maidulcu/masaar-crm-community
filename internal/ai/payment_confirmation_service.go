package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type PaymentConfirmationService struct{}

func NewPaymentConfirmationService(
	paymentRepo *repo.PaymentRepo,
	confirmationRepo *repo.PaymentConfirmationRepo,
	leaseRepo *repo.LeaseRepo,
	tenantRepo *repo.TenantRepo,
	propertyRepo *repo.RentalPropertyRepo,
	companySettingsRepo *repo.CompanySettingsRepo,
	emailService *email.Service,
) *PaymentConfirmationService {
	return &PaymentConfirmationService{}
}

func (s *PaymentConfirmationService) CreateAndSendConfirmation(ctx context.Context, paymentID uuid.UUID) error {
	return nil
}

func (s *PaymentConfirmationService) SendPendingConfirmations(ctx context.Context, companyID uuid.UUID) error {
	return nil
}
