package auctionservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

type AuctionService struct {
	auctions         auction.AuctionRepository
	tournaments      auction.TournamentRepository
	tournamentEvents auction.TournamentEventRepository
	teams            auction.TeamRepository
	nowMs            func() int64
}

func NewAuctionService(d Deps) *AuctionService {
	return &AuctionService{
		auctions:         d.AuctionRepo,
		tournaments:      d.TournamentRepo,
		tournamentEvents: d.TournamentEventRepo,
		teams:            d.TeamRepo,
		nowMs:            d.now(),
	}
}

// CreateAuction creates an auction session for a tournament event.
// filterPresets are saved on the auction (typically persisted as JSON) for UI convenience.
// runMode is "test" (default, allows reset) or "live".
// rules sets min/max single-bid amounts (paisa) on the auction row.
func (s *AuctionService) CreateAuction(ctx context.Context, tournamentID, tournamentEventID string, mode auction.AuctionMode, runMode auction.AuctionRunMode, filterPresets []auction.FilterPreset, rules auction.AuctionRules) (*auction.Auction, error) {
	if tournamentID == "" || tournamentEventID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("tournament_id and tournament_event_id are required")}
	}
	if runMode != "" && runMode != auction.AuctionRunModeTest && runMode != auction.AuctionRunModeLive {
		return nil, &ValidationError{Err: fmt.Errorf("runMode must be \"test\" or \"live\"")}
	}
	if runMode == "" {
		runMode = auction.AuctionRunModeTest
	}
	te, err := s.tournamentEvents.GetByID(ctx, tournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event not found")}
	}
	if te.TournamentID != tournamentID {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event does not belong to tournament")}
	}
	if te.Attrs.TeamEventRules == nil || !te.Attrs.TeamEventRules.IsAuction {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event is not configured for auction")}
	}
	rules = auction.ApplyDefaultAuctionRules(rules)
	if err := rules.Validate(); err != nil {
		return nil, &ValidationError{Err: err}
	}

	a := &auction.Auction{
		ID:                uuid.New().String(),
		TournamentID:      tournamentID,
		TournamentEventID: tournamentEventID,
		Mode:              mode,
		RunMode:           runMode,
		Status:            auction.AuctionStatusCreated,
		FilterPresets:     filterPresets,
		Rules:             rules,
		CreatedAt:         s.nowMs(),
	}
	if err := s.auctions.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create auction: %w", err)
	}
	return a, nil
}

// UpdateAuctionRunMode sets run_mode to test or live.
func (s *AuctionService) UpdateAuctionRunMode(ctx context.Context, auctionID string, runMode auction.AuctionRunMode) (*auction.Auction, error) {
	if auctionID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	if runMode != auction.AuctionRunModeTest && runMode != auction.AuctionRunModeLive {
		return nil, &ValidationError{Err: fmt.Errorf("runMode must be \"test\" or \"live\"")}
	}
	existing, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := s.auctions.UpdateRunMode(ctx, auctionID, runMode); err != nil {
		if strings.Contains(err.Error(), "auction not found") {
			return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
		}
		return nil, fmt.Errorf("update run mode: %w", err)
	}
	return s.auctions.GetByID(ctx, auctionID)
}

// UpdateAuctionStatus sets lifecycle status: created, running, or completed.
func (s *AuctionService) UpdateAuctionStatus(ctx context.Context, auctionID string, status auction.AuctionStatus) (*auction.Auction, error) {
	if auctionID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	if _, err := auction.ParseAuctionStatus(string(status)); err != nil {
		return nil, &ValidationError{Err: err}
	}
	existing, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := s.auctions.UpdateStatus(ctx, auctionID, status); err != nil {
		if strings.Contains(err.Error(), "auction not found") {
			return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
		}
		return nil, fmt.Errorf("update status: %w", err)
	}
	return s.auctions.GetByID(ctx, auctionID)
}

// GetAuction loads an auction by id (includes persisted filter presets).
func (s *AuctionService) GetAuction(ctx context.Context, id string) (*auction.Auction, error) {
	if id == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	a, err := s.auctions.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	return a, nil
}

// GetTournament loads a tournament by id (logo + sponsors JSON).
func (s *AuctionService) GetTournament(ctx context.Context, id string) (*auction.Tournament, error) {
	if id == "" {
		return nil, &ValidationError{Err: fmt.Errorf("tournament_id is required")}
	}
	if s.tournaments == nil {
		return nil, fmt.Errorf("tournament repository not configured")
	}
	t, err := s.tournaments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, &ValidationError{Err: fmt.Errorf("tournament not found")}
	}
	return t, nil
}

// ListRegisteredTeams returns all team registrations for a tournament event.
// tournamentID must match the event's parent tournament.
func (s *AuctionService) ListRegisteredTeams(ctx context.Context, tournamentID, tournamentEventID string) ([]*auction.TournamentTeamRegistration, error) {
	if tournamentID == "" || tournamentEventID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("tournament_id and tournament_event_id are required")}
	}
	te, err := s.tournamentEvents.GetByID(ctx, tournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event not found")}
	}
	if te.TournamentID != tournamentID {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event does not belong to tournament")}
	}
	teams, err := s.teams.ListByTournamentEvent(ctx, tournamentID, tournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	if teams == nil {
		return []*auction.TournamentTeamRegistration{}, nil
	}
	return teams, nil
}

