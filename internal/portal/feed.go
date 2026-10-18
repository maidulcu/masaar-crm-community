package portal

import (
	"encoding/xml"
	"io"
	"strconv"
	"time"
)

// FeedListing is one published listing in the feed.
type FeedListing struct {
	ID              string
	Reference       string
	PermitNumber    string
	PermitExpiry    *time.Time
	Title           string
	Description     string
	PropertyType    string
	ListingType     string // sale | rent
	RentPeriod      string
	Price           float64
	Currency        string
	Emirate         string
	City            string
	Community       string
	SubCommunity    string
	Area            string
	Bedrooms        int
	Bathrooms       int
	SizeSqft        float64
	Furnishing      string
	Amenities       []string
	Photos          []string
	Latitude        *float64
	Longitude       *float64
	AgentName       string
	AgentEmail      string
	AgentPhone      string
	AgentBRN        string
	UpdatedAt       time.Time
	PublicURL       string
	AvailableFrom   *time.Time
	CompanyName     string
	CompanyORN      string
	CompanyLicence  string
	CompanyPhone    string
	CompanyEmail    string
	CompanyAddress  string
	CompanyLogoURL  string
	CompanyCountry  string
	CompanyCurrency string
}

// The XML below is Masaar's own format ("masaar-listings/1"). Element names are stable;
// new elements may be added.
type xmlFeed struct {
	XMLName   xml.Name     `xml:"listings"`
	Format    string       `xml:"format,attr"`
	Generated string       `xml:"generated_at,attr"`
	Count     int          `xml:"count,attr"`
	Broker    xmlBroker    `xml:"broker"`
	Listings  []xmlListing `xml:"listing"`
}

type xmlBroker struct {
	Name    string `xml:"name"`
	ORN     string `xml:"orn,omitempty"`
	Licence string `xml:"trade_license,omitempty"`
	Phone   string `xml:"phone,omitempty"`
	Email   string `xml:"email,omitempty"`
	Address string `xml:"address,omitempty"`
	Logo    string `xml:"logo_url,omitempty"`
}

type xmlListing struct {
	ID            string     `xml:"id,attr"`
	UpdatedAt     string     `xml:"updated_at,attr"`
	Reference     string     `xml:"reference"`
	Permit        string     `xml:"permit_number,omitempty"`
	PermitExpiry  string     `xml:"permit_expiry,omitempty"`
	Offering      string     `xml:"offering"` // sale | rent
	PropertyType  string     `xml:"property_type"`
	Title         string     `xml:"title"`
	Description   string     `xml:"description"`
	Price         xmlPrice   `xml:"price"`
	Location      xmlLoc     `xml:"location"`
	Bedrooms      int        `xml:"bedrooms"`
	Bathrooms     int        `xml:"bathrooms"`
	SizeSqft      string     `xml:"size_sqft,omitempty"`
	Furnishing    string     `xml:"furnishing,omitempty"`
	AvailableFrom string     `xml:"available_from,omitempty"`
	Amenities     []string   `xml:"amenities>amenity,omitempty"`
	Photos        []xmlPhoto `xml:"photos>photo,omitempty"`
	Agent         xmlAgent   `xml:"agent"`
	URL           string     `xml:"url,omitempty"`
}

type xmlPrice struct {
	Currency string `xml:"currency,attr"`
	Period   string `xml:"period,attr,omitempty"`
	Value    string `xml:",chardata"`
}

type xmlLoc struct {
	Emirate      string `xml:"emirate"`
	City         string `xml:"city,omitempty"`
	Community    string `xml:"community,omitempty"`
	SubCommunity string `xml:"sub_community,omitempty"`
	Area         string `xml:"area,omitempty"`
	Latitude     string `xml:"latitude,omitempty"`
	Longitude    string `xml:"longitude,omitempty"`
}

type xmlPhoto struct {
	URL string `xml:",chardata"`
}

type xmlAgent struct {
	Name  string `xml:"name"`
	Email string `xml:"email,omitempty"`
	Phone string `xml:"phone,omitempty"`
	BRN   string `xml:"brn,omitempty"`
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func day(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// WriteXML writes the feed. All text is escaped by encoding/xml.
func WriteXML(w io.Writer, items []FeedListing, now time.Time) error {
	f := xmlFeed{Format: "masaar-listings/1", Generated: now.UTC().Format(time.RFC3339), Count: len(items)}
	if len(items) > 0 {
		c := items[0]
		f.Broker = xmlBroker{Name: c.CompanyName, ORN: c.CompanyORN, Licence: c.CompanyLicence, Phone: c.CompanyPhone,
			Email: c.CompanyEmail, Address: c.CompanyAddress, Logo: c.CompanyLogoURL}
	}
	for _, it := range items {
		x := xmlListing{
			ID: it.ID, UpdatedAt: it.UpdatedAt.UTC().Format(time.RFC3339), Reference: it.Reference,
			Permit: it.PermitNumber, PermitExpiry: day(it.PermitExpiry), Offering: it.ListingType,
			PropertyType: it.PropertyType, Title: it.Title, Description: it.Description,
			Price:    xmlPrice{Currency: it.Currency, Period: it.RentPeriod, Value: num(it.Price)},
			Location: xmlLoc{Emirate: it.Emirate, City: it.City, Community: it.Community, SubCommunity: it.SubCommunity, Area: it.Area},
			Bedrooms: it.Bedrooms, Bathrooms: it.Bathrooms, Furnishing: it.Furnishing, AvailableFrom: day(it.AvailableFrom),
			Amenities: it.Amenities, Agent: xmlAgent{Name: it.AgentName, Email: it.AgentEmail, Phone: it.AgentPhone, BRN: it.AgentBRN},
			URL: it.PublicURL,
		}
		if it.Reference == "" {
			x.Reference = it.ID
		}
		if it.SizeSqft > 0 {
			x.SizeSqft = num(it.SizeSqft)
		}
		if it.Latitude != nil && it.Longitude != nil {
			x.Location.Latitude, x.Location.Longitude = num(*it.Latitude), num(*it.Longitude)
		}
		for _, p := range it.Photos {
			x.Photos = append(x.Photos, xmlPhoto{URL: p})
		}
		f.Listings = append(f.Listings, x)
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(f); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
