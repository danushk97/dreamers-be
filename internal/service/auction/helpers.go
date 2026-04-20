package auctionservice

import (
	"context"

	"github.com/dreamers-be/internal/domain/auction"
)

func countSoldToTeam(ctx context.Context, lots auction.AuctionPlayerRepository, auctionID string, teamRegistrationID string) (int, error) {
	items, err := lots.ListByAuction(ctx, auctionID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, l := range items {
		if l == nil {
			continue
		}
		if l.Status == auction.AuctionPlayerSold && l.SoldToTeamRegistrationID == teamRegistrationID {
			n++
		}
	}
	return n, nil
}

func countRetainedToTeam(ctx context.Context, lots auction.AuctionPlayerRepository, auctionID string, teamRegistrationID string) (int, error) {
	items, err := lots.ListByAuction(ctx, auctionID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, l := range items {
		if l == nil {
			continue
		}
		if l.Status == auction.AuctionPlayerSold &&
			l.SoldToTeamRegistrationID == teamRegistrationID &&
			l.Notes.IsRetained {
			n++
		}
	}
	return n, nil
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
