package auction

// AuctionMode identifies how bidding is run (extend as you add modes).
type AuctionMode string

const (
	AuctionModeOpen       AuctionMode = "open"        // default / English-style
	AuctionModeSealed     AuctionMode = "sealed"
	AuctionModeRoundRobin AuctionMode = "round_robin"
)

// AuctionRunMode distinguishes sandbox vs production auctions (DB column run_mode).
type AuctionRunMode string

const (
	AuctionRunModeTest AuctionRunMode = "test" // default; allows reset
	AuctionRunModeLive AuctionRunMode = "live"
)

// Auction is a bidding session for a tournament event (tournament_events row).
type Auction struct {
	ID                string
	TournamentID      string
	TournamentEventID string
	Mode              AuctionMode // bidding style (DB column mode): open, sealed, etc.
	RunMode           AuctionRunMode `json:"runMode"` // test vs live (DB column run_mode)
	// Status is created → running → completed (DB column auctions.status). Bids/settlement/lot writes require running.
	Status AuctionStatus `json:"status"`
	// DisplayAuctionPlayerID is the lot row the console/relay should show (any status). Empty = unset.
	DisplayAuctionPlayerID string `json:"displayAuctionPlayerId,omitempty"`
	// FilterPresets is a saved list of category filters (persist as JSON in DB).
	FilterPresets []FilterPreset
	// Rules is per-auction min/max single bid (paisa); JSON column auctions.rules.
	Rules     AuctionRules `json:"rules"`
	CreatedAt int64        // unix milliseconds
}

// FilterPreset is a named category filter that the UI can reuse.
type FilterPreset struct {
	ID        string             `json:"id"` // optional client-generated id
	Name      string             `json:"name"`
	Filter    RegistrationFilter `json:"filter"`
	CreatedAt int64              `json:"createdAt"` // unix milliseconds
}

// FilterForPreset returns the registration filter for a saved preset on this auction, if found.
func (a *Auction) FilterForPreset(presetID string) (RegistrationFilter, bool) {
	if a == nil || presetID == "" {
		return RegistrationFilter{}, false
	}
	for _, p := range a.FilterPresets {
		if p.ID == presetID {
			return p.Filter, true
		}
	}
	return RegistrationFilter{}, false
}
