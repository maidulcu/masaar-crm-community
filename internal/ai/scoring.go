package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ScoringService struct{}

func NewScoringService(leadRepo *repo.LeadRepo, commRepo *repo.CommunicationHistoryRepo, tagRepo *repo.LeadTagRepo) *ScoringService {
	return &ScoringService{}
}

func (s *ScoringService) CalculateScore(ctx context.Context, leadID uuid.UUID) (int, error) {
	return 0, nil
}

func (s *ScoringService) UpdateLeadScore(ctx context.Context, leadID uuid.UUID) error {
	return nil
}

func (s *ScoringService) UpdateScoreOnStageChange(ctx context.Context, leadID uuid.UUID, newStage domain.LeadStage) error {
	return nil
}
