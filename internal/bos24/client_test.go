package bos24

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maidulcu/masaar-crm/internal/safehttp"
)

type seen struct {
	method, path, rawQuery, auth string
	body                         map[string]any
}

// fakeAPI answers like the data API: fixed bodies per path, recording each request.
func fakeAPI(t *testing.T, routes map[string]any) (*Client, *[]seen) {
	t.Helper()
	var log []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := seen{method: r.Method, path: r.URL.EscapedPath(), rawQuery: r.URL.RawQuery, auth: r.Header.Get("Authorization")}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			_ = json.Unmarshal(b, &s.body)
		}
		log = append(log, s)
		v, ok := routes[r.URL.EscapedPath()]
		switch {
		case !ok:
			w.WriteHeader(http.StatusNotFound)
		case v == 401 || v == 429 || v == 502:
			w.WriteHeader(v.(int))
		default:
			_ = json.NewEncoder(w).Encode(v)
		}
	}))
	t.Cleanup(srv.Close)
	c := NewClient("bos24_test", srv.URL+"/", nil)
	c.http.Transport = http.DefaultTransport
	return c, &log
}

func TestNewClientDisabledWithoutToken(t *testing.T) {
	if NewClient("  ", "https://example.test", nil) != nil {
		t.Fatal("client without token should be nil")
	}
}

func TestTransactionsTranslateFiltersAndUnwrap(t *testing.T) {
	c, log := fakeAPI(t, map[string]any{
		"/api/v1/transactions": map[string]any{"data": []any{map[string]any{"id": 1}, map[string]any{"id": 2}}, "total": 2},
	})
	got, err := c.GetTransactions(context.Background(), map[string]interface{}{
		"area": "Dubai Marina", "trans_type": "Sales", "limit": 5, "min_price": 1e6,
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("got %v, %v", got, err)
	}
	r := (*log)[0]
	if r.auth != "Bearer bos24_test" {
		t.Errorf("auth = %q", r.auth)
	}
	if r.rawQuery != "area=Dubai+Marina&min_price=1e%2B06&per_page=5&trans_group=Sales" {
		t.Errorf("query = %q", r.rawQuery)
	}
}

func TestShapes(t *testing.T) {
	c, log := fakeAPI(t, map[string]any{
		"/api/v1/insights/top-areas":                map[string]any{"ranking_metric": "volume", "top_areas": []any{map[string]any{"area": "A"}}},
		"/api/v1/rent-contracts/areas":              []any{map[string]any{"area": "B"}},
		"/api/v1/maps/areas":                        []any{map[string]any{"name": "C"}, "junk"},
		"/api/v1/areas/summary/Dubai%20Marina":      map[string]any{"area_name": "Dubai Marina"},
		"/api/v1/transactions/by-project/A%2FB%20C": map[string]any{"data": []any{map[string]any{"id": 9}}},
	})
	ctx := context.Background()
	if top, _ := c.GetTopAreas(ctx, "volume", "", 3); len(top) != 1 || top[0]["area"] != "A" {
		t.Errorf("top areas = %v", top)
	}
	if (*log)[0].rawQuery != "limit=3&sort_by=volume" {
		t.Errorf("top areas query = %q", (*log)[0].rawQuery)
	}
	if ra, _ := c.GetRentalAreas(ctx); len(ra["data"].([]any)) != 1 {
		t.Errorf("rental areas = %v", ra)
	}
	if areas, _ := c.GetAreasWithLocations(ctx); len(areas) != 1 {
		t.Errorf("map areas = %v", areas)
	}
	if sum, err := c.GetAreaSummary(ctx, "Dubai Marina"); err != nil || sum["area_name"] != "Dubai Marina" {
		t.Errorf("area summary = %v, %v", sum, err)
	}
	// Path segments are escaped: a "/" in a name cannot reach another endpoint.
	if tx, err := c.GetTransactionsByProject(ctx, "A/B C", nil); err != nil || len(tx) != 1 {
		t.Errorf("by project = %v, %v", tx, err)
	}
}

func TestErrors(t *testing.T) {
	c, _ := fakeAPI(t, map[string]any{
		"/api/v1/buildings/1": 401,
		"/api/v1/buildings/2": 429,
		"/api/v1/buildings/3": 502,
	})
	ctx := context.Background()
	cases := map[int]error{1: ErrUnauthorized, 2: ErrRateLimited, 4: ErrNotFound}
	for id, want := range cases {
		if _, err := c.GetBuilding(ctx, id); !errors.Is(err, want) {
			t.Errorf("building %d: %v, want %v", id, err, want)
		}
	}
	var apiErr *APIError
	if _, err := c.GetBuilding(ctx, 3); !errors.As(err, &apiErr) || apiErr.Status != 502 {
		t.Errorf("building 3: %v", err)
	}
}

func TestDescribeSendsOnlyKnownFields(t *testing.T) {
	c, log := fakeAPI(t, map[string]any{
		"/api/v1/search/describe": map[string]any{"description": "Nice flat", "property": map[string]any{}},
	})
	desc, err := c.DescribeProperty(context.Background(), map[string]interface{}{
		"tone": "professional", "area": "Marina", "rooms": 2, "callback_url": "https://evil.test", "admin": true,
	})
	if err != nil || desc != "Nice flat" {
		t.Fatalf("desc = %q, %v", desc, err)
	}
	b := (*log)[0].body
	if (*log)[0].method != "POST" || b["tone"] != "professional" || b["callback_url"] != nil || b["admin"] != nil {
		t.Errorf("body = %v", b)
	}
	if pd, _ := b["property_data"].(map[string]any); pd["area"] != "Marina" || pd["rooms"] != float64(2) {
		t.Errorf("property_data = %v", b["property_data"])
	}
}

func TestClient_SSRFBlocked(t *testing.T) {
	// A loopback address (e.g., 127.0.0.1) or private IP must be blocked by safehttp dialer.
	c := NewClient("bos24_test", "http://127.0.0.1:8080", nil)
	if c == nil {
		t.Fatal("expected non-nil client")
	}

	_, err := c.GetBuilding(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected SSRF error when sending request to internal IP, got nil")
	}
	if !errors.Is(err, safehttp.ErrBlocked) && !testing.Verbose() {
		t.Logf("got expected error for SSRF target: %v", err)
	}
}
