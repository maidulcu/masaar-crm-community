package bos24

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// ErrCommunityEdition is returned by all BOS24 methods in the community edition.
var ErrCommunityEdition = errors.New("real estate market data requires a Pro plan — visit https://masaar.io/pricing")

// Client is a no-op stub for the community edition.
type Client struct{}

func NewClient(token, baseURL string, rdb *redis.Client) *Client { return &Client{} }

func IsEnabled(token string) bool { return false }

func (c *Client) SearchProperties(ctx context.Context, query string, limit int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetTransactions(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetTransaction(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetTransactionAreas(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetTransactionsByProject(ctx context.Context, projectName string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetTransactionStats(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetEnrichedTransaction(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetBuilding(ctx context.Context, buildingID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) SearchBuildings(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetEjariStats(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetEjariRentals(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetEjariYield(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetNearbySchools(ctx context.Context, lat, lng, radiusKm float64, limit int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreas(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreaBuildings(ctx context.Context, areaSlug string, limit int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreaSummary(ctx context.Context, areaSlug string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreaTransactionSummary(ctx context.Context, areaName string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreaComparison(ctx context.Context, areas, propertyType string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreaLocation(ctx context.Context, areaName string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreaByID(ctx context.Context, areaID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetAreasWithLocations(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetPOIs(ctx context.Context, lat, lng, radiusKm float64, category string, limit int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetPOICategories(ctx context.Context) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetRentals(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetRentalStats(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetRentalAreas(ctx context.Context) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetRentalsByProject(ctx context.Context, projectName string, limit, offset int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetRentalsByBuilding(ctx context.Context, buildingName string, limit, offset int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetDevelopers(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetDeveloper(ctx context.Context, developerID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetProjects(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetProject(ctx context.Context, projectID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) SearchProjects(ctx context.Context, query string, limit int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetUnit(ctx context.Context, unitID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetUnits(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetLands(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetLand(ctx context.Context, landID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetValuation(ctx context.Context, valuationID int) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetValuations(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetMapConfig(ctx context.Context) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetMapBounds(ctx context.Context) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetPropertyHeatmap(ctx context.Context) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetMarketOverview(ctx context.Context, period, propertyType string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetPriceTrends(ctx context.Context, area, propertyType, granularity string) (map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetTopAreas(ctx context.Context, metric, propertyType string, limit int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) GetBrokers(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	return nil, ErrCommunityEdition
}
func (c *Client) DescribeProperty(ctx context.Context, params map[string]interface{}) (string, error) {
	return "", ErrCommunityEdition
}
