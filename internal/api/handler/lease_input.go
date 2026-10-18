package handler

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// leaseDateKeys are the lease JSON fields holding dates.
var leaseDateKeys = []string{
	"start_date", "end_date", "renewal_start_date", "renewal_end_date", "move_out_date",
	"move_out_inspection_date", "signed_by_landlord_date", "signed_by_tenant_date",
	"termination_date", "last_generated_payment_date",
}

var (
	validFrequencies = map[domain.PaymentFrequency]bool{
		domain.FrequencyMonthly: true, domain.FrequencyQuarterly: true, domain.FrequencySemiAnnual: true,
		domain.FrequencyAnnual: true, "bi_monthly": true, // the web form offers bi_monthly
	}
	validLeaseStatuses = map[domain.LeaseStatus]bool{
		domain.LeaseStatusActive: true, domain.LeaseStatusRenewed: true,
		domain.LeaseStatusTerminated: true, domain.LeaseStatusExpired: true,
	}
)

const maxLeaseYears = 20

// validateLease normalises and validates the client-editable fields of a lease in place.
// Money, dates and the day of the month used to reach the database unchecked, where start >= end
// or a non-positive rent surfaced as an opaque 422 "invalid data".
func validateLease(l *domain.Lease) error {
	if l.StartDate.IsZero() || l.EndDate.IsZero() {
		return errors.New("start_date and end_date are required")
	}
	if !l.StartDate.Before(l.EndDate) {
		return errors.New("end_date must be after start_date")
	}
	if l.EndDate.After(l.StartDate.AddDate(maxLeaseYears, 0, 0)) {
		return errors.New("a lease cannot be longer than 20 years")
	}
	if l.MonthlyRent <= 0 || badMoney(l.MonthlyRent) || math.IsNaN(l.MonthlyRent) {
		return errors.New("monthly_rent must be greater than 0")
	}
	if badMoney(l.SecurityDeposit) || badMoney(l.UtilityCharges) {
		return errors.New("security_deposit and utility_charges must not be negative")
	}
	if l.LateFeePct < 0 || l.LateFeePct > 100 || math.IsNaN(l.LateFeePct) {
		return errors.New("late_fee_percent must be between 0 and 100")
	}
	cur, ok := normalizeCurrency(l.Currency)
	if !ok {
		return errors.New("currency must be a 3-letter code such as AED")
	}
	l.Currency = cur
	if l.PaymentFrequency == "" {
		l.PaymentFrequency = domain.FrequencyMonthly
	}
	if !validFrequencies[l.PaymentFrequency] {
		return errors.New("payment_frequency must be one of: monthly, quarterly, semi_annual, annual, bi_monthly")
	}
	if l.PaymentDayOfMonth == 0 {
		l.PaymentDayOfMonth = 1
	}
	if l.PaymentDayOfMonth < 1 || l.PaymentDayOfMonth > 31 {
		return errors.New("payment_day_of_month must be between 1 and 31")
	}
	if l.NoticePeriodDays < 0 || l.NoticePeriodDays > 730 {
		return errors.New("notice_period_days must be between 0 and 730")
	}
	if l.Status == "" {
		l.Status = domain.LeaseStatusActive
	}
	if !validLeaseStatuses[l.Status] {
		return errors.New("status must be one of: active, renewed, terminated, expired")
	}
	if tooLong(l.EjariNumber, 100) || tooLong(l.TerminationReason, 255) || tooLong(l.Notes, 10000) {
		return errors.New("a field is too long")
	}
	return nil
}

// dateOnly is t's calendar date at 00:00 UTC (the leases columns are DATE).
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// validateLeaseTemplate applies defaults and validates a lease template in place.
func validateLeaseTemplate(t *domain.LeaseTemplate) error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" || t.PaymentFrequency == "" {
		return errors.New("name and payment_frequency are required")
	}
	if tooLong(t.Name, 255) || tooLong(t.Description, 5000) || tooLong(t.TermsConditions, 100000) {
		return errors.New("a field is too long")
	}
	if !validFrequencies[t.PaymentFrequency] {
		return errors.New("payment_frequency must be one of: monthly, quarterly, semi_annual, annual, bi_monthly")
	}
	if t.PaymentDayOfMonth == 0 {
		t.PaymentDayOfMonth = 1
	}
	if t.PaymentDayOfMonth < 1 || t.PaymentDayOfMonth > 31 {
		return errors.New("payment_day_of_month must be between 1 and 31")
	}
	if t.DefaultSecurityDepositPct < 0 || t.DefaultSecurityDepositPct > 100 || t.DefaultLateFeePercent < 0 || t.DefaultLateFeePercent > 100 {
		return errors.New("deposit and late-fee percentages must be between 0 and 100")
	}
	if badMoney(t.DefaultUtilityCharges) {
		return errors.New("default_utility_charges must not be negative")
	}
	if t.DefaultLeaseDurationMonths < 0 || t.DefaultLeaseDurationMonths > maxLeaseYears*12 ||
		t.DefaultRenewalDurationMonths < 0 || t.DefaultRenewalDurationMonths > maxLeaseYears*12 ||
		t.DefaultNoticePeriodDays < 0 || t.DefaultNoticePeriodDays > 730 {
		return errors.New("durations are out of range")
	}
	if t.Status == "" {
		t.Status = "active"
	}
	if t.Status != "active" && t.Status != "inactive" {
		return errors.New("status must be active or inactive")
	}
	return nil
}
