package pdf

import "time"

type BrochureData struct {
	CompanyName    string
	CompanyAddress string
	CompanyPhone   string
	CompanyEmail   string
	LogoURL        string
	PrimaryColor   string
	Disclaimer     string
	AgentName      string
	AgentPhone     string
	AgentEmail     string
	ListingID      string
	Title          string
	ReferenceNumber string
	PropertyType   string
	ListingType    string
	Price          float64
	Currency       string
	RentPeriod     string
	Bedrooms       int
	Bathrooms      int
	TotalSqft      float64
	ParkingSpaces  int
	Furnishing     string
	YearBuilt      int
	Area           string
	Community      string
	City           string
	Emirate        string
	Description    string
	Amenities      []string
	CoverImageURL  string
	AvailableFrom  *time.Time
	GeneratedAt    time.Time
}

func GenerateBrochure(d BrochureData) ([]byte, error) {
	return nil, errProOnly
}
