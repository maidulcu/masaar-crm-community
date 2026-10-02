package repo

import "errors"

// ErrForeignReference is returned when a write references an entity (property, tenant,
// contact, ...) that does not exist in the caller's company.
var ErrForeignReference = errors.New("referenced entity not found in this company")
