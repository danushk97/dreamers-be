package auction

// TournamentPlayerRegistration ties a registered player to a tournament/event.
// TeamID is set after assignment (e.g. auction win); nil/empty when unassigned.
type TournamentPlayerRegistration struct {
	ID             string
	TournamentID   string
	EventID        string
	PlayerID       string // canonical player entity id once normalized
	TeamID         string // tournament team registration id; empty until assigned
	PlayerInfo     RegistrationPlayerInfo // denormalized snapshot until fully normalized
	CreatedAt      int64 // unix milliseconds
}

// RegistrationPlayerInfo holds capture-time fields before full normalization to PlayerID.
type RegistrationPlayerInfo struct {
	DisplayName string
	Gender       string // optional until fully normalized
	DateOfBirthMs int64 // unix milliseconds; used for age screening even before PlayerID is normalized
}

// TournamentTeamRegistration is a team participating in a tournament/event.
type TournamentTeamRegistration struct {
	ID           string
	TournamentID string
	EventID      string
	TeamName     string
	TeamLogoURL  string
	CreatedAt    int64 // unix milliseconds
}
