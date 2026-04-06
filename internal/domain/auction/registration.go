package auction

// TournamentPlayerRegistration ties a registered player to a tournament event.
// TeamID is set after assignment (e.g. auction win); empty when unassigned.
type TournamentPlayerRegistration struct {
	ID                string
	TournamentID      string
	TournamentEventID string
	PlayerID          string // canonical player entity id
	TeamID            string // tournament team registration id; empty until assigned
	SerialNumber      int    // auto-assigned per insert (DB sequence)
	CreatedAt         int64  // unix milliseconds
}

// TournamentTeamRegistration is a team participating in a tournament event.
type TournamentTeamRegistration struct {
	ID                string
	TournamentID      string
	TournamentEventID string
	TeamName          string
	TeamLogoURL      string
	CreatedAt         int64 // unix milliseconds
}

