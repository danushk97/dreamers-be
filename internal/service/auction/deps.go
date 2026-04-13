package auctionservice

import "github.com/dreamers-be/internal/domain/auction"

// Deps groups the repositories used by auction services.
// Individual services depend on the subset they actually need.
type Deps struct {
	AuctionRepo         auction.AuctionRepository
	AuctionPlayerRepo   auction.AuctionPlayerRepository
	TournamentRepo      auction.TournamentRepository
	TournamentEventRepo auction.TournamentEventRepository
	RegistrationRepo  auction.RegistrationRepository
	TeamRepo          auction.TeamRepository
	BidRepo           auction.BidRepository
	WalletRepo        auction.WalletRepository
	NowMs             func() int64
}

func (d Deps) now() func() int64 {
	if d.NowMs != nil {
		return d.NowMs
	}
	return func() int64 { return 0 }
}

