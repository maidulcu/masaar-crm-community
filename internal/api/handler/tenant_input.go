package handler

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// The web forms and the Go constants disagree on some spellings (emirates_id vs emirati_id,
// cheque vs check, ...). The columns are free text, so both are accepted rather than rejecting
// what the UI itself sends.
var (
	validIDTypes = map[domain.IDType]bool{
		"emirati_id": true, "emirates_id": true, "passport": true, "visa": true,
		"driving_license": true, "trade_license": true,
	}
	validEmployment = map[domain.EmploymentStatus]bool{
		"": true, "employed": true, "self_employed": true, "unemployed": true, "retired": true, "student": true,
	}
	validTenantStatus = map[domain.TenantStatus]bool{
		domain.TenantStatusActive: true, domain.TenantStatusInactive: true, domain.TenantStatusBlacklisted: true,
	}
)

// tenantDateKeys are the tenant JSON fields holding dates.
var tenantDateKeys = []string{"id_expiry_date", "verification_date"}

// validateTenant applies defaults and validates the client-editable fields of a tenant in place.
// The verification fields are deliberately not touched: only POST /tenants/:id/verify sets them.
func validateTenant(t *domain.Tenant) error {
	t.FullNameEN = strings.TrimSpace(t.FullNameEN)
	if t.FullNameEN == "" || t.IDType == "" {
		return errors.New("full_name_en and id_type are required")
	}
	if tooLong(t.FullNameEN, 255) || tooLong(t.FullNameAR, 255) {
		return errors.New("name is too long")
	}
	t.IDType = domain.IDType(strings.ToLower(strings.TrimSpace(string(t.IDType))))
	if !validIDTypes[t.IDType] {
		return errors.New("id_type must be one of: emirates_id, passport, visa, driving_license, trade_license")
	}
	t.IDNumber = strings.TrimSpace(t.IDNumber)
	if tooLong(t.IDNumber, 100) {
		return errors.New("id_number is too long")
	}
	t.Email = strings.TrimSpace(t.Email)
	if t.Email != "" {
		if addr, err := mail.ParseAddress(t.Email); err != nil || addr.Address != t.Email || len(t.Email) > 254 {
			return errors.New("email is not a valid address")
		}
	}
	if tooLong(t.Phone, 30) || tooLong(t.PhoneWA, 30) || tooLong(t.EmergencyContactPhone, 30) {
		return errors.New("a phone number is too long")
	}
	if !validEmployment[t.EmploymentStatus] {
		return errors.New("employment_status must be one of: employed, self_employed, unemployed, retired, student")
	}
	if t.AnnualIncome < 0 || badMoney(t.AnnualIncome) {
		return errors.New("annual_income must not be negative")
	}
	if t.IncomeCurrency == "" {
		t.IncomeCurrency = "AED"
	}
	cur, ok := normalizeCurrency(t.IncomeCurrency)
	if !ok {
		return errors.New("income_currency must be a 3-letter code such as AED")
	}
	t.IncomeCurrency = cur
	if t.Status == "" {
		t.Status = domain.TenantStatusActive
	}
	if !validTenantStatus[t.Status] {
		return errors.New("status must be one of: active, inactive, blacklisted")
	}
	if tooLong(t.Notes, 10000) || tooLong(t.PermanentAddress, 1000) {
		return errors.New("notes or address are too long")
	}
	return nil
}
