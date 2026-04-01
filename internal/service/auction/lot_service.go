package auctionservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

type LotService struct {
	auctions auction.AuctionRepository
	lots     auction.AuctionPlayerRepository
	events   auction.EventRepository
	regs     auction.RegistrationRepository
	nowMs    func() int64
}

func NewLotService(d Deps) *LotService {
	return &LotService{
		auctions: d.AuctionRepo,
		lots:     d.AuctionPlayerRepo,
		events:   d.EventRepo,
		regs:     d.RegistrationRepo,
		nowMs:    d.now(),
	}
}

type CreateLotInput struct {
	AuctionID                      string
	TournamentPlayerRegistrationID string
	LotNumber                      int
	BasePrice                      int64
}

// CreateLot creates a sellable lot from a tournament registration.
// Category filtering is performed by ListEligibleRegistrations and the auctioneer's selection,
// not stored on the Auction itself.
func (s *LotService) CreateLot(ctx context.Context, in CreateLotInput) (*auction.AuctionPlayer, error) {
	if in.AuctionID == "" || in.TournamentPlayerRegistrationID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id and tournament_player_registration_id are required")}
	}
	a, err := s.auctions.GetByID(ctx, in.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	ev, err := s.events.GetByID(ctx, a.EventID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if ev == nil || ev.Attrs.TeamEventRules == nil {
		return nil, &ValidationError{Err: fmt.Errorf("event rules not found")}
	}

	reg, err := s.regs.GetByID(ctx, in.TournamentPlayerRegistrationID)
	if err != nil {
		return nil, fmt.Errorf("get registration: %w", err)
	}
	if reg == nil {
		return nil, &ValidationError{Err: fmt.Errorf("registration not found")}
	}
	if reg.TeamID != "" {
		return nil, &ValidationError{Err: fmt.Errorf("player already assigned to a team")}
	}

	base := in.BasePrice
	if base <= 0 {
		base = ev.Attrs.TeamEventRules.BaseBid
	}
	ap := &auction.AuctionPlayer{
		ID:                             uuid.New().String(),
		AuctionID:                      in.AuctionID,
		TournamentPlayerRegistrationID: in.TournamentPlayerRegistrationID,
		Status:                         auction.AuctionPlayerPending,
		BasePrice:                      base,
		LotNumber:                      in.LotNumber,
		OrderIndex:                     in.LotNumber,
		IsActive:                       false,
		CreatedAt:                      s.nowMs(),
	}
	if err := s.lots.Create(ctx, ap); err != nil {
		return nil, fmt.Errorf("create lot: %w", err)
	}
	return ap, nil
}

// ListEligibleRegistrations returns tournament registrations eligible for auction based on a runtime filter.
// Implementations should generally return only unassigned players.
func (s *LotService) ListEligibleRegistrations(ctx context.Context, auctionID string, f auction.RegistrationFilter) ([]*auction.TournamentPlayerRegistration, error) {
	if auctionID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	a, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return nil, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	return s.regs.ListEligible(ctx, a.TournamentID, a.EventID, f)
}

type CreateLotsBulkInput struct {
	AuctionID        string
	RegistrationIDs  []string
	StartLotNumber   int
	BasePrice        int64 // 0 = event base bid
}

// CreateLotsBulk creates lots in a single call for the auctioneer-selected category/round.
func (s *LotService) CreateLotsBulk(ctx context.Context, in CreateLotsBulkInput) ([]*auction.AuctionPlayer, error) {
	if in.AuctionID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	if len(in.RegistrationIDs) == 0 {
		return nil, &ValidationError{Err: fmt.Errorf("registration_ids is required")}
	}
	a, err := s.auctions.GetByID(ctx, in.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	ev, err := s.events.GetByID(ctx, a.EventID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if ev == nil || ev.Attrs.TeamEventRules == nil {
		return nil, &ValidationError{Err: fmt.Errorf("event rules not found")}
	}
	base := in.BasePrice
	if base <= 0 {
		base = ev.Attrs.TeamEventRules.BaseBid
	}
	if in.StartLotNumber <= 0 {
		in.StartLotNumber = 1
	}

	out := make([]*auction.AuctionPlayer, 0, len(in.RegistrationIDs))
	lotNum := in.StartLotNumber
	for _, regID := range in.RegistrationIDs {
		regID = regID
		if regID == "" {
			return nil, &ValidationError{Err: fmt.Errorf("registration_ids contains empty value")}
		}
		reg, err := s.regs.GetByID(ctx, regID)
		if err != nil {
			return nil, fmt.Errorf("get registration: %w", err)
		}
		if reg == nil {
			return nil, &ValidationError{Err: fmt.Errorf("registration not found")}
		}
		if reg.TeamID != "" {
			return nil, &ValidationError{Err: fmt.Errorf("player already assigned to a team")}
		}

		ap := &auction.AuctionPlayer{
			ID:                             uuid.New().String(),
			AuctionID:                      in.AuctionID,
			TournamentPlayerRegistrationID: regID,
			Status:                         auction.AuctionPlayerPending,
			BasePrice:                      base,
			LotNumber:                      lotNum,
			OrderIndex:                     lotNum,
			IsActive:                       false,
			CreatedAt:                      s.nowMs(),
		}
		if err := s.lots.Create(ctx, ap); err != nil {
			return nil, fmt.Errorf("create lot: %w", err)
		}
		out = append(out, ap)
		lotNum++
	}

	return out, nil
}

