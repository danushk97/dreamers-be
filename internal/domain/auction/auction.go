package auction

// AuctionMode identifies how bidding is run (extend as you add modes).
type AuctionMode string

const (
	AuctionModeOpen       AuctionMode = "open"        // default / English-style
	AuctionModeSealed     AuctionMode = "sealed"
	AuctionModeRoundRobin AuctionMode = "round_robin"
)

// Auction is a bidding session for a tournament/event.
type Auction struct {
	ID           string
	TournamentID string
	EventID      string
	Mode         AuctionMode
	// FilterPresets is a saved list of category filters (persist as JSON in DB).
	FilterPresets []FilterPreset
	CreatedAt    int64 // unix milliseconds
}

// FilterPreset is a named category filter that the UI can reuse.
type FilterPreset struct {
	ID        string // optional client-generated id
	Name      string
	Filter    RegistrationFilter
	CreatedAt int64 // unix milliseconds
}
