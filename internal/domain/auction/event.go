package auction

// Event is a competition format or phase (e.g. knockout, draft round).
type Event struct {
	ID        string
	Name      string
	Attrs     EventAttrs
	CreatedAt int64 // unix milliseconds
}

// EventAttrs carries event-specific configuration; only team-event auction rules are defined for now.
type EventAttrs struct {
	TeamEventRules *TeamEventRules
}

// TeamEventRules applies when the event is a team-based competition with optional auction.
type TeamEventRules struct {
	MinPlayersPerTeam int
	MaxPlayersPerTeam int
	IsAuction         bool
	// Amounts are in the smallest currency unit (e.g. paise) to avoid float drift.
	BaseBid int64
	MaxBid  int64
}
