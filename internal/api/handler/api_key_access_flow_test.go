package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// TestAPIKeyAccessFlow runs API-key requests through the real gate, handlers and repositories:
// writes need no signed-in user, are audited under the key, and stay inside the key's company.
func TestAPIKeyAccessFlow(t *testing.T) {
	e := newLeadEnv(t)
	ctx := context.Background()
	keys := repo.NewApiKeyRepo(e.pool)
	newKey := func(co uuid.UUID, name, scopes string) string {
		plain, hash, prefix, err := keys.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := keys.Create(ctx, co, name, hash, prefix, scopes); err != nil {
			t.Fatal(err)
		}
		return plain
	}
	writer := newKey(e.a, "writer", "contacts:write,leads:read,leads:write")
	reader := newKey(e.a, "reader", "leads:read")
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM api_keys WHERE company_id = ANY($1)`, []uuid.UUID{e.a, e.b})
	})

	audit := repo.NewAuditLogRepo(e.pool)
	leadRepo, contactRepo := repo.NewLeadRepo(e.pool), repo.NewContactRepo(e.pool)
	lh := NewLeadHandler(leadRepo, contactRepo, nil, nil, repo.NewLeadTagRepo(e.pool), ws.NewHub(), audit, e.disp, repo.NewPipelineStageRepo(e.pool))
	ch := NewContactHandler(contactRepo, audit)

	app := fiber.New()
	v1 := app.Group("/api/v1", middleware.ProblemDetails(), middleware.APIKeyAccess(keys, []middleware.APIKeyRoute{
		{Method: "GET", Path: "/api/v1/leads/:id", Scope: "leads:read"},
		{Method: "POST", Path: "/api/v1/leads", Scope: "leads:write"},
		{Method: "PATCH", Path: "/api/v1/leads/:id/stage", Scope: "leads:write"},
		{Method: "POST", Path: "/api/v1/contacts", Scope: "contacts:write"},
	}))
	v1.Get("/leads/:id", lh.Get)
	v1.Post("/leads", lh.Create)
	v1.Patch("/leads/:id/stage", lh.UpdateStage)
	v1.Post("/contacts", ch.Create)

	do := func(key, method, path string, body any) (int, map[string]any) {
		t.Helper()
		var rdr io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, rdr)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		return resp.StatusCode, out
	}

	code, contact := do(writer, "POST", "/api/v1/contacts", fiber.Map{"full_name": "Key Contact", "phone_wa": "+971509990001"})
	if code != 201 {
		t.Fatalf("create contact = %d %v", code, contact)
	}
	code, lead := do(writer, "POST", "/api/v1/leads", fiber.Map{"contact_id": contact["id"]})
	if code != 201 {
		t.Fatalf("create lead = %d %v", code, lead)
	}
	leadID := lead["id"].(string)
	if code, out := do(writer, "PATCH", "/api/v1/leads/"+leadID+"/stage", fiber.Map{"stage": "talking"}); code != 200 {
		t.Fatalf("stage = %d %v", code, out)
	}
	if code, out := do(reader, "GET", "/api/v1/leads/"+leadID, nil); code != 200 || out["stage"] != "talking" {
		t.Fatalf("read lead = %d %v", code, out)
	}
	if code, _ := do(reader, "PATCH", "/api/v1/leads/"+leadID+"/stage", fiber.Map{"stage": "fresh"}); code != 403 {
		t.Fatalf("reader stage change = %d, want 403", code)
	}

	// Writes are audited under the key, with no user.
	var keyed, unkeyed int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE api_key_id IS NOT NULL), count(*) FILTER (WHERE api_key_id IS NULL)
		FROM audit_logs WHERE company_id = $1`, e.a).Scan(&keyed, &unkeyed); err != nil {
		t.Fatal(err)
	}
	if keyed != 3 || unkeyed != 0 {
		t.Fatalf("audit rows: %d by key, %d without, want 3 and 0", keyed, unkeyed)
	}

	// Company B's key cannot see company A's lead.
	other := newKey(e.b, "other", "leads:read")
	if code, _ := do(other, "GET", "/api/v1/leads/"+leadID, nil); code != 404 {
		t.Fatalf("cross-company read = %d, want 404", code)
	}
}
