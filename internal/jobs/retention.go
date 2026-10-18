package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type retentionStore interface {
	ExpiredBankStatements(ctx context.Context, today time.Time) ([]repo.RetainedFile, error)
	MarkBankStatementDeleted(ctx context.Context, id uuid.UUID) error
	PurgeExpiredRecords(ctx context.Context, today time.Time) (int64, error)
}

// Retention enforces retention_until: bank statements past it lose their stored file and
// are soft-deleted; documents and payment confirmations past it are soft-deleted.
// removeFile deletes a stored bank-statement file by key (nil when storage is disabled).
func Retention(store retentionStore, removeFile func(key string) error) Job {
	return Job{
		Name:     "retention",
		Interval: 24 * time.Hour,
		Lease:    30 * time.Minute,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			y, m, d := time.Now().In(location(co.Timezone)).Date()
			today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
			files, err := store.ExpiredBankStatements(ctx, today)
			if err != nil {
				return 0, err
			}
			n := 0
			for _, f := range files {
				if f.Path != "" && removeFile != nil {
					if err := removeFile(f.Path); err != nil {
						continue // keep the row so the next run retries the file
					}
				}
				if err := store.MarkBankStatementDeleted(ctx, f.ID); err != nil {
					return n, err
				}
				n++
			}
			purged, err := store.PurgeExpiredRecords(ctx, today)
			return n + int(purged), err
		},
	}
}
