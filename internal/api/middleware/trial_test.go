package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type fakeCompanies struct {
	company   domain.Company
	gets      int
	endTrials int
}

func (f *fakeCompanies) GetByID(_ context.Context, id uuid.UUID) (*domain.Company, error) {
	f.gets++
	c := f.company
	c.ID = id
	return &c, nil
}

func (f *fakeCompanies) EndTrial(context.Context, uuid.UUID) error {
	f.endTrials++
	f.company.OnTrial = false
	f.company.Plan = "community"
	return nil
}

func trialApp(store CompanyStore, companyID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &jwt.Token{Claims: jwt.MapClaims{"company_id": companyID.String()}})
		return c.Next()
	})
	app.Use(TrialCheck(store))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString(c.Locals("plan").(string)) })
	return app
}

func TestTrialCheckCachesCompanyLookups(t *testing.T) {
	store := &fakeCompanies{company: domain.Company{IsActive: true, Plan: "pro"}}
	app := trialApp(store, uuid.New())

	for i := 0; i < 5; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("request %d: status %v err %v", i, resp, err)
		}
	}
	if store.gets != 1 {
		t.Fatalf("GetByID called %d times for 5 requests, want 1", store.gets)
	}
}

func TestTrialCheckSuspendedCompanyBlocked(t *testing.T) {
	store := &fakeCompanies{company: domain.Company{IsActive: false, Plan: "pro"}}
	resp, err := trialApp(store, uuid.New()).Test(httptest.NewRequest("GET", "/", nil))
	if err != nil || resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("suspended company: status %v err %v, want 403", resp, err)
	}
}

func TestTrialCheckDowngradesExpiredTrialOnce(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	store := &fakeCompanies{company: domain.Company{IsActive: true, OnTrial: true, TrialEndsAt: &past, Plan: "pro"}}
	app := trialApp(store, uuid.New())

	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("request %d: %v %v", i, resp, err)
		}
	}
	if store.endTrials != 1 {
		t.Fatalf("EndTrial called %d times, want 1", store.endTrials)
	}
}

func TestCompanyCacheExpires(t *testing.T) {
	cc := newCompanyCache()
	now := time.Now()
	cc.now = func() time.Time { return now }
	id := uuid.New()
	cc.put(domain.Company{ID: id, IsActive: true})
	if _, ok := cc.get(id); !ok {
		t.Fatal("fresh entry should hit")
	}
	now = now.Add(companyCacheTTL + time.Second)
	if _, ok := cc.get(id); ok {
		t.Fatal("expired entry should miss")
	}
}
