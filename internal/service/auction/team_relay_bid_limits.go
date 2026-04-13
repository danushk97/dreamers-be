package auctionservice

import (
	"context"
	"fmt"

	"github.com/dreamers-be/internal/domain/auction"
)

// RelayTeamBidLimit is one team's current bid ceiling for relay / spectator UI (paise).
type RelayTeamBidLimit struct {
	TeamRegistrationID  string `json:"teamRegistrationId"`
	DerivedMaxBidAmount int64  `json:"derivedMaxBidAmount"`
	MinBidAmount        int64  `json:"minBidAmount"`
	MaxBidAmount        int64  `json:"maxBidAmount"` // configured wallet cap; 0 = none
	RemainingPlayersToPick int `json:"remainingPlayersToPick"`
}

func soldCountForTeamOnLots(lots []*auction.AuctionPlayer, auctionID, teamRegID string) int {
	n := 0
	for _, l := range lots {
		if l == nil || l.AuctionID != auctionID {
			continue
		}
		if l.Status == auction.AuctionPlayerSold && l.SoldToTeamRegistrationID == teamRegID {
			n++
		}
	}
	return n
}

// TeamRelayBidLimits returns per-team derived max bid and caps for an auction (all teams in the event).
func (s *BidService) TeamRelayBidLimits(ctx context.Context, a *auction.Auction, lots []*auction.AuctionPlayer) ([]RelayTeamBidLimit, error) {
	if a == nil {
		return nil, fmt.Errorf("auction is nil")
	}
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("get tournament event: %w", err)
	}
	minPlayers := 0
	maxPlayers := 0
	if te != nil && te.Attrs.TeamEventRules != nil {
		minPlayers = te.Attrs.TeamEventRules.MinPlayersPerTeam
		maxPlayers = te.Attrs.TeamEventRules.MaxPlayersPerTeam
	}
	teamList, err := s.teams.ListByTournamentEvent(ctx, a.TournamentID, a.TournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	out := make([]RelayTeamBidLimit, 0, len(teamList))
	for _, t := range teamList {
		if t == nil {
			continue
		}
		w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, t.ID)
		if err != nil {
			return nil, fmt.Errorf("get wallet: %w", err)
		}
		if w == nil {
			out = append(out, RelayTeamBidLimit{
				TeamRegistrationID: t.ID,
				MinBidAmount:       a.Rules.MinBidAmount,
				RemainingPlayersToPick: maxPlayers,
			})
			continue
		}
		sold := soldCountForTeamOnLots(lots, a.ID, t.ID)
		remainingPlayers := 0
		if maxPlayers > 0 {
			remainingPlayers = maxPlayers - sold
			if remainingPlayers < 0 {
				remainingPlayers = 0
			}
		}
		derived := auction.DerivedMaxBidAmount(w.Balance, sold, minPlayers, a.Rules.MinBidAmount, a.Rules.MaxBidAmount, w.MaxBidAmount)
		out = append(out, RelayTeamBidLimit{
			TeamRegistrationID:  t.ID,
			DerivedMaxBidAmount: derived,
			MinBidAmount:        a.Rules.MinBidAmount,
			MaxBidAmount:        w.MaxBidAmount,
			RemainingPlayersToPick: remainingPlayers,
		})
	}
	return out, nil
}
