package portal

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestWriteXMLEscapesAndParses(t *testing.T) {
	exp := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	items := []FeedListing{{
		ID: "abc", Reference: "MAS-1", PermitNumber: "7114", PermitExpiry: &exp, Title: `Villa <b>"sea" & sun</b>`,
		PropertyType: "villa", ListingType: "sale", Price: 4500000, Currency: "AED", Emirate: "Dubai",
		Photos: []string{"https://cdn.example/1.jpg"}, AgentName: "Agent", AgentBRN: "BRN-1",
		CompanyName: "Masaar Realty", CompanyORN: "12345", UpdatedAt: time.Now(),
	}}
	var buf bytes.Buffer
	if err := WriteXML(&buf, items, time.Now()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "<b>") || !strings.Contains(out, "&lt;b&gt;") {
		t.Fatalf("title not escaped:\n%s", out)
	}
	var back xmlFeed
	if err := xml.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("feed is not valid XML: %v", err)
	}
	if back.Count != 1 || back.Listings[0].Permit != "7114" || back.Listings[0].PermitExpiry != "2027-01-31" ||
		back.Listings[0].Price.Value != "4500000" || back.Broker.ORN != "12345" || len(back.Listings[0].Photos) != 1 {
		t.Fatalf("round trip = %+v", back)
	}
}
