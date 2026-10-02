package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

func TestAPIErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"no rows", pgx.ErrNoRows, http.StatusNotFound},
		{"wrapped no rows", fmt.Errorf("get thing: %w", pgx.ErrNoRows), http.StatusNotFound},
		{"foreign reference", repo.ErrForeignReference, http.StatusNotFound},
		{"missing tenant", tenant.ErrMissing, http.StatusUnauthorized},
		{"unique violation", &pgconn.PgError{Code: "23505"}, http.StatusConflict},
		{"check violation", &pgconn.PgError{Code: "23514"}, http.StatusUnprocessableEntity},
		{"unknown", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if got, _ := apiError(tt.err); got != tt.status {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.status)
		}
	}
}

func TestServerErrorDoesNotLeakDetails(t *testing.T) {
	secret := `pq: relation "users" does not exist; password=hunter2 host=10.0.0.5`
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error { return serverError(c, errors.New(secret)) })
	app.Get("/up", func(c *fiber.Ctx) error { return upstreamError(c, errors.New(secret)) })
	app.Post("/bad", func(c *fiber.Ctx) error { return badRequest(c, errors.New(secret)) })

	for _, path := range []string{"/", "/up"} {
		resp, _ := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(b), "hunter2") || strings.Contains(string(b), "relation") {
			t.Fatalf("%s leaked internal error text: %s", path, b)
		}
	}
	resp, _ := app.Test(httptest.NewRequest(http.MethodPost, "/bad", nil))
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(b), "hunter2") || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("badRequest leaked or wrong status: %d %s", resp.StatusCode, b)
	}
}
