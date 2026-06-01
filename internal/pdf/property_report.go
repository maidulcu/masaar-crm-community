package pdf

import "time"

type PropertyReportData struct {
	CompanyName    string
	CompanyAddress string
	CompanyPhone   string
	CompanyEmail   string
	AgentName      string
	ClientName     string
	Area           string
	PropertyType   string
	Bedrooms       string
	BudgetMin      float64
	BudgetMax      float64
	Currency       string
	AreaSummary    map[string]interface{}
	Comparables    []map[string]interface{}
	YieldData      []map[string]interface{}
	POIs           []map[string]interface{}
	AIDescription  string
	GeneratedAt    time.Time
}

func GeneratePropertyReport(d PropertyReportData) ([]byte, error) {
	return nil, errProOnly
}
