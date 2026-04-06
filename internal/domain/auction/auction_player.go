package auction

// AuctionPlayerStatus is the lifecycle of a lot in an auction.
type AuctionPlayerStatus string

const (
	AuctionPlayerPending AuctionPlayerStatus = "pending"
	AuctionPlayerActive  AuctionPlayerStatus = "active"
	AuctionPlayerSold    AuctionPlayerStatus = "sold"
	AuctionPlayerUnsold  AuctionPlayerStatus = "unsold"
	AuctionPlayerSkipped AuctionPlayerStatus = "skipped"
)

// AuctionPlayer is one sellable lot (registered player) inside an auction.
type AuctionPlayer struct {
	ID                               string
	AuctionID                        string
	TournamentPlayerRegistrationID   string
	Status                           AuctionPlayerStatus
	BasePrice                        int64 // smallest currency unit; often aligned with event BaseBid
	FinalPrice                       int64 // set when sold; 0 if not sold
	SoldToTeamRegistrationID         string // TournamentTeamRegistration id; empty if not sold
	LotNumber                        int    // display order within auction
	IsActive                         bool   // only one lot might be "on the block" at a time
	CreatedAt                        int64 // unix milliseconds
}
