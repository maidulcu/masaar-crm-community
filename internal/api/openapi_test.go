package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/docs"
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/apiscope"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// maxUndocumented ratchets the routes still missing annotations down: lower it as handlers get
// documented, never raise it (scripts/openapi.sh -v lists them).
const maxUndocumented = 56

func committedOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(docs.OpenAPI, &doc); err != nil {
		t.Fatalf("docs/openapi.json: %v", err)
	}
	return doc
}

// The committed document matches the router: every operation exists, routes open to API keys
// carry their scope, and the conversion of the committed Swagger 2.0 reproduces it.
func TestOpenAPIMatchesRouter(t *testing.T) {
	doc := committedOpenAPI(t)
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("openapi = %v", doc["openapi"])
	}
	served := map[string]bool{}
	for _, r := range ServedRoutes() {
		served[routeKey(r.Method, r.Path)] = true
	}
	ops := map[string]map[string]any{}
	for path, item := range doc["paths"].(map[string]any) {
		for method, op := range item.(map[string]any) {
			k := routeKey(method, path)
			if !served[k] {
				t.Errorf("documented but not served: %s", k)
			}
			ops[k] = op.(map[string]any)
		}
	}
	intake := middleware.APIKeyRoute{Method: "POST", Path: "/webhooks/leads", Scope: apiscope.LeadCreate}
	for _, r := range append(append([]middleware.APIKeyRoute{}, apiKeyRoutes...), intake) {
		op, ok := ops[routeKey(r.Method, r.Path)]
		if !ok {
			t.Errorf("API-key route %s %s is not documented", r.Method, r.Path)
			continue
		}
		if op["x-api-key-scope"] != r.Scope || !strings.Contains(mustJSON(op["security"]), `"ApiKeyAuth":["`+r.Scope+`"]`) {
			t.Errorf("%s %s: scope not documented as %s", r.Method, r.Path, r.Scope)
		}
	}

	swagger, err := os.ReadFile("../../docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	built, rep, err := BuildOpenAPI(swagger)
	if err != nil {
		t.Fatal(err)
	}
	if string(built) != string(docs.OpenAPI) {
		t.Error("docs/openapi.json is out of date with docs/swagger.json: run scripts/openapi.sh")
	}
	if len(rep.Dropped) > 0 {
		t.Errorf("annotations for routes that do not exist: %v", rep.Dropped)
	}
	if len(rep.Undocumented) > maxUndocumented {
		t.Errorf("%d undocumented routes (max %d): annotate new handlers. Missing: %v", len(rep.Undocumented), maxUndocumented, rep.Undocumented)
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

// publicV1Routes lists exactly the /api/v1 routes router.go registers outside the auth group.
func TestPublicV1Routes(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`app\.(?:Get|Post|Put|Patch|Delete)\("(/api/v1/[^"]+)"`)
	found := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		found[m[1]] = true
		if !publicV1Routes[m[1]] {
			t.Errorf("%s is registered outside the auth group but missing from publicV1Routes", m[1])
		}
	}
	for p := range publicV1Routes {
		if !found[p] {
			t.Errorf("publicV1Routes lists %s, which router.go does not register on app", p)
		}
	}
}

func TestOperationID(t *testing.T) {
	cases := map[string]string{
		"GET /api/v1/leads/{id}/tags":                 "getLeadsByIdTags",
		"POST /webhooks/leads":                        "postWebhooksLeads",
		"GET /api/public/feeds/{portal}/listings.xml": "getPublicFeedsByPortalListingsXml",
		"DELETE /api/v1/settings/api-keys/{id}":       "deleteSettingsApiKeysById",
	}
	for in, want := range cases {
		m, p, _ := strings.Cut(in, " ")
		if got := operationID(m, p); got != want {
			t.Errorf("operationID(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestServeOpenAPI(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app, &Handlers{}, ws.NewHub(), &config.Config{JWTSecret: "x", AppEnv: "test"}, nil, nil, nil, nil, nil)
	resp, err := app.Test(httptest.NewRequest("GET", "/openapi.json", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") || string(body) != string(docs.OpenAPI) {
		t.Fatalf("GET /openapi.json = %d %s (%d bytes)", resp.StatusCode, resp.Header.Get("Content-Type"), len(body))
	}
}
