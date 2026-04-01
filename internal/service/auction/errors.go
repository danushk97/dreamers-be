package auctionservice

import (
	"errors"
)

// ValidationError indicates a business validation failure (client fault, 400).
type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

