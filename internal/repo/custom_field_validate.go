package repo

import (
	"fmt"
	"strconv"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// ValidateCustomValue checks a non-empty value against its field's type.
func ValidateCustomValue(f domain.CustomField, value string) error {
	if len(value) > 2000 {
		return fmt.Errorf("%w: %s is too long", ErrCustomFieldInvalid, f.Name)
	}
	switch f.Type {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("%w: %s must be a number", ErrCustomFieldInvalid, f.Name)
		}
	case "date":
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("%w: %s must be a date (YYYY-MM-DD)", ErrCustomFieldInvalid, f.Name)
		}
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("%w: %s must be true or false", ErrCustomFieldInvalid, f.Name)
		}
	case "select":
		for _, o := range f.Options {
			if o == value {
				return nil
			}
		}
		return fmt.Errorf("%w: %s must be one of its options", ErrCustomFieldInvalid, f.Name)
	}
	return nil
}
