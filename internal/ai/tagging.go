package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type TaggingService struct{}

func NewTaggingService(
	aiClient *Client,
	leadRepo *repo.LeadRepo,
	waRepo *repo.WhatsAppRepo,
	tagRepo *repo.LeadTagRepo,
	contactRepo *repo.ContactRepo,
) *TaggingService {
	return &TaggingService{}
}

func (s *TaggingService) AutoTagFromMessage(ctx context.Context, contactID uuid.UUID, messageBody string) error {
	return nil
}
