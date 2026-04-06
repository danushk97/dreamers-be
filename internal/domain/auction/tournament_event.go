package auction

// TournamentEvent is a competition phase/format scoped to one tournament (name + rules + optional parent round).
type TournamentEvent struct {
	ID            string
	TournamentID  string
	Name          string
	Attrs         EventAttrs
	ParentEventID string // empty if none; references another tournament_events row
	CreatedAt     int64  // unix milliseconds
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
