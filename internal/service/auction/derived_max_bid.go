package auctionservice

import (
	"context"
	"fmt"

	"github.com/dreamers-be/internal/domain/auction"
)

// DerivedMaxBidForTeam computes the current single-bid ceiling for a wallet in an auction
// (balance − roster reserve, then auction + per-team caps). See auction.DerivedMaxBidAmount.
func (s *BidService) DerivedMaxBidForTeam(ctx context.Context, auctionID, teamRegistrationID string, w *auction.Wallet) (int64, error) {
	if auctionID == "" || teamRegistrationID == "" || w == nil {
		return 0, fmt.Errorf("auction_id, team_registration_id, and wallet are required")
	}
	a, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return 0, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return 0, fmt.Errorf("auction not found")
	}
	if w.TournamentID != a.TournamentID || w.TournamentEventID != a.TournamentEventID {
		return 0, fmt.Errorf("wallet does not match auction tournament event")
	}
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return 0, fmt.Errorf("get tournament event: %w", err)
	}
	minPlayers := 0
	if te != nil && te.Attrs.TeamEventRules != nil {
		minPlayers = te.Attrs.TeamEventRules.MinPlayersPerTeam
	}
	soldCount, err := countSoldToTeam(ctx, s.lots, auctionID, teamRegistrationID)
	if err != nil {
		return 0, fmt.Errorf("count sold: %w", err)
	}
	return auction.DerivedMaxBidAmount(w.Balance, soldCount, minPlayers, a.Rules.MinBidAmount, a.Rules.MaxBidAmount, w.MaxBidAmount), nil
}
