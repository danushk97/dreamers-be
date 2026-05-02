package auction

import "fmt"

// AuctionRules holds bid limits for one auction (persisted as auctions.rules JSONB).
// JSON keys match the persisted shape: MinBidAmount, MaxBidAmount (paisa).
type AuctionRules struct {
	MinBidAmount          int64 `json:"MinBidAmount"`
	MaxBidAmount          int64 `json:"MaxBidAmount"`
	MaxRetainPlayers      int   `json:"MaxRetainPlayers"`
	MaxRetainPlayerAmount int64 `json:"MaxRetainPlayerAmount"`
	MaxSubstitutePlayers  int   `json:"MaxSubstitutePlayers"`
}

// Validate checks internal consistency. Zero values mean "unset" for that bound.
func (r AuctionRules) Validate() error {
	if r.MinBidAmount < 0 || r.MaxBidAmount < 0 {
		return fmt.Errorf("MinBidAmount and MaxBidAmount must be non-negative")
	}
	if r.MaxRetainPlayers < 0 || r.MaxRetainPlayerAmount < 0 || r.MaxSubstitutePlayers < 0 {
		return fmt.Errorf("MaxRetainPlayers, MaxRetainPlayerAmount, and MaxSubstitutePlayers must be non-negative")
	}
	if r.MaxBidAmount > 0 && r.MinBidAmount > r.MaxBidAmount {
		return fmt.Errorf("MinBidAmount must be <= MaxBidAmount")
	}
	return nil
}

// TeamWalletMaxBidOK returns true if walletMax is 0 (no per-team cap) or lies within [min,max] when bounds are set.
func (r AuctionRules) TeamWalletMaxBidOK(walletMax int64) bool {
	if walletMax <= 0 {
		return true
	}
	if r.MinBidAmount > 0 && walletMax < r.MinBidAmount {
		return false
	}
	if r.MaxBidAmount > 0 && walletMax > r.MaxBidAmount {
		return false
	}
	return true
}
