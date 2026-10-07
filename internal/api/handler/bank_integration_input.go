package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

var (
	validIntegrationTypes = map[string]bool{"manual": true, "api": true, "file_import": true}
	validIntegrationState = map[string]bool{"active": true, "inactive": true, "error": true}
)

// validateBankIntegration applies defaults and validates the client-editable fields in place.
func validateBankIntegration(bi *domain.BankIntegration) error {
	bi.BankName = strings.TrimSpace(bi.BankName)
	bi.IntegrationType = strings.ToLower(strings.TrimSpace(bi.IntegrationType))
	bi.IBAN = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(bi.IBAN), " ", ""))
	if bi.BankName == "" || bi.AccountNumber == "" || bi.IntegrationType == "" {
		return errors.New("bank_name, account_number, and integration_type are required")
	}
	if !validIntegrationTypes[bi.IntegrationType] {
		return errors.New("integration_type must be one of: manual, api, file_import")
	}
	// The column default only applies when the column is omitted; '' was being stored.
	if bi.Status == "" {
		bi.Status = "active"
	}
	if !validIntegrationState[bi.Status] {
		return errors.New("status must be one of: active, inactive, error")
	}
	if bi.SyncIntervalHours == 0 {
		bi.SyncIntervalHours = 24
	}
	if bi.SyncIntervalHours < 1 || bi.SyncIntervalHours > 24*30 {
		return errors.New("sync_interval_hours must be between 1 and 720")
	}
	if tooLong(bi.BankName, 255) || tooLong(bi.BankCode, 50) || tooLong(bi.AccountNumber, 100) ||
		tooLong(bi.AccountName, 255) || tooLong(bi.IBAN, 50) || tooLong(bi.APIEndpoint, 500) {
		return errors.New("a field is too long")
	}
	// An IBAN is 15-34 letters/digits (UAE: 23).
	if bi.IBAN != "" {
		if len(bi.IBAN) < 15 || len(bi.IBAN) > 34 {
			return errors.New("iban is not valid")
		}
		for _, r := range bi.IBAN {
			if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				return errors.New("iban is not valid")
			}
		}
	}
	if bi.APIEndpoint != "" {
		u, err := url.Parse(bi.APIEndpoint)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" {
			return errors.New("api_endpoint must be an https URL")
		}
	}
	return nil
}
