package bos24

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

type SyncResult struct {
	ListingsImported int
	ListingsUpdated  int
	InquiriesCreated int
}

type SyncService struct{}

func NewSyncService(
	bos24Repo *repo.BOS24IntegrationRepo,
	contactRepo *repo.ContactRepo,
	hub *ws.Hub,
) *SyncService {
	return &SyncService{}
}

func (s *SyncService) SyncAll(ctx context.Context, companyID uuid.UUID, apiKey string, lastSyncAt *time.Time) (*SyncResult, error) {
	return &SyncResult{}, nil
}
