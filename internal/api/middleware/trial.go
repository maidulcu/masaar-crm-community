package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

// companyCacheTTL bounds how long a company's status/plan is served from memory. Every
// authenticated request passes through TrialCheck, so without a cache each one costs a
// database round trip. A suspension or plan change takes effect within this window.
const companyCacheTTL = 15 * time.Second

// companyCacheMax caps the cache size; when exceeded, expired entries are dropped.
const companyCacheMax = 1024

// CompanyStore is the subset of the company repository TrialCheck needs.
type CompanyStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error)
	EndTrial(ctx context.Context, id uuid.UUID) error
}

type cachedCompany struct {
	company   domain.Company
	expiresAt time.Time
}

type companyCache struct {
	mu      sync.Mutex
	entries map[uuid.UUID]cachedCompany
	now     func() time.Time
}

func newCompanyCache() *companyCache {
	return &companyCache{entries: make(map[uuid.UUID]cachedCompany), now: time.Now}
}

func (cc *companyCache) get(id uuid.UUID) (domain.Company, bool) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	e, ok := cc.entries[id]
	if !ok || !cc.now().Before(e.expiresAt) {
		return domain.Company{}, false
	}
	return e.company, true
}

func (cc *companyCache) put(c domain.Company) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	now := cc.now()
	if len(cc.entries) >= companyCacheMax {
		for id, e := range cc.entries {
			if !now.Before(e.expiresAt) {
				delete(cc.entries, id)
			}
		}
	}
	cc.entries[c.ID] = cachedCompany{company: c, expiresAt: now.Add(companyCacheTTL)}
}

func (cc *companyCache) invalidate(id uuid.UUID) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.entries, id)
}

// TrialCheck reads company_id from JWT claims (not Locals) and verifies the
// company is active. If the trial has expired, it auto-downgrades to community.
// Sets plan and on_trial in locals for downstream handlers.
func TrialCheck(companies CompanyStore) fiber.Handler {
	cache := newCompanyCache()
	return func(c *fiber.Ctx) error {
		claims := ClaimsFromCtx(c)
		cidStr, ok := claims["company_id"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}
		companyID, err := uuid.Parse(cidStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}

		company, hit := cache.get(companyID)
		if !hit {
			fresh, err := companies.GetByID(c.Context(), companyID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "company not found"})
			}
			company = *fresh
			company.ID = companyID
			cache.put(company)
		}

		if !company.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account_suspended"})
		}

		// Auto-downgrade expired trials
		if company.OnTrial && company.TrialEndsAt != nil && company.TrialEndsAt.Before(time.Now()) {
			_ = companies.EndTrial(c.Context(), companyID)
			cache.invalidate(companyID)
			company.Plan = "community"
			company.OnTrial = false
		}

		c.Locals("plan", company.Plan)
		c.Locals("on_trial", company.OnTrial)
		return c.Next()
	}
}
