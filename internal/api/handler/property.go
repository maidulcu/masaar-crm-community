package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/bos24"
	"github.com/maidulcu/masaar-crm/internal/pdf"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

const bos24Timeout = 15 * time.Second

type PropertyHandler struct {
	bos24Client         *bos24.Client
	companySettingsRepo *repo.CompanySettingsRepo
}

func NewPropertyHandler(bos24Client *bos24.Client, companySettingsRepo *repo.CompanySettingsRepo) *PropertyHandler {
	return &PropertyHandler{
		bos24Client:         bos24Client,
		companySettingsRepo: companySettingsRepo,
	}
}

func (h *PropertyHandler) notEnabled(c *fiber.Ctx) error {
	return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
		"error": "real estate service not enabled — configure BOS24_API_TOKEN",
	})
}

// SearchProperties godoc
// @Summary      AI natural-language property search
// @Description  Search for properties using natural language (e.g., "2BR apartments in Marina").
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        body  body  object{query=string,limit=int}  true  "Search query"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/search [post]
func (h *PropertyHandler) SearchProperties(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	var req SearchPropertiesRequest
	if err := c.BodyParser(&req); err != nil || req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "query is required"})
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.SearchProperties(ctx, req.Query, limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// GetTransactions godoc
// @Summary      Property transaction history
// @Description  Get DLD transaction records with optional filters.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  false  "Area name"
// @Param        property_type  query  string  false  "Unit, Villa, etc."
// @Param        trans_type     query  string  false  "Sell or Rent"
// @Param        min_price      query  number  false  "Min price AED"
// @Param        max_price      query  number  false  "Max price AED"
// @Param        limit          query  int     false  "Result limit (default 20)"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/transactions [get]
func (h *PropertyHandler) GetTransactions(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if v := c.Query("trans_type"); v != "" {
		filters["trans_type"] = v
	}
	if v := c.Query("min_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filters["min_price"] = f
		}
	}
	if v := c.Query("max_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filters["max_price"] = f
		}
	}
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			filters["limit"] = l
		}
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetTransactions(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"transactions": result})
}

// GetBuildings godoc
// @Summary      Building autocomplete search
// @Description  Search buildings by name for autocomplete dropdowns.
// @Tags         Properties
// @Produce      json
// @Param        q      query  string  true   "Building name"
// @Param        limit  query  int     false  "Result limit (default 10)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/buildings [get]
func (h *PropertyHandler) GetBuildings(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "q is required"})
	}

	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.SearchBuildings(ctx, query, limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"buildings": result})
}

// GetBuildingByID godoc
// @Summary      Building details
// @Description  Get full details for a building by its numeric ID.
// @Tags         Properties
// @Produce      json
// @Param        id  path  int  true  "Building ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/buildings/{id} [get]
func (h *PropertyHandler) GetBuildingByID(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	buildingID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid building ID"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetBuilding(ctx, buildingID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// GetNearbyPOIs godoc
// @Summary      Points of interest near a location
// @Description  Get metro stations, schools, malls, hospitals within radius of coordinates.
// @Tags         Properties
// @Produce      json
// @Param        lat        query  number  true   "Latitude"
// @Param        lng        query  number  true   "Longitude"
// @Param        radius_km  query  number  false  "Radius in km (default 2)"
// @Param        category   query  string  false  "metro, school, mall, hospital (all if omitted)"
// @Param        limit      query  int     false  "Result limit (default 20)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/pois [get]
func (h *PropertyHandler) GetNearbyPOIs(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lat and lng are required"})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}

	radius := 2.0
	if r, err := strconv.ParseFloat(c.Query("radius_km"), 64); err == nil && r > 0 {
		radius = r
	}

	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetPOIs(ctx, lat, lng, radius, c.Query("category"), limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"pois": result})
}

// GetNearbySchools godoc
// @Summary      Schools near a location
// @Description  Convenience endpoint for schools specifically — subset of /pois.
// @Tags         Properties
// @Produce      json
// @Param        lat        query  number  true   "Latitude"
// @Param        lng        query  number  true   "Longitude"
// @Param        radius_km  query  number  false  "Radius in km (default 2)"
// @Param        limit      query  int     false  "Result limit (default 10)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/schools/nearby [get]
func (h *PropertyHandler) GetNearbySchools(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lat and lng are required"})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}

	radius := 2.0
	if r, err := strconv.ParseFloat(c.Query("radius_km"), 64); err == nil && r > 0 {
		radius = r
	}

	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetNearbySchools(ctx, lat, lng, radius, limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"schools": result})
}

// GetAreas godoc
// @Summary      List UAE real estate areas
// @Description  Returns all areas/districts for use in filter dropdowns.
// @Tags         Properties
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/areas [get]
func (h *PropertyHandler) GetAreas(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetAreas(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"areas": result})
}

// GetAreaSummary godoc
// @Summary      Area market summary
// @Description  Returns avg prices, transaction volume, and trends for a specific area.
// @Tags         Properties
// @Produce      json
// @Param        slug  path  string  true  "Area slug (e.g. dubai-marina)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/areas/{slug}/summary [get]
func (h *PropertyHandler) GetAreaSummary(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area slug is required"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetAreaSummary(ctx, slug)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// GetAreaBuildings godoc
// @Summary      Buildings in an area
// @Description  Returns buildings within a named area for project/building drilldown.
// @Tags         Properties
// @Produce      json
// @Param        slug   path   string  true   "Area slug"
// @Param        limit  query  int     false  "Result limit (default 20)"
// @Success      200  {array}   map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/areas/{slug}/buildings [get]
func (h *PropertyHandler) GetAreaBuildings(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area slug is required"})
	}

	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetAreaBuildings(ctx, slug, limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"buildings": result})
}

// GetMapAreas godoc
// @Summary      Areas with geo coordinates for map rendering
// @Description  Returns area centroids and boundaries for rendering an interactive map.
// @Tags         Properties
// @Produce      json
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/map/areas [get]
func (h *PropertyHandler) GetMapAreas(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetAreasWithLocations(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"areas": result})
}

// GetTransactionAreas godoc
// @Summary      Transaction heatmap data by area
// @Description  Returns transaction counts and avg prices grouped by area — used to drive heatmaps.
// @Tags         Properties
// @Produce      json
// @Param        property_type  query  string  false  "Unit, Villa, etc."
// @Param        trans_type     query  string  false  "Sell or Rent"
// @Param        limit          query  int     false  "Result limit (default 30)"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/transactions/areas [get]
func (h *PropertyHandler) GetTransactionAreas(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if v := c.Query("trans_type"); v != "" {
		filters["trans_type"] = v
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetTransactionAreas(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"areas": result})
}

// GetEjariRentals godoc
// @Summary      Ejari rental contracts
// @Description  Returns rental contract data from Ejari registry.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  false  "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Param        rooms          query  string  false  "1BR, 2BR, Studio, etc."
// @Param        limit          query  int     false  "Result limit (default 20)"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals/ejari [get]
func (h *PropertyHandler) GetEjariRentals(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if v := c.Query("rooms"); v != "" {
		filters["rooms"] = v
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetEjariRentals(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"rentals": result})
}

// GetEjariYield godoc
// @Summary      Rental yield analysis from Ejari
// @Description  Returns calculated rental yields (annual rent / sale price) by area and type.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  false  "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals/ejari/yield [get]
func (h *PropertyHandler) GetEjariYield(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetEjariYield(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"yield": result})
}

// GetRentals godoc
// @Summary      General rental listings
// @Description  Returns current rental listings.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  false  "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Param        limit          query  int     false  "Result limit (default 20)"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals [get]
func (h *PropertyHandler) GetRentals(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetRentals(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"rentals": result})
}

// GetDevelopers godoc
// @Summary      Real estate developer listing
// @Description  Returns list of developers for autocomplete and filtering.
// @Tags         Properties
// @Produce      json
// @Param        q      query  string  false  "Developer name search"
// @Param        limit  query  int     false  "Result limit (default 20)"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/developers [get]
func (h *PropertyHandler) GetDevelopers(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetDevelopers(ctx, c.Query("q"), limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"developers": result})
}

// GetProjects godoc
// @Summary      Real estate project listing
// @Description  Returns off-plan and completed project listings.
// @Tags         Properties
// @Produce      json
// @Param        area       query  string  false  "Area name"
// @Param        developer  query  string  false  "Developer name"
// @Param        status     query  string  false  "off-plan, completed, under-construction"
// @Param        limit      query  int     false  "Result limit (default 20)"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/projects [get]
func (h *PropertyHandler) GetProjects(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("developer"); v != "" {
		filters["developer"] = v
	}
	if v := c.Query("status"); v != "" {
		filters["status"] = v
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetProjects(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"projects": result})
}

// SearchProjects godoc
// @Summary      AI natural-language project search
// @Description  Search for off-plan projects using natural language.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        body  body  object{query=string,limit=int}  true  "Search query"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/projects/search [post]
func (h *PropertyHandler) SearchProjects(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	var req SearchPropertiesRequest
	if err := c.BodyParser(&req); err != nil || req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "query is required"})
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.SearchProjects(ctx, req.Query, limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// GetValuations godoc
// @Summary      Automated property valuations
// @Description  Returns AVM (automated valuation model) estimates by area and type.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  false  "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Param        rooms          query  string  false  "1BR, 2BR, Studio, etc."
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/valuations [get]
func (h *PropertyHandler) GetValuations(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if v := c.Query("rooms"); v != "" {
		filters["rooms"] = v
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetValuations(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"valuations": result})
}

// DescribeProperty godoc
// @Summary      AI property description generator
// @Description  Generates a professional marketing description for a property listing.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        body  body  object{}  true  "Property details (area, type, rooms, size, amenities, etc.)"
// @Success      200   {object}  object{description=string}
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/ai/describe [post]
func (h *PropertyHandler) DescribeProperty(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	var details map[string]interface{}
	if err := c.BodyParser(&details); err != nil || len(details) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "property details are required"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.DescribeProperty(ctx, details)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// GetUnits godoc
// @Summary      Property unit listings
// @Description  Returns individual unit listings within a building or project.
// @Tags         Properties
// @Produce      json
// @Param        building_id    query  int     false  "Building ID"
// @Param        project_id     query  string  false  "Project ID"
// @Param        property_type  query  string  false  "Property type"
// @Param        limit          query  int     false  "Result limit (default 20)"
// @Success      200  {array}   map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/units [get]
func (h *PropertyHandler) GetUnits(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	filters := make(map[string]interface{})
	if v := c.Query("building_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			filters["building_id"] = id
		}
	}
	if v := c.Query("project_id"); v != "" {
		filters["project_id"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	result, err := h.bos24Client.GetUnits(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"units": result})
}

// GetYieldAnalysis godoc
// @Summary      Rental yield analysis (combined)
// @Description  Combines rental and sales stats to calculate indicative yield for an area.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  true   "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Success      200   {object}  YieldAnalysisResponse
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/yield-analysis [get]
func (h *PropertyHandler) GetYieldAnalysis(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	area := c.Query("area")
	if area == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area is required"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	filters := map[string]interface{}{"area": area}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}

	rentalStats, err := h.bos24Client.GetEjariStats(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	salesStats, err := h.bos24Client.GetTransactionStats(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(YieldAnalysisResponse{
		Area:        area,
		RentalStats: rentalStats,
		SalesStats:  salesStats,
	})
}

// GetComparables godoc
// @Summary      Comparable property transactions
// @Description  Returns recent transactions in an area for CMA (comparative market analysis).
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  true   "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Param        limit          query  int     false  "Result limit (default 10)"
// @Success      200   {object}  ComparablesResponse
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/comparables [get]
func (h *PropertyHandler) GetComparables(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	area := c.Query("area")
	if area == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area is required"})
	}

	filters := map[string]interface{}{"area": area}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	transactions, err := h.bos24Client.GetTransactions(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	comparables := make([]interface{}, len(transactions))
	for i, t := range transactions {
		comparables[i] = t
	}

	return c.JSON(ComparablesResponse{
		Area:        area,
		Comparables: comparables,
		ResultCount: len(transactions),
	})
}

// GetMarketTrends godoc
// @Summary      Market trends by area
// @Description  Returns combined sales and rental trends for a given area.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  true   "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Param        period         query  string  false  "month, quarter, year"
// @Success      200   {object}  MarketTrendsResponse
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/market-trends [get]
func (h *PropertyHandler) GetMarketTrends(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	area := c.Query("area")
	if area == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area is required"})
	}

	filters := map[string]interface{}{"area": area}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}

	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()

	stats, err := h.bos24Client.GetTransactionStats(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}

	rentalStats, err := h.bos24Client.GetEjariStats(ctx, filters)
	if err != nil {
		rentalStats = map[string]interface{}{}
	}

	return c.JSON(MarketTrendsResponse{
		Area:        area,
		SalesStats:  stats,
		RentalStats: rentalStats,
		Period:      c.Query("period", "month"),
	})
}

// GetMarketOverview godoc
// @Summary      Market overview insights
// @Description  Returns high-level market overview for a period and property type.
// @Tags         Properties
// @Produce      json
// @Param        period         query  string  false  "month, quarter, year"
// @Param        property_type  query  string  false  "Property type"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/insights/market-overview [get]
func (h *PropertyHandler) GetMarketOverview(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetMarketOverview(ctx, c.Query("period"), c.Query("property_type"))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetAreaComparison godoc
// @Summary      Compare multiple areas
// @Description  Returns side-by-side stats for multiple areas.
// @Tags         Properties
// @Produce      json
// @Param        areas          query  string  true   "Comma-separated area names"
// @Param        property_type  query  string  false  "Property type"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/insights/area-comparison [get]
func (h *PropertyHandler) GetAreaComparison(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	areas := c.Query("areas")
	if areas == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "areas is required"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetAreaComparison(ctx, areas, c.Query("property_type"))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetPriceTrends godoc
// @Summary      Price trends for an area
// @Description  Returns historical price trends for a specific area.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  true   "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Param        granularity    query  string  false  "monthly, quarterly, yearly"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/insights/price-trends [get]
func (h *PropertyHandler) GetPriceTrends(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	area := c.Query("area")
	if area == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area is required"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetPriceTrends(ctx, area, c.Query("property_type"), c.Query("granularity"))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetTopAreas godoc
// @Summary      Top performing areas
// @Description  Returns top areas ranked by a specific metric.
// @Tags         Properties
// @Produce      json
// @Param        metric         query  string  false  "volume, price, yield"
// @Param        property_type  query  string  false  "Property type"
// @Param        limit          query  int     false  "Result limit (default 10)"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/insights/top-areas [get]
func (h *PropertyHandler) GetTopAreas(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetTopAreas(ctx, c.Query("metric"), c.Query("property_type"), limit)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetBrokers godoc
// @Summary      List registered brokers
// @Description  Returns paginated list of real estate brokers.
// @Tags         Properties
// @Produce      json
// @Param        limit   query  int  false  "Result limit (default 20)"
// @Param        offset  query  int  false  "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/brokers [get]
func (h *PropertyHandler) GetBrokers(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetBrokers(ctx, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetTransaction godoc
// @Summary      Single transaction details
// @Description  Returns details for a specific DLD transaction by ID.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Transaction ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/transactions/{id} [get]
func (h *PropertyHandler) GetTransaction(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetTransaction(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetEnrichedTransaction godoc
// @Summary      Enriched transaction details
// @Description  Returns a transaction with additional market context.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Transaction ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/transactions/{id}/enriched [get]
func (h *PropertyHandler) GetEnrichedTransaction(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetEnrichedTransaction(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetTransactionsByProject godoc
// @Summary      Transactions for a project
// @Description  Returns all DLD transactions for a specific project name.
// @Tags         Properties
// @Produce      json
// @Param        name   path   string  true   "Project name"
// @Param        limit  query  int     false  "Result limit"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/transactions/by-project/{name} [get]
func (h *PropertyHandler) GetTransactionsByProject(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	filters := make(map[string]interface{})
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		filters["limit"] = l
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetTransactionsByProject(ctx, name, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetAreaTransactionSummary godoc
// @Summary      Transaction summary for an area
// @Description  Returns aggregated transaction summary for a named area.
// @Tags         Properties
// @Produce      json
// @Param        name  path  string  true  "Area name"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/transactions/area/{name}/summary [get]
func (h *PropertyHandler) GetAreaTransactionSummary(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetAreaTransactionSummary(ctx, name)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetRentalStats godoc
// @Summary      Rental market statistics
// @Description  Returns aggregated rental statistics with optional filters.
// @Tags         Properties
// @Produce      json
// @Param        area           query  string  false  "Area name"
// @Param        property_type  query  string  false  "Property type"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals/stats [get]
func (h *PropertyHandler) GetRentalStats(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	filters := make(map[string]interface{})
	if v := c.Query("area"); v != "" {
		filters["area"] = v
	}
	if v := c.Query("property_type"); v != "" {
		filters["property_type"] = v
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetRentalStats(ctx, filters)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetRentalAreas godoc
// @Summary      Areas with rental data
// @Description  Returns list of areas that have rental contract data.
// @Tags         Properties
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals/areas [get]
func (h *PropertyHandler) GetRentalAreas(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetRentalAreas(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetRentalsByProject godoc
// @Summary      Rentals for a project
// @Description  Returns rental contracts for a specific project.
// @Tags         Properties
// @Produce      json
// @Param        name    path   string  true   "Project name"
// @Param        limit   query  int     false  "Result limit (default 20)"
// @Param        offset  query  int     false  "Offset"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals/project/{name} [get]
func (h *PropertyHandler) GetRentalsByProject(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetRentalsByProject(ctx, name, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetRentalsByBuilding godoc
// @Summary      Rentals for a building
// @Description  Returns rental contracts for a specific building.
// @Tags         Properties
// @Produce      json
// @Param        name    path   string  true   "Building name"
// @Param        limit   query  int     false  "Result limit (default 20)"
// @Param        offset  query  int     false  "Offset"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/rentals/building/{name} [get]
func (h *PropertyHandler) GetRentalsByBuilding(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetRentalsByBuilding(ctx, name, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetLands godoc
// @Summary      Land parcel listings
// @Description  Returns paginated land parcels from DLD records.
// @Tags         Properties
// @Produce      json
// @Param        limit   query  int  false  "Result limit (default 20)"
// @Param        offset  query  int  false  "Offset"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/lands [get]
func (h *PropertyHandler) GetLands(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetLands(ctx, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetLand godoc
// @Summary      Single land parcel
// @Description  Returns details for a specific land parcel.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Land ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/lands/{id} [get]
func (h *PropertyHandler) GetLand(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	idInt, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetLand(ctx, idInt)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetMapConfig godoc
// @Summary      Map configuration
// @Description  Returns map rendering config (tile server, default center, zoom).
// @Tags         Properties
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/map/config [get]
func (h *PropertyHandler) GetMapConfig(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetMapConfig(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetMapBounds godoc
// @Summary      Map geographic bounds
// @Description  Returns the bounding box for the map view.
// @Tags         Properties
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/map/bounds [get]
func (h *PropertyHandler) GetMapBounds(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetMapBounds(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetPOICategories godoc
// @Summary      POI category list
// @Description  Returns all available POI categories for filtering.
// @Tags         Properties
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/map/poi-categories [get]
func (h *PropertyHandler) GetPOICategories(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetPOICategories(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetPropertyHeatmap godoc
// @Summary      Property price heatmap data
// @Description  Returns heatmap data points for rendering price density on a map.
// @Tags         Properties
// @Produce      json
// @Param        property_type  query  string  false  "Property type"
// @Param        trans_type     query  string  false  "Sell or Rent"
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/map/heatmap [get]
func (h *PropertyHandler) GetPropertyHeatmap(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetPropertyHeatmap(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetAreaLocation godoc
// @Summary      Area geocoordinates
// @Description  Returns the geographic center and boundary for a named area.
// @Tags         Properties
// @Produce      json
// @Param        name  path  string  true  "Area name"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/map/area/{name} [get]
func (h *PropertyHandler) GetAreaLocation(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetAreaLocation(ctx, name)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetAreaByID godoc
// @Summary      Area details by ID
// @Description  Returns area details for a given area ID.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Area ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/areas/{id} [get]
func (h *PropertyHandler) GetAreaByID(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	idInt, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetAreaByID(ctx, idInt)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetDeveloper godoc
// @Summary      Developer details by ID
// @Description  Returns details for a specific developer.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Developer ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/developers/{id} [get]
func (h *PropertyHandler) GetDeveloper(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	idInt, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetDeveloper(ctx, idInt)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetProject godoc
// @Summary      Project details by ID
// @Description  Returns details for a specific real estate project.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Project ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/projects/{id} [get]
func (h *PropertyHandler) GetProject(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	idInt, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetProject(ctx, idInt)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetUnit godoc
// @Summary      Unit details by ID
// @Description  Returns details for a specific property unit.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Unit ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/units/{id} [get]
func (h *PropertyHandler) GetUnit(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	idInt, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetUnit(ctx, idInt)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetValuationByID godoc
// @Summary      Valuation details by ID
// @Description  Returns AVM valuation details for a specific record.
// @Tags         Properties
// @Produce      json
// @Param        id  path  string  true  "Valuation ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  object{error=string}
// @Failure      503  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/valuations/{id} [get]
func (h *PropertyHandler) GetValuationByID(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}
	idInt, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	ctx, cancel := context.WithTimeout(c.Context(), bos24Timeout)
	defer cancel()
	result, err := h.bos24Client.GetValuation(ctx, idInt)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GeneratePropertyReport godoc
// @Summary      Generate a branded property market report PDF
// @Description  Generates a client-facing PDF report with DLD data, yield analysis, and nearby amenities.
// @Tags         Properties
// @Accept       json
// @Produce      application/pdf
// @Param        body  body  GenerateReportRequest  true  "Report parameters"
// @Success      200   {file}  binary  "PDF file"
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /properties/report/pdf [post]
func (h *PropertyHandler) GeneratePropertyReport(c *fiber.Ctx) error {
	if h.bos24Client == nil {
		return h.notEnabled(c)
	}

	var req GenerateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Area == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "area is required"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	// Company branding
	company, _ := h.companySettingsRepo.Get(ctx)
	companyName := "Masaar CRM"
	companyAddress := "Dubai, United Arab Emirates"
	companyPhone := ""
	companyEmail := ""
	if company != nil {
		companyName = company.Name
		companyAddress = company.BusinessAddress
		companyPhone = company.BusinessPhone
		companyEmail = company.BusinessEmail
	}

	// Area slug for data lookups
	areaSlug := req.AreaSlug
	if areaSlug == "" {
		areaSlug = req.Area
	}

	// Fetch area summary
	areaSummary, _ := h.bos24Client.GetAreaSummary(ctx, areaSlug)

	// Fetch comparable transactions
	transFilters := map[string]interface{}{}
	if req.PropertyType != "" {
		transFilters["property_type"] = req.PropertyType
	}
	if req.BudgetMax > 0 {
		transFilters["max_price"] = req.BudgetMax
	}
	if req.BudgetMin > 0 {
		transFilters["min_price"] = req.BudgetMin
	}
	transFilters["limit"] = 10
	comparables, _ := h.bos24Client.GetTransactions(ctx, transFilters)

	// Fetch yield data if requested
	var yieldData []map[string]interface{}
	if req.IncludeYield {
		yieldFilters := map[string]interface{}{}
		if req.PropertyType != "" {
			yieldFilters["property_type"] = req.PropertyType
		}
		yieldData, _ = h.bos24Client.GetEjariYield(ctx, yieldFilters)
	}

	// Fetch POIs if coordinates provided
	var pois []map[string]interface{}
	if req.IncludePOIs && req.Lat != 0 && req.Lng != 0 {
		pois, _ = h.bos24Client.GetPOIs(ctx, req.Lat, req.Lng, 2.0, "", 10)
	}

	// AI description
	aiDescription := ""
	if req.AIDescription != "" {
		aiDescription = req.AIDescription
	}

	pdfData := pdf.PropertyReportData{
		CompanyName:    companyName,
		CompanyAddress: companyAddress,
		CompanyPhone:   companyPhone,
		CompanyEmail:   companyEmail,
		AgentName:      req.AgentName,
		ClientName:     req.ClientName,
		Area:           req.Area,
		PropertyType:   req.PropertyType,
		Bedrooms:       req.Bedrooms,
		BudgetMin:      req.BudgetMin,
		BudgetMax:      req.BudgetMax,
		AreaSummary:    areaSummary,
		Comparables:    comparables,
		YieldData:      yieldData,
		POIs:           pois,
		AIDescription:  aiDescription,
		GeneratedAt:    time.Now(),
	}

	pdfBytes, err := pdf.GeneratePropertyReport(pdfData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate PDF"})
	}

	filename := fmt.Sprintf("property-report-%s.pdf", areaSlug)
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.Send(pdfBytes)
}

// Request/Response types

type SearchPropertiesRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type GenerateReportRequest struct {
	ClientName    string  `json:"client_name"`
	AgentName     string  `json:"agent_name"`
	Area          string  `json:"area"`
	AreaSlug      string  `json:"area_slug"`
	PropertyType  string  `json:"property_type"`
	Bedrooms      string  `json:"bedrooms"`
	BudgetMin     float64 `json:"budget_min"`
	BudgetMax     float64 `json:"budget_max"`
	IncludeYield  bool    `json:"include_yield"`
	IncludePOIs   bool    `json:"include_pois"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	LeadID        string  `json:"lead_id"`
	AIDescription string  `json:"ai_description"`
}

type YieldAnalysisResponse struct {
	Area        string                 `json:"area"`
	RentalStats map[string]interface{} `json:"rental_stats"`
	SalesStats  map[string]interface{} `json:"sales_stats"`
}

type ComparablesResponse struct {
	Area        string        `json:"area"`
	Comparables []interface{} `json:"comparables"`
	ResultCount int           `json:"result_count"`
}

type MarketTrendsResponse struct {
	Area        string                 `json:"area"`
	SalesStats  map[string]interface{} `json:"sales_stats"`
	RentalStats map[string]interface{} `json:"rental_stats"`
	Period      string                 `json:"period"`
}
