package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type fakeConsent struct {
	withdrawn map[uuid.UUID]bool
	recorded  []domain.Consent
}

func (f *fakeConsent) RecordConsent(_ context.Context, c *domain.Consent) error {
	f.recorded = append(f.recorded, *c)
	return nil
}
func (f *fakeConsent) MayContact(_ context.Context, id uuid.UUID, _ string) (bool, error) {
	return !f.withdrawn[id], nil
}
func (f *fakeConsent) EmailOptedOut(context.Context, string) (bool, error) { return false, nil }

func TestOptedOutBlocksMessaging(t *testing.T) {
	out, in := uuid.New(), uuid.New()
	store := &fakeConsent{withdrawn: map[uuid.UUID]bool{out: true}}
	app := fiber.New()
	app.Post("/send/:id", func(c *fiber.Ctx) error {
		if blocked, err := optedOut(c, store, uuid.MustParse(c.Params("id")), "whatsapp"); blocked {
			return err
		}
		return c.SendStatus(fiber.StatusCreated)
	})
	for id, want := range map[uuid.UUID]int{out: fiber.StatusForbidden, in: fiber.StatusCreated} {
		resp, _ := app.Test(httptest.NewRequest("POST", "/send/"+id.String(), nil))
		if resp.StatusCode != want {
			t.Errorf("contact %s: status %d, want %d", id, resp.StatusCode, want)
		}
	}
}

func TestRecordConsentChoices(t *testing.T) {
	store := &fakeConsent{}
	recordConsentChoices(context.Background(), store, uuid.New(),
		map[string]bool{"whatsapp": true, "email": false, "fax": true}, "public_form", nil, nil)
	if len(store.recorded) != 2 {
		t.Fatalf("recorded %d choices, want 2 (unknown channel ignored)", len(store.recorded))
	}
	for _, c := range store.recorded {
		want := map[string]string{"whatsapp": "granted", "email": "withdrawn"}[c.Channel]
		if c.Status != want || c.Source != "public_form" {
			t.Errorf("%s = %+v", c.Channel, c)
		}
	}
}

func TestPermitBlocked(t *testing.T) {
	app := fiber.New()
	listings := map[string]*domain.Listing{
		"dubai-missing": {Emirate: "Dubai"},
		"dubai-ok":      {Emirate: "Dubai", PermitNumber: "123"},
		"sharjah":       {Emirate: "Sharjah"},
	}
	app.Get("/:k", func(c *fiber.Ctx) error {
		if blocked, err := permitBlocked(c, listings[c.Params("k")]); blocked {
			return err
		}
		return c.SendStatus(fiber.StatusOK)
	})
	for k, want := range map[string]int{"dubai-missing": 422, "dubai-ok": 200, "sharjah": 200} {
		resp, _ := app.Test(httptest.NewRequest("GET", "/"+k, nil))
		if resp.StatusCode != want {
			t.Errorf("%s: %d, want %d", k, resp.StatusCode, want)
		}
	}
}

func TestEraseRequiresConfirmation(t *testing.T) {
	h := NewPrivacyHandler(nil, nil, nil)
	app := fiber.New()
	app.Post("/contacts/:id/erase", h.Erase)
	status, _ := post(t, app, "/contacts/"+uuid.NewString()+"/erase", map[string]any{"reason": "request"})
	if status != fiber.StatusUnprocessableEntity {
		t.Fatalf("erase without confirm: %d", status)
	}
}
