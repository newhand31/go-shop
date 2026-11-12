package service

import (
	"errors"
	"fmt"
)

// ErrProductNotFound is a sentinel error: callers compare against it with errors.Is.
// In Go an error is just a value you return, so "not found" is a value too, not an exception type.
var ErrProductNotFound = errors.New("product not found")

// ValidationError is a typed error: callers pull it out with errors.As to read its fields.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}
