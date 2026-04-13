package auctionservice

import (
	"errors"
)

// BidPlacementHints is returned with bid validation responses so the UI can cap raises without guessing.
type BidPlacementHints struct {
	DerivedMaxBidAmount int64 `json:"derivedMaxBidAmount"`
	MinBidAmount        int64 `json:"minBidAmount"`
	// MaxBidAmount is the configured per-team single-bid cap on the wallet (paise); 0 = unset.
	MaxBidAmount int64 `json:"maxBidAmount"`
	// RemainingPlayersToPick is how many more players this team can still buy (0 when max is unknown/unset).
	RemainingPlayersToPick int `json:"remainingPlayersToPick"`
}

// ValidationError indicates a business validation failure (client fault, 400).
type ValidationError struct {
	Err   error
	Hints *BidPlacementHints // optional; set for bid placement when limits are known
}

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

