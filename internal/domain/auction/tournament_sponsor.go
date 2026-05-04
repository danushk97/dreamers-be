package auction

// TournamentSponsorEntry is one sponsor name + logo within a group.
type TournamentSponsorEntry struct {
	Name             string `json:"name"`
	Logo             string `json:"logo"`
	BackgroundColour string `json:"backgroundColour,omitempty"`
}

// TournamentSponsorGroup is a labeled section of sponsors (e.g. "Title", "Partners").
// JSON: { "type": string, "sponsors": [ { "name", "logo", "backgroundColour?" }, ... ] }
type TournamentSponsorGroup struct {
	Type     string                   `json:"type"`
	Sponsors []TournamentSponsorEntry `json:"sponsors"`
}
