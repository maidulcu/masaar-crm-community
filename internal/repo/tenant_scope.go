package repo

import "errors"

// ErrInvalidPhone is returned for a phone number that cannot be normalised to E.164.
var ErrInvalidPhone = errors.New("invalid phone number")

// ErrForeignReference is returned when a write references an entity (property, tenant,
// contact, ...) that does not exist in the caller's company.
var ErrForeignReference = errors.New("referenced entity not found in this company")
