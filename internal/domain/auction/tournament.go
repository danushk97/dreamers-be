package auction

// Tournament is a competition container (e.g. league season).
type Tournament struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SportID   string `json:"sportId"`
	StartDate int64  `json:"startDate"` // unix milliseconds
	EndDate   int64  `json:"endDate"`   // unix milliseconds
	CreatedAt int64  `json:"createdAt"` // unix milliseconds
	// LogoURL is a public URL for tournament artwork (e.g. hero / watermark).
	LogoURL string `json:"logoUrl"`
	// Sponsors is stored as JSONB; see TournamentSponsorGroup.
	Sponsors []TournamentSponsorGroup `json:"sponsors"`
}
