package auction

// TournamentEvent links an event to a tournament and optional parent event (e.g. subgroup / round).
type TournamentEvent struct {
	ID            string
	TournamentID  string
	EventID       string
	ParentEventID string // empty if none
	CreatedAt     int64 // unix milliseconds
}
