package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// propertyTypes are the types the web forms offer. The columns are free-form varchar, so without
// this check anything could be stored and the UI filters would never match it.
var propertyTypes = map[string]bool{
	"apartment": true, "villa": true, "townhouse": true, "commercial": true,
	"land": true, "studio": true, "warehouse": true, "office": true,
}

var validListingStatuses = map[domain.ListingStatus]bool{
	domain.ListingStatusDraft: true, domain.ListingStatusPublished: true, domain.ListingStatusSold: true,
	domain.ListingStatusRented: true, domain.ListingStatusExpired: true, domain.ListingStatusWithdrawn: true,
}

const (
	maxPropertyMoney = 9_999_999_999_999.99 // numeric(15,2)
	maxCountField    = 100000               // generous cap for unit counts
)

// blankDatesToNull rewrites `"key": ""` to `"key": null` for the given keys. The forms send an
// empty string for an empty date input, which encoding/json refuses to parse into *time.Time, so
// every save of a record without that date failed with "invalid request body".
func blankDatesToNull(body []byte, keys ...string) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	changed := false
	for _, k := range keys {
		if raw, ok := m[k]; ok && string(bytes.TrimSpace(raw)) == `""` {
			m[k] = json.RawMessage("null")
			changed = true
		}
	}
	if !changed {
		return body
	}
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

func badMoney(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > maxPropertyMoney
}

// validateListing normalises and validates the client-editable fields of a listing in place.
func validateListing(l *domain.Listing) error {
	l.Title = strings.TrimSpace(l.Title)
	if l.Title == "" {
		return errors.New("title is required")
	}
	if tooLong(l.Title, 255) {
		return errors.New("title is too long")
	}
	l.PropertyType = domain.PropertyType(strings.ToLower(strings.TrimSpace(string(l.PropertyType))))
	if !propertyTypes[string(l.PropertyType)] {
		return errors.New("property_type must be one of: apartment, villa, townhouse, commercial, land, studio, warehouse, office")
	}
	if l.ListingType != domain.ListingTypeRent && l.ListingType != domain.ListingTypeSale {
		return errors.New("listing_type must be rent or sale")
	}
	if l.Price <= 0 || badMoney(l.Price) {
		return errors.New("price must be greater than 0")
	}
	cur, ok := normalizeCurrency(l.Currency)
	if !ok {
		return errors.New("currency must be a 3-letter code such as AED")
	}
	l.Currency = cur
	for name, v := range map[string]*string{
		"area": &l.Area, "community": &l.Community, "subcommunity": &l.Subcommunity, "city": &l.City,
		"reference_number": &l.ReferenceNumber,
	} {
		if tooLong(*v, 100) {
			return errors.New(name + " is too long")
		}
	}
	if tooLong(l.Emirate, 50) || tooLong(l.OwnerName, 255) || tooLong(l.OwnerPhone, 50) || tooLong(l.OwnerEmail, 255) {
		return errors.New("a field is too long")
	}
	if tooLong(l.Description, 20000) {
		return errors.New("description is too long")
	}
	if l.Latitude != nil && (*l.Latitude < -90 || *l.Latitude > 90) {
		return errors.New("latitude must be between -90 and 90")
	}
	if l.Longitude != nil && (*l.Longitude < -180 || *l.Longitude > 180) {
		return errors.New("longitude must be between -180 and 180")
	}
	if l.Bedrooms < 0 || l.Bedrooms > 1000 || l.Bathrooms < 0 || l.Bathrooms > 1000 || l.ParkingSpaces < 0 || l.ParkingSpaces > 1000 {
		return errors.New("bedrooms, bathrooms and parking_spaces must be between 0 and 1000")
	}
	if badMoney(l.TotalSqft) || badMoney(l.PlotSqft) {
		return errors.New("total_sqft and plot_sqft must not be negative")
	}
	if l.YearBuilt != nil && (*l.YearBuilt < 1800 || *l.YearBuilt > 2200) {
		return errors.New("year_built is out of range")
	}
	if len(l.ImageURLs) > 100 || len(l.Amenities) > 200 {
		return errors.New("too many images or amenities")
	}
	return nil
}

var (
	validPropertyStatuses = map[domain.PropertyStatus]bool{
		domain.PropertyStatusActive: true, domain.PropertyStatusInactive: true,
		domain.PropertyStatusSold: true, domain.PropertyStatusMaintenance: true,
	}
	validOccupancy = map[domain.OccupancyStatus]bool{
		domain.OccupancyVacant: true, domain.OccupancyOccupied: true, domain.OccupancyMaintenance: true,
	}
)

// validateRentalProperty applies defaults and validates a rental property in place. Zero values
// used to be inserted literally (status ”, currency ”, 0 units) because the INSERT lists every
// column, so the column defaults never applied; and an occupied count above the unit count failed
// the table's CHECK as an opaque "invalid data".
func validateRentalProperty(p *domain.RentalProperty) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("name and property_type are required")
	}
	if tooLong(p.Name, 255) {
		return errors.New("name is too long")
	}
	p.PropertyType = domain.PropertyType(strings.ToLower(strings.TrimSpace(string(p.PropertyType))))
	if p.PropertyType == "" {
		return errors.New("name and property_type are required")
	}
	if !propertyTypes[string(p.PropertyType)] {
		return errors.New("property_type must be one of: apartment, villa, townhouse, commercial, land, studio, warehouse, office")
	}
	if p.UnitsCount == 0 {
		p.UnitsCount = 1
	}
	if p.UnitsCount < 1 || p.UnitsCount > maxCountField || p.TotalOccupiedUnits < 0 {
		return errors.New("units_count must be at least 1 and total_occupied_units cannot be negative")
	}
	if p.TotalOccupiedUnits > p.UnitsCount {
		return errors.New("total_occupied_units cannot exceed units_count")
	}
	if p.Status == "" {
		p.Status = domain.PropertyStatusActive
	}
	if !validPropertyStatuses[p.Status] {
		return errors.New("status must be one of: active, inactive, sold, maintenance")
	}
	if p.OccupancyStatus == "" {
		p.OccupancyStatus = domain.OccupancyVacant
	}
	if !validOccupancy[p.OccupancyStatus] {
		return errors.New("occupancy_status must be one of: vacant, occupied, maintenance")
	}
	cur, ok := normalizeCurrency(p.Currency)
	if !ok {
		return errors.New("currency must be a 3-letter code such as AED")
	}
	p.Currency = cur
	if badMoney(p.PurchasePrice) || badMoney(p.MarketValue) || badMoney(p.TotalSqft) {
		return errors.New("purchase_price, market_value and total_sqft must be between 0 and 9999999999999.99")
	}
	if p.Bedrooms < 0 || p.Bathrooms < 0 || p.ParkingSpaces < 0 || p.Bedrooms > 1000 || p.Bathrooms > 1000 || p.ParkingSpaces > 1000 {
		return errors.New("bedrooms, bathrooms and parking_spaces must be between 0 and 1000")
	}
	for name, v := range map[string]struct {
		s   string
		max int
	}{
		"area": {p.Area, 100}, "city": {p.City, 100}, "emirate": {p.Emirate, 50}, "street_address": {p.StreetAddress, 255},
		"building_number": {p.BuildingNumber, 50}, "unit_number": {p.UnitNumber, 50}, "postal_code": {p.PostalCode, 20},
		"title_deed_number": {p.TitleDeedNumber, 100}, "municipality_registration": {p.MunicipalityRegNum, 100},
		"property_deed_url": {p.PropertyDeedURL, 500},
	} {
		if tooLong(v.s, v.max) {
			return errors.New(name + " is too long")
		}
	}
	if tooLong(p.Description, 20000) || len(p.Amenities) > 200 {
		return errors.New("description or amenities are too long")
	}
	return nil
}
