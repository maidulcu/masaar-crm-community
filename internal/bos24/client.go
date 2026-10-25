// Package bos24 is the client for the WAQOV real-estate data API (DLD transactions, Ejari rents,
// buildings, projects, valuations). The API describes itself at <base URL>/openapi.json and is
// called with a bearer API key. The package keeps its historical name; config uses BOS24_*.
package bos24

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/safehttp"
	"github.com/redis/go-redis/v9"
)

var (
	ErrNotFound     = errors.New("market data: not found")
	ErrUnauthorized = errors.New("market data: API key rejected")
	ErrRateLimited  = errors.New("market data: rate limited")
)

// APIError is any other non-2xx answer. The body is not kept: it may echo the request.
type APIError struct {
	Status int
	Path   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("market data: %s answered %d", e.Path, e.Status)
}

const (
	maxResponseBytes = 10 << 20
	cacheShort       = 10 * time.Minute // listings, transactions, search
	cacheLong        = 6 * time.Hour    // stats, areas, map layers
	cacheStatic      = 24 * time.Hour   // map config, POI categories
)

// Client calls the data API. A nil *Client means the integration is off (no token).
type Client struct {
	base  string
	token string
	http  *http.Client
	rdb   *redis.Client // optional response cache for GETs
}

// NewClient returns nil when token is empty, so handlers answer 503 "not enabled".
// It uses safehttp.NewClient to enforce dial-time SSRF protection against private or internal addresses.
func NewClient(token, baseURL string, rdb *redis.Client) *Client {
	if !IsEnabled(token) {
		return nil
	}
	return &Client{
		base:  strings.TrimRight(baseURL, "/"),
		token: strings.TrimSpace(token),
		http:  safehttp.NewClient(20 * time.Second),
		rdb:   rdb,
	}
}

func IsEnabled(token string) bool { return strings.TrimSpace(token) != "" }

// do calls method base+path. GET responses are cached in Redis for ttl when a cache is set.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, body any, ttl time.Duration) (any, error) {
	target := c.base + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}

	cacheKey := ""
	if method == http.MethodGet && c.rdb != nil && ttl > 0 {
		sum := sha256.Sum256([]byte(target))
		cacheKey = "dldapi:" + hex.EncodeToString(sum[:])
		if raw, err := c.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
			var v any
			if json.Unmarshal(raw, &v) == nil {
				return v, nil
			}
		}
	}

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("market data: %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("market data: %s: %w", path, err)
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, ErrUnauthorized
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return nil, &APIError{Status: resp.StatusCode, Path: path}
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("market data: %s: invalid JSON: %w", path, err)
	}
	if cacheKey != "" {
		_ = c.rdb.Set(ctx, cacheKey, raw, ttl).Err()
	}
	return v, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, ttl time.Duration) (any, error) {
	return c.do(ctx, http.MethodGet, path, q, nil, ttl)
}

func (c *Client) getMap(ctx context.Context, path string, q url.Values, ttl time.Duration) (map[string]interface{}, error) {
	v, err := c.get(ctx, path, q, ttl)
	if err != nil {
		return nil, err
	}
	return asMap(v), nil
}

// getList returns a list response, unwrapping {"data": [...]} or {key: [...]}.
func (c *Client) getList(ctx context.Context, path string, q url.Values, ttl time.Duration, keys ...string) ([]map[string]interface{}, error) {
	v, err := c.get(ctx, path, q, ttl)
	if err != nil {
		return nil, err
	}
	return asList(v, keys...), nil
}

// asMap returns an object response as is and wraps a list as {"data": list}.
func asMap(v any) map[string]interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return t
	case []interface{}:
		return map[string]interface{}{"data": t}
	}
	return map[string]interface{}{}
}

// asList returns a list response, or the list under the first of keys (default "data") in an
// object response. Non-object items are dropped.
func asList(v any, keys ...string) []map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		if len(keys) == 0 {
			keys = []string{"data"}
		}
		v = nil
		for _, k := range keys {
			if inner, ok := m[k]; ok {
				v = inner
				break
			}
		}
	}
	items, _ := v.([]interface{})
	out := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// query turns handler filters into query parameters, renaming keys per rename ("" drops one).
func query(filters map[string]interface{}, rename map[string]string) url.Values {
	q := url.Values{}
	for k, v := range filters {
		if to, ok := rename[k]; ok {
			k = to
		}
		if k == "" || v == nil {
			continue
		}
		s := fmt.Sprint(v)
		if s == "" {
			continue
		}
		q.Set(k, s)
	}
	return q
}

// pageQuery is query for endpoints paginated with page/per_page: "limit" becomes per_page.
func pageQuery(filters map[string]interface{}, rename map[string]string) url.Values {
	r := map[string]string{"limit": "per_page", "trans_type": "trans_group"}
	for k, v := range rename {
		r[k] = v
	}
	return query(filters, r)
}

func seg(s string) string { return url.PathEscape(s) }

func itoa(i int) string { return strconv.Itoa(i) }

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// page converts limit/offset to the API's 1-based page number.
func page(limit, offset int) string {
	if limit <= 0 {
		return "1"
	}
	return itoa(offset/limit + 1)
}

// ── Search ──────────────────────────────────────────────────────────────────

func (c *Client) SearchProperties(ctx context.Context, q string, limit int) (map[string]interface{}, error) {
	v, err := c.do(ctx, http.MethodPost, "/api/v1/search/properties", nil, map[string]any{"q": q, "limit": limit}, 0)
	if err != nil {
		return nil, err
	}
	return asMap(v), nil
}

func (c *Client) SearchBuildings(ctx context.Context, q string, limit int) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/search/buildings", url.Values{"q": {q}, "limit": {itoa(limit)}}, cacheShort)
}

func (c *Client) SearchProjects(ctx context.Context, q string, limit int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/search/projects", url.Values{"q": {q}, "limit": {itoa(limit)}}, cacheShort)
}

// describeFields are the property_data fields the describe endpoint reads.
var describeFields = map[string]bool{
	"name": true, "area": true, "master_project": true, "property_type": true,
	"floors": true, "rooms": true, "actual_area": true,
}

// describeBody keeps only what the describe endpoint accepts: property_id, tone, language and
// property_data. Known property fields sent flat are moved into property_data.
func describeBody(params map[string]interface{}) map[string]interface{} {
	body := map[string]interface{}{}
	data := map[string]interface{}{}
	if pd, ok := params["property_data"].(map[string]interface{}); ok {
		for k, v := range pd {
			if describeFields[k] {
				data[k] = v
			}
		}
	}
	for k, v := range params {
		switch {
		case k == "property_id" || k == "tone" || k == "language":
			body[k] = v
		case describeFields[k]:
			data[k] = v
		}
	}
	if len(data) > 0 {
		body["property_data"] = data
	}
	return body
}

// DescribeProperty asks the API for a listing description (see describeBody for params).
func (c *Client) DescribeProperty(ctx context.Context, params map[string]interface{}) (string, error) {
	v, err := c.do(ctx, http.MethodPost, "/api/v1/search/describe", nil, describeBody(params), 0)
	if err != nil {
		return "", err
	}
	desc, _ := asMap(v)["description"].(string)
	return desc, nil
}

// ── Transactions ────────────────────────────────────────────────────────────

func (c *Client) GetTransactions(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/transactions", pageQuery(filters, nil), cacheShort)
}

func (c *Client) GetTransaction(ctx context.Context, id string) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/transactions/"+seg(id), nil, cacheLong)
}

func (c *Client) GetEnrichedTransaction(ctx context.Context, id string) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/transactions/"+seg(id)+"/enriched", nil, cacheLong)
}

func (c *Client) GetTransactionAreas(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/transactions/areas", query(filters, map[string]string{"trans_type": "trans_group"}), cacheLong)
}

func (c *Client) GetTransactionsByProject(ctx context.Context, project string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/transactions/by-project/"+seg(project), query(filters, nil), cacheShort)
}

func (c *Client) GetTransactionStats(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/transactions/stats", query(filters, map[string]string{"trans_type": "trans_group"}), cacheLong)
}

func (c *Client) GetAreaTransactionSummary(ctx context.Context, area string) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/transactions/area/"+seg(area)+"/summary", nil, cacheLong)
}

// ── Buildings, units, lands, valuations ────────────────────────────────────

func (c *Client) GetBuilding(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/buildings/"+itoa(id), nil, cacheLong)
}

func (c *Client) GetUnit(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/units/"+itoa(id), nil, cacheLong)
}

func (c *Client) GetUnits(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/units", query(filters, nil), cacheShort)
}

func (c *Client) GetLands(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/lands", url.Values{"limit": {itoa(limit)}, "offset": {itoa(offset)}}, cacheShort)
}

func (c *Client) GetLand(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/lands/"+itoa(id), nil, cacheLong)
}

func (c *Client) GetValuations(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/valuations", query(filters, nil), cacheShort)
}

func (c *Client) GetValuation(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/valuations/"+itoa(id), nil, cacheLong)
}

// ── Areas ───────────────────────────────────────────────────────────────────

// GetAreas returns every area (about 350), one page of up to 500.
func (c *Client) GetAreas(ctx context.Context) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/areas", url.Values{"per_page": {"500"}, "limit": {"500"}}, cacheLong)
}

func (c *Client) GetAreaByID(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/areas/"+itoa(id), nil, cacheLong)
}

func (c *Client) GetAreaSummary(ctx context.Context, area string) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/areas/summary/"+seg(area), nil, cacheLong)
}

// GetAreaBuildings lists an area's buildings, most traded first.
func (c *Client) GetAreaBuildings(ctx context.Context, area string, limit int) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/buildings", url.Values{"area": {area}, "per_page": {itoa(limit)}}, cacheLong)
}

func (c *Client) GetAreaComparison(ctx context.Context, areas, propertyType string) (map[string]interface{}, error) {
	q := url.Values{"areas": {areas}}
	if propertyType != "" {
		q.Set("property_type", propertyType)
	}
	return c.getMap(ctx, "/api/v1/insights/area-comparison", q, cacheLong)
}

// ── Maps and points of interest ─────────────────────────────────────────────

func (c *Client) GetAreasWithLocations(ctx context.Context) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/maps/areas", nil, cacheLong)
}

func (c *Client) GetAreaLocation(ctx context.Context, area string) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/maps/areas/"+seg(area), nil, cacheLong)
}

func (c *Client) GetMapConfig(ctx context.Context) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/maps/config", nil, cacheStatic)
}

func (c *Client) GetMapBounds(ctx context.Context) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/maps/bounds", nil, cacheStatic)
}

func (c *Client) GetPropertyHeatmap(ctx context.Context) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/maps/heatmap/properties", nil, cacheLong)
}

func (c *Client) GetPOIs(ctx context.Context, lat, lng, radiusKm float64, category string, limit int) ([]map[string]interface{}, error) {
	q := url.Values{"lat": {ftoa(lat)}, "lng": {ftoa(lng)}, "radius_km": {ftoa(radiusKm)}, "limit": {itoa(limit)}}
	if category != "" {
		q.Set("category", category)
	}
	return c.getList(ctx, "/api/v1/maps/pois", q, cacheLong)
}

func (c *Client) GetNearbySchools(ctx context.Context, lat, lng, radiusKm float64, limit int) ([]map[string]interface{}, error) {
	return c.GetPOIs(ctx, lat, lng, radiusKm, "schools", limit)
}

func (c *Client) GetPOICategories(ctx context.Context) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/maps/pois/categories", nil, cacheStatic)
}

// ── Rentals (Ejari) ─────────────────────────────────────────────────────────

func (c *Client) GetRentals(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/rentals", pageQuery(filters, nil), cacheShort)
}

func (c *Client) GetEjariRentals(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/rent-contracts/ejari", query(filters, nil), cacheShort)
}

func (c *Client) GetEjariStats(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/rent-contracts/ejari/stats", query(filters, map[string]string{"limit": ""}), cacheLong)
}

func (c *Client) GetEjariYield(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/rent-contracts/ejari/yield", query(filters, nil), cacheLong)
}

func (c *Client) GetRentalStats(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/rent-contracts/stats", query(filters, nil), cacheLong)
}

func (c *Client) GetRentalAreas(ctx context.Context) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/rent-contracts/areas", nil, cacheLong)
}

func (c *Client) GetRentalsByProject(ctx context.Context, project string, limit, offset int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/rent-contracts/project/"+seg(project),
		url.Values{"limit": {itoa(limit)}, "offset": {itoa(offset)}}, cacheShort)
}

func (c *Client) GetRentalsByBuilding(ctx context.Context, building string, limit, offset int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/rent-contracts/building/"+seg(building),
		url.Values{"limit": {itoa(limit)}, "offset": {itoa(offset)}}, cacheShort)
}

// ── Developers, projects, brokers ───────────────────────────────────────────

func (c *Client) GetDevelopers(ctx context.Context, search string, limit int) ([]map[string]interface{}, error) {
	q := url.Values{"per_page": {itoa(limit)}}
	if search != "" {
		q.Set("search", search)
	}
	return c.getList(ctx, "/api/v1/developers", q, cacheLong)
}

func (c *Client) GetDeveloper(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/developers/"+itoa(id), nil, cacheLong)
}

func (c *Client) GetProjects(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/projects", pageQuery(filters, nil), cacheLong)
}

func (c *Client) GetProject(ctx context.Context, id int) (map[string]interface{}, error) {
	return c.getMap(ctx, "/api/v1/projects/"+itoa(id), nil, cacheLong)
}

func (c *Client) GetBrokers(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	return c.getList(ctx, "/api/v1/brokers", url.Values{"per_page": {itoa(limit)}, "page": {page(limit, offset)}}, cacheLong)
}

// ── Insights ────────────────────────────────────────────────────────────────

func (c *Client) GetMarketOverview(ctx context.Context, period, propertyType string) (map[string]interface{}, error) {
	q := url.Values{}
	if period != "" {
		q.Set("period", period)
	}
	if propertyType != "" {
		q.Set("property_type", propertyType)
	}
	return c.getMap(ctx, "/api/v1/insights/market-overview", q, cacheLong)
}

func (c *Client) GetPriceTrends(ctx context.Context, area, propertyType, granularity string) (map[string]interface{}, error) {
	q := url.Values{"area": {area}}
	if propertyType != "" {
		q.Set("property_type", propertyType)
	}
	if granularity != "" {
		q.Set("granularity", granularity)
	}
	return c.getMap(ctx, "/api/v1/insights/price-trends", q, cacheLong)
}

// GetTopAreas ranks areas by metric (the API's sort_by: volume, count, avg_price…).
func (c *Client) GetTopAreas(ctx context.Context, metric, propertyType string, limit int) ([]map[string]interface{}, error) {
	q := url.Values{"limit": {itoa(limit)}}
	if metric != "" {
		q.Set("sort_by", metric)
	}
	if propertyType != "" {
		q.Set("property_type", propertyType)
	}
	return c.getList(ctx, "/api/v1/insights/top-areas", q, cacheLong, "top_areas", "data")
}
