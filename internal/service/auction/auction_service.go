package auctionservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

type AuctionService struct {
	auctions auction.AuctionRepository
	events   auction.EventRepository
	nowMs    func() int64
}

func NewAuctionService(d Deps) *AuctionService {
	return &AuctionService{
		auctions: d.AuctionRepo,
		events:   d.EventRepo,
		nowMs:    d.now(),
	}
}

// CreateAuction creates an auction session for a tournament/event.
// filterPresets are saved on the auction (typically persisted as JSON) for UI convenience.
func (s *AuctionService) CreateAuction(ctx context.Context, tournamentID, eventID string, mode auction.AuctionMode, filterPresets []auction.FilterPreset) (*auction.Auction, error) {
	if tournamentID == "" || eventID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("tournament_id and event_id are required")}
	}
	ev, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if ev == nil {
		return nil, &ValidationError{Err: fmt.Errorf("event not found")}
	}
	if ev.Attrs.TeamEventRules == nil || !ev.Attrs.TeamEventRules.IsAuction {
		return nil, &ValidationError{Err: fmt.Errorf("event is not configured for auction")}
	}

	a := &auction.Auction{
		ID:           uuid.New().String(),
		TournamentID: tournamentID,
		EventID:      eventID,
		Mode:         mode,
		FilterPresets: filterPresets,
		CreatedAt:    s.nowMs(),
	}
	if err := s.auctions.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create auction: %w", err)
	}
	return a, nil
}

