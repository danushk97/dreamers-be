package auction

// Bid is an offer on an auction lot. Wallet debits happen on sale settlement, not on bid insert.
type Bid struct {
	ID                         string
	AuctionPlayerID            string
	TeamRegistrationID         string // TournamentTeamRegistration id
	Amount                     int64
	RecordedAt                 int64 // unix milliseconds
}
