package auction

// TournamentSponsorEntry is one sponsor name + logo within a group.
type TournamentSponsorEntry struct {
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// TournamentSponsorGroup is a labeled section of sponsors (e.g. "Title", "Partners").
// JSON: { "type": string, "sponsors": [ { "name", "logo" }, ... ] }
type TournamentSponsorGroup struct {
	Type     string                     `json:"type"`
	Sponsors []TournamentSponsorEntry `json:"sponsors"`
}
