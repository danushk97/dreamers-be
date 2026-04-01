package auction

import "context"

// TournamentRepository supports CRUD/backfill for tournaments.
type TournamentRepository interface {
	Create(ctx context.Context, t *Tournament) error
	GetByID(ctx context.Context, id string) (*Tournament, error)
}

// EventRepository supports CRUD/backfill for events and their rules.
type EventRepository interface {
	Create(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
}

// TeamRepository manages tournament team registrations.
type TeamRepository interface {
	Create(ctx context.Context, team *TournamentTeamRegistration) error
	GetByID(ctx context.Context, id string) (*TournamentTeamRegistration, error)
}

// RegistrationRepository manages tournament player registrations and assignment to teams.
type RegistrationRepository interface {
	GetByID(ctx context.Context, id string) (*TournamentPlayerRegistration, error)
	// ListEligible lists registrations that match a runtime category filter.
	// Implementations should usually return only unassigned players (TeamID empty) for auctions.
	ListEligible(ctx context.Context, tournamentID, eventID string, f RegistrationFilter) ([]*TournamentPlayerRegistration, error)
	AssignToTeam(ctx context.Context, registrationID string, teamRegistrationID string) error
	UnassignTeam(ctx context.Context, registrationID string) error
}

// AuctionRepository manages auctions.
type AuctionRepository interface {
	Create(ctx context.Context, a *Auction) error
	GetByID(ctx context.Context, id string) (*Auction, error)
}

// AuctionPlayerRepository manages auction lots.
type AuctionPlayerRepository interface {
	Create(ctx context.Context, ap *AuctionPlayer) error
	GetByID(ctx context.Context, id string) (*AuctionPlayer, error)
	ListByAuction(ctx context.Context, auctionID string) ([]*AuctionPlayer, error)
	UpdateStatus(ctx context.Context, id string, status AuctionPlayerStatus, isActive bool) error
	MarkSold(ctx context.Context, id string, finalPrice int64, soldToTeamRegistrationID string) error
	ClearSale(ctx context.Context, id string) error
}

// BidRepository records bids and provides current-highest queries.
type BidRepository interface {
	Create(ctx context.Context, b *Bid) error
	ListByAuctionPlayer(ctx context.Context, auctionPlayerID string) ([]*Bid, error)
	GetHighestBid(ctx context.Context, auctionPlayerID string) (*Bid, error)
}

// WalletRepository provides wallet balance and ledger writes.
type WalletRepository interface {
	GetByTournamentEventTeam(ctx context.Context, tournamentID, eventID, teamRegistrationID string) (*Wallet, error)
	UpdateBalance(ctx context.Context, walletID string, newBalance int64, updatedAtMs int64) error
	CreateTransaction(ctx context.Context, tx *WalletTransaction) error
}

