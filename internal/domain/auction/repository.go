package auction

import "context"

// TournamentRepository supports CRUD/backfill for tournaments.
type TournamentRepository interface {
	Create(ctx context.Context, t *Tournament) error
	GetByID(ctx context.Context, id string) (*Tournament, error)
}

// TournamentEventRepository supports CRUD/backfill for tournament-scoped events.
type TournamentEventRepository interface {
	Create(ctx context.Context, te *TournamentEvent) error
	GetByID(ctx context.Context, id string) (*TournamentEvent, error)
}

// TeamRepository manages tournament team registrations.
type TeamRepository interface {
	Create(ctx context.Context, team *TournamentTeamRegistration) error
	GetByID(ctx context.Context, id string) (*TournamentTeamRegistration, error)
	// ListByTournamentEvent returns all teams registered for the given tournament event.
	ListByTournamentEvent(ctx context.Context, tournamentID, tournamentEventID string) ([]*TournamentTeamRegistration, error)
}

// RegistrationRepository manages tournament player registrations and assignment to teams.
type RegistrationRepository interface {
	GetByID(ctx context.Context, id string) (*TournamentPlayerRegistration, error)
	// ListEligible lists registrations that match a runtime category filter.
	// Implementations should usually return only unassigned players (TeamID empty) for auctions.
	ListEligible(ctx context.Context, tournamentID, tournamentEventID string, f RegistrationFilter) ([]*TournamentPlayerRegistration, error)
	AssignToTeam(ctx context.Context, registrationID string, teamRegistrationID string) error
	UnassignTeam(ctx context.Context, registrationID string) error
}

// AuctionRepository manages auctions.
type AuctionRepository interface {
	Create(ctx context.Context, a *Auction) error
	GetByID(ctx context.Context, id string) (*Auction, error)
	UpdateRunMode(ctx context.Context, auctionID string, runMode AuctionRunMode) error
}

// AuctionPlayerRepository manages auction lots.
type AuctionPlayerRepository interface {
	Create(ctx context.Context, ap *AuctionPlayer) error
	GetByID(ctx context.Context, id string) (*AuctionPlayer, error)
	// GetByAuctionAndRegistrationSerial finds the lot for this auction whose registration
	// has the given serial_number within the auction's tournament event.
	GetByAuctionAndRegistrationSerial(ctx context.Context, auctionID string, serialNumber int) (*AuctionPlayer, error)
	ListByAuction(ctx context.Context, auctionID string) ([]*AuctionPlayer, error)
	UpdateStatus(ctx context.Context, id string, status AuctionPlayerStatus, isActive bool) error
	MarkSold(ctx context.Context, id string, finalPrice int64, soldToTeamRegistrationID string) error
	ClearSale(ctx context.Context, id string) error
	// ResetAllLotsToPending sets every lot in the auction to pending with no sale/active state.
	ResetAllLotsToPending(ctx context.Context, auctionID string) error
}

// BidRepository records bids and provides current-highest queries.
type BidRepository interface {
	Create(ctx context.Context, b *Bid) error
	ListByAuctionPlayer(ctx context.Context, auctionPlayerID string) ([]*Bid, error)
	GetHighestBid(ctx context.Context, auctionPlayerID string) (*Bid, error)
	DeleteByAuction(ctx context.Context, auctionID string) error
}

// WalletRepository provides wallet balance and ledger writes.
type WalletRepository interface {
	GetByTournamentEventTeam(ctx context.Context, tournamentID, tournamentEventID, teamRegistrationID string) (*Wallet, error)
	UpdateBalance(ctx context.Context, walletID string, newBalance int64, updatedAtMs int64) error
	CreateTransaction(ctx context.Context, tx *WalletTransaction) error
}

