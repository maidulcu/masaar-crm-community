package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/internal/apiscope"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// OpenAPI 3.1 document for the API (served at /openapi.json, committed as docs/openapi.json).
//
// Source of truth: the swag annotations on the handlers, generated as Swagger 2.0 by
// scripts/openapi.sh, then converted here. The conversion is checked against the live router:
// an annotation for a route that does not exist is dropped, paths written with or without the
// /api/v1 prefix are made absolute, and API-key access (scopes, problem details,
// Idempotency-Key) comes from apiKeyRoutes rather than from comments.

// Route is one method + Fiber path the server answers.
type Route struct{ Method, Path string }

// ServedRoutes returns the routes RegisterRoutes registers, without HEAD and middleware.
// Handlers are not called, so empty dependencies are enough.
func ServedRoutes() []Route {
	app := fiber.New()
	RegisterRoutes(app, &Handlers{}, ws.NewHub(), &config.Config{JWTSecret: "openapi", AppEnv: "openapi"}, nil, nil, nil, nil, nil)
	var out []Route
	seen := map[string]bool{}
	for _, r := range app.GetRoutes(true) {
		if r.Method == fiber.MethodHead || r.Method == "USE" || r.Method == fiber.MethodConnect || r.Method == fiber.MethodTrace {
			continue
		}
		k := r.Method + " " + r.Path
		if !seen[k] {
			seen[k] = true
			out = append(out, Route{r.Method, r.Path})
		}
	}
	return out
}

var (
	fiberParam   = regexp.MustCompile(`:[A-Za-z_][A-Za-z0-9_]*\??`)
	openapiParam = regexp.MustCompile(`\{[^}]+\}`)
)

// routeKey ignores parameter names and case: "GET /api/v1/leads/{}".
func routeKey(method, path string) string {
	p := fiberParam.ReplaceAllString(path, "{}")
	p = openapiParam.ReplaceAllString(p, "{}")
	return strings.ToUpper(method) + " " + strings.ToLower(strings.TrimSuffix(p, "/"))
}

// OpenAPIReport says how the annotations matched the router.
type OpenAPIReport struct {
	Documented   int      // operations in the output
	Dropped      []string // annotated operations with no such route
	Undocumented []string // served /api/v1 and /webhooks routes with no annotation
}

// BuildOpenAPI converts swag's Swagger 2.0 output into the published OpenAPI 3.1 document.
func BuildOpenAPI(swagger2 []byte) ([]byte, OpenAPIReport, error) {
	var rep OpenAPIReport
	var v2 map[string]any
	if err := json.Unmarshal(swagger2, &v2); err != nil {
		return nil, rep, fmt.Errorf("swagger 2.0: %w", err)
	}
	if v2["swagger"] != "2.0" {
		return nil, rep, fmt.Errorf("input is not Swagger 2.0")
	}
	doc := convert2to3(v2)

	served := map[string]Route{}
	for _, r := range ServedRoutes() {
		served[routeKey(r.Method, r.Path)] = r
	}
	keyScopes := map[string]string{}
	for _, r := range apiKeyRoutes {
		keyScopes[routeKey(r.Method, r.Path)] = r.Scope
	}
	keyScopes[routeKey("POST", "/webhooks/leads")] = apiscope.LeadCreate

	// Re-key every operation onto its real, absolute path.
	oldPaths, _ := doc["paths"].(map[string]any)
	paths := map[string]any{}
	documented := map[string]bool{}
	for p, item := range oldPaths {
		ops, _ := item.(map[string]any)
		for method, op := range ops {
			opm, ok := op.(map[string]any)
			if !ok || !isHTTPMethod(method) {
				continue
			}
			abs, found := "", false
			for _, cand := range []string{p, "/api/v1" + p} {
				if _, ok := served[routeKey(method, cand)]; ok {
					abs, found = cand, true
					break
				}
			}
			if !found {
				rep.Dropped = append(rep.Dropped, strings.ToUpper(method)+" "+p)
				continue
			}
			k := routeKey(method, abs)
			if documented[k] {
				continue // the same route annotated twice: keep the first
			}
			documented[k] = true
			if params, ok := item.(map[string]any)["parameters"]; ok {
				opm["parameters"] = append(asSlice(params), asSlice(opm["parameters"])...)
			}
			fixSecurity(opm, abs)
			ensurePathParams(opm, abs)
			opm["operationId"] = operationID(method, abs)
			if scope, ok := keyScopes[k]; ok {
				addAPIKeyAccess(opm, method, scope, abs == "/webhooks/leads")
			}
			ensureErrorResponse(opm)
			target, _ := paths[abs].(map[string]any)
			if target == nil {
				target = map[string]any{}
				paths[abs] = target
			}
			target[strings.ToLower(method)] = opm
		}
	}
	for k, r := range served {
		if !documented[k] && (strings.HasPrefix(r.Path, "/api/v1/") || strings.HasPrefix(r.Path, "/webhooks/")) {
			rep.Undocumented = append(rep.Undocumented, r.Method+" "+r.Path)
		}
	}
	sort.Strings(rep.Dropped)
	sort.Strings(rep.Undocumented)
	rep.Documented = len(documented)

	doc["paths"] = paths
	doc["openapi"] = "3.1.0"
	doc["servers"] = []any{map[string]any{"url": "/", "description": "This Masaar server"}}
	if info, ok := doc["info"].(map[string]any); ok {
		if lic, ok := info["license"].(map[string]any); ok && lic["name"] == "MIT" {
			lic["identifier"] = "MIT"
			delete(lic, "url") // 3.1: identifier and url are exclusive
		}
	}
	comps, _ := doc["components"].(map[string]any)
	if comps == nil {
		comps = map[string]any{}
		doc["components"] = comps
	}
	comps["securitySchemes"] = map[string]any{
		"BearerAuth": map[string]any{
			"type": "http", "scheme": "bearer", "bearerFormat": "JWT",
			"description": "Session token from POST /api/v1/auth/login.",
		},
		"ApiKeyAuth": map[string]any{
			"type": "http", "scheme": "bearer",
			"description": "API key (sk_live_...) from Settings → API Keys. Each operation lists the scope it needs; " +
				"routes without ApiKeyAuth refuse keys with 403.",
		},
	}
	schemas, _ := comps["schemas"].(map[string]any)
	if schemas == nil {
		schemas = map[string]any{}
		comps["schemas"] = schemas
	}
	schemas["Problem"] = problemSchema
	schemas["Error"] = map[string]any{
		"type": "object", "required": []any{"error"},
		"properties": map[string]any{"error": map[string]any{"type": "string", "examples": []any{"lead not found"}}},
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, rep, err
	}
	return append(out, '\n'), rep, nil
}

func isHTTPMethod(m string) bool {
	switch strings.ToUpper(m) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	}
	return false
}

func asSlice(v any) []any { s, _ := v.([]any); return s }

// publicV1Routes are the /api/v1 routes registered on app, outside the authenticated group
// (TestPublicV1Routes keeps this list in step with router.go).
var publicV1Routes = map[string]bool{
	"/api/v1/auth/login": true, "/api/v1/auth/register": true, "/api/v1/auth/refresh": true,
	"/api/v1/auth/magic-link/request": true, "/api/v1/auth/magic-link/verify": true,
	"/api/v1/auth/sms/request": true, "/api/v1/auth/sms/verify": true,
	"/api/v1/auth/forgot-password": true, "/api/v1/auth/reset-password": true,
	"/api/v1/auth/verify-totp": true, "/api/v1/auth/totp/enroll": true, "/api/v1/auth/totp/enroll/confirm": true,
}

// fixSecurity names the session scheme consistently (annotations use "Bearer" and
// "BearerAuth"): every /api/v1 operation outside publicV1Routes needs a session; operations
// elsewhere (/api/public, /webhooks) carry none unless API-key access is added later.
func fixSecurity(op map[string]any, path string) {
	if strings.HasPrefix(path, "/api/v1/") && !publicV1Routes[path] {
		op["security"] = []any{map[string]any{"BearerAuth": []any{}}}
	} else {
		op["security"] = []any{} // explicitly public (webhooks check their own tokens or signatures)
	}
}

// ensurePathParams declares any {param} of path the annotation left out.
func ensurePathParams(op map[string]any, path string) {
	have := map[string]bool{}
	for _, p := range asSlice(op["parameters"]) {
		if pm := asMap(p); pm["in"] == "path" {
			have[fmt.Sprint(pm["name"])] = true
		}
	}
	for _, m := range openapiParam.FindAllString(path, -1) {
		name := strings.Trim(m, "{}")
		if !have[name] {
			op["parameters"] = append(asSlice(op["parameters"]), map[string]any{
				"name": name, "in": "path", "required": true, "schema": map[string]any{"type": "string"},
			})
		}
	}
}

// ensureErrorResponse gives an operation without any 4xx response the API's usual error body.
func ensureErrorResponse(op map[string]any) {
	responses := asMap(op["responses"])
	for code := range responses {
		if strings.HasPrefix(code, "4") {
			return
		}
	}
	responses["4XX"] = map[string]any{
		"description": "Client error",
		"content": map[string]any{"application/json": map[string]any{
			"schema": map[string]any{"$ref": "#/components/schemas/Error"},
		}},
	}
}

var nonWord = regexp.MustCompile(`[^A-Za-z0-9]+`)

// operationID derives a stable id from method and path: GET /api/v1/leads/{id}/tags →
// getLeadsByIdTags.
func operationID(method, path string) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(method))
	for _, seg := range strings.Split(strings.TrimPrefix(strings.TrimPrefix(path, "/api/v1"), "/api"), "/") {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, "{") {
			b.WriteString("By")
			seg = strings.Trim(seg, "{}")
		}
		for _, w := range nonWord.Split(seg, -1) {
			if w != "" {
				b.WriteString(strings.ToUpper(w[:1]) + w[1:])
			}
		}
	}
	return b.String()
}

// addAPIKeyAccess documents an operation open to API keys: the scope (OpenAPI 3.1 allows role
// names for non-OAuth schemes), problem-details errors and, on writes, Idempotency-Key.
func addAPIKeyAccess(op map[string]any, method, scope string, keyOnly bool) {
	sec := []any{map[string]any{"ApiKeyAuth": []any{scope}}}
	if !keyOnly {
		sec = append([]any{map[string]any{"BearerAuth": []any{}}}, sec...)
	}
	op["security"] = sec
	op["x-api-key-scope"] = scope

	m := strings.ToUpper(method)
	if m == "POST" || m == "PATCH" {
		op["parameters"] = append(asSlice(op["parameters"]), map[string]any{
			"name": "Idempotency-Key", "in": "header", "required": false,
			"description": "API keys only: a retry with the same key and request returns the first response " +
				"(Idempotent-Replayed: true). Stored 24 hours.",
			"schema": map[string]any{"type": "string", "minLength": 1, "maxLength": 255},
		})
	}
	responses, _ := op["responses"].(map[string]any)
	if responses == nil {
		responses = map[string]any{}
		op["responses"] = responses
	}
	problem := map[string]any{"application/problem+json": map[string]any{
		"schema": map[string]any{"$ref": "#/components/schemas/Problem"},
	}}
	for code, desc := range map[string]string{
		"401": "Missing, unknown or revoked API key (WWW-Authenticate: Bearer error=\"invalid_token\")",
		"403": "The key lacks the scope (error \"insufficient_scope\"), or its company is suspended",
		"429": "Rate limit: 300 requests a minute per key",
	} {
		if _, exists := responses[code]; !exists {
			responses[code] = map[string]any{"description": desc, "content": problem}
		}
	}
}

// problemSchema is RFC 9457 problem details as the API sends them to API-key requests.
var problemSchema = map[string]any{
	"type":        "object",
	"description": "RFC 9457 problem details. The error field repeats detail for older clients.",
	"properties": map[string]any{
		"type":   map[string]any{"type": "string", "example": "about:blank"},
		"title":  map[string]any{"type": "string", "example": "Forbidden"},
		"status": map[string]any{"type": "integer", "example": 403},
		"detail": map[string]any{"type": "string", "example": "insufficient_scope"},
		"error":  map[string]any{"type": "string", "example": "insufficient_scope"},
	},
	"required": []any{"type", "title", "status"},
}

// convert2to3 turns swag's Swagger 2.0 document into OpenAPI 3.1 structure: definitions become
// components/schemas, body and form parameters become requestBody, response schemas get a
// media type. Security and servers are set by the caller.
func convert2to3(v2 map[string]any) map[string]any {
	consumes := strList(v2["consumes"], "application/json")
	produces := strList(v2["produces"], "application/json")
	doc := map[string]any{"info": v2["info"], "components": map[string]any{}}
	if tags, ok := v2["tags"]; ok {
		doc["tags"] = tags
	}
	schemas := map[string]any{}
	for name, s := range asMap(v2["definitions"]) {
		schemas[name] = schema31(s)
	}
	doc["components"].(map[string]any)["schemas"] = schemas

	paths := map[string]any{}
	for p, item := range asMap(v2["paths"]) {
		out := map[string]any{}
		for method, op := range asMap(item) {
			if method == "parameters" {
				out["parameters"] = params31(asSlice(op))
				continue
			}
			opm := asMap(op)
			if opm == nil {
				continue
			}
			out[method] = operation31(opm, strList(opm["consumes"], consumes...), strList(opm["produces"], produces...))
		}
		paths[p] = out
	}
	doc["paths"] = paths
	return doc
}

func operation31(op map[string]any, consumes, produces []string) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"summary", "description", "operationId", "tags", "deprecated", "security"} {
		if v, ok := op[k]; ok {
			out[k] = v
		}
	}
	var params []any
	var form map[string]any
	var formRequired []any
	for _, raw := range asSlice(op["parameters"]) {
		p := asMap(raw)
		switch p["in"] {
		case "body":
			content := map[string]any{}
			for _, ct := range consumes {
				content[ct] = map[string]any{"schema": schema31(p["schema"])}
			}
			body := map[string]any{"content": content, "required": p["required"] == true}
			if d, ok := p["description"]; ok {
				body["description"] = d
			}
			out["requestBody"] = body
		case "formData":
			if form == nil {
				form = map[string]any{}
			}
			form[p["name"].(string)] = paramSchema(p)
			if p["required"] == true {
				formRequired = append(formRequired, p["name"])
			}
		default:
			params = append(params, param31(p))
		}
	}
	if form != nil {
		ct := "application/x-www-form-urlencoded"
		for _, c := range consumes {
			if c == "multipart/form-data" {
				ct = c
			}
		}
		s := map[string]any{"type": "object", "properties": form}
		if len(formRequired) > 0 {
			s["required"] = formRequired
		}
		out["requestBody"] = map[string]any{"required": len(formRequired) > 0, "content": map[string]any{ct: map[string]any{"schema": s}}}
	}
	if len(params) > 0 {
		out["parameters"] = params
	}
	responses := map[string]any{}
	for code, raw := range asMap(op["responses"]) {
		r := asMap(raw)
		resp := map[string]any{"description": r["description"]}
		if resp["description"] == nil || resp["description"] == "" {
			resp["description"] = "Response " + code
		}
		if s, ok := r["schema"]; ok {
			content := map[string]any{}
			for _, ct := range produces {
				content[ct] = map[string]any{"schema": schema31(s)}
			}
			resp["content"] = content
		}
		if hs := asMap(r["headers"]); len(hs) > 0 {
			headers := map[string]any{}
			for name, h := range hs {
				hm := asMap(h)
				headers[name] = map[string]any{"description": hm["description"], "schema": paramSchema(hm)}
			}
			resp["headers"] = headers
		}
		responses[code] = resp
	}
	if len(responses) == 0 {
		responses["default"] = map[string]any{"description": "Response"}
	}
	out["responses"] = responses
	return out
}

func params31(ps []any) []any {
	out := make([]any, 0, len(ps))
	for _, p := range ps {
		out = append(out, param31(asMap(p)))
	}
	return out
}

func param31(p map[string]any) map[string]any {
	out := map[string]any{"name": p["name"], "in": p["in"], "schema": paramSchema(p)}
	if d, ok := p["description"]; ok {
		out["description"] = d
	}
	if p["required"] == true || p["in"] == "path" {
		out["required"] = true
	}
	return out
}

// paramSchema builds a schema from a 2.0 non-body parameter's inline type fields.
func paramSchema(p map[string]any) map[string]any {
	s := map[string]any{}
	for _, k := range []string{"type", "format", "items", "enum", "default", "minimum", "maximum",
		"minLength", "maxLength", "pattern", "example"} {
		if v, ok := p[k]; ok {
			s[k] = v
		}
	}
	if s["type"] == "file" {
		s["type"] = "string"
		s["contentMediaType"] = "application/octet-stream"
	}
	if items, ok := s["items"]; ok {
		s["items"] = schema31(items)
	}
	return schema31(s).(map[string]any)
}

// schema31 rewrites a 2.0 schema: #/definitions refs, x-nullable, file type, example.
func schema31(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			out[k] = schema31(child)
		}
		if ref, ok := out["$ref"].(string); ok {
			out["$ref"] = strings.Replace(ref, "#/definitions/", "#/components/schemas/", 1)
		}
		if out["type"] == "file" {
			out["type"] = "string"
			out["contentMediaType"] = "application/octet-stream"
		}
		if out["x-nullable"] == true {
			if typ, ok := out["type"].(string); ok {
				out["type"] = []any{typ, "null"}
			}
			delete(out, "x-nullable")
		}
		if ex, ok := out["example"]; ok {
			if _, isSchema := out["type"]; isSchema {
				out["examples"] = []any{ex}
				delete(out, "example")
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, child := range t {
			out[i] = schema31(child)
		}
		return out
	}
	return v
}

func asMap(v any) map[string]any { m, _ := v.(map[string]any); return m }

func strList(v any, fallback ...string) []string {
	var out []string
	for _, s := range asSlice(v) {
		if str, ok := s.(string); ok {
			out = append(out, str)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
