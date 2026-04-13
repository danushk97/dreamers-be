package auctionservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

type LotService struct {
	auctions         auction.AuctionRepository
	lots             auction.AuctionPlayerRepository
	tournamentEvents auction.TournamentEventRepository
	regs             auction.RegistrationRepository
	nowMs            func() int64
}

func NewLotService(d Deps) *LotService {
	return &LotService{
		auctions:         d.AuctionRepo,
		lots:             d.AuctionPlayerRepo,
		tournamentEvents: d.TournamentEventRepo,
		regs:             d.RegistrationRepo,
		nowMs:            d.now(),
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
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil || te.Attrs.TeamEventRules == nil || !te.Attrs.TeamEventRules.IsAuction {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event is not configured for auction")}
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
		base = a.Rules.MinBidAmount
	}
	ap := &auction.AuctionPlayer{
		ID:                             uuid.New().String(),
		AuctionID:                      in.AuctionID,
		TournamentPlayerRegistrationID: in.TournamentPlayerRegistrationID,
		Status:                         auction.AuctionPlayerPending,
		BasePrice:                      base,
		LotNumber:                      in.LotNumber,
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
	return s.regs.ListEligible(ctx, a.TournamentID, a.TournamentEventID, f)
}

type CreateLotsBulkInput struct {
	AuctionID       string
	RegistrationIDs []string
	StartLotNumber  int
	BasePrice       int64 // 0 = auction rules MinBidAmount
}

// CreateLotsByQueryInput creates lots by fetching eligible registrations server-side.
// Client does not need to send registration IDs.
type CreateLotsByQueryInput struct {
	AuctionID          string
	TournamentID      string
	TournamentEventID string
	Filter             auction.RegistrationFilter
	StartLotNumber    int
	BasePrice         int64 // 0 = auction rules MinBidAmount
	Limit              int   // 0 = no limit
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
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil || te.Attrs.TeamEventRules == nil || !te.Attrs.TeamEventRules.IsAuction {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event is not configured for auction")}
	}
	base := in.BasePrice
	if base <= 0 {
		base = a.Rules.MinBidAmount
	}
	if in.StartLotNumber <= 0 {
		in.StartLotNumber = 1
	}

	// Reauction support: if an auction_player row already exists for the same
	// registration and is in `unsold` state, switch it back to `active`.
	// This avoids creating duplicates for reauction flows.
	existingLots, err := s.lots.ListByAuction(ctx, in.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("list existing lots: %w", err)
	}
	existingByRegID := make(map[string]*auction.AuctionPlayer, len(existingLots))
	for _, l := range existingLots {
		if l == nil {
			continue
		}
		// If there are duplicates from earlier runs, prefer unsold so reauction can activate them.
		if prev, ok := existingByRegID[l.TournamentPlayerRegistrationID]; !ok || prev.Status != auction.AuctionPlayerUnsold && l.Status == auction.AuctionPlayerUnsold {
			existingByRegID[l.TournamentPlayerRegistrationID] = l
		}
	}

	out := make([]*auction.AuctionPlayer, 0, len(in.RegistrationIDs))
	lotNum := in.StartLotNumber
	for _, regID := range in.RegistrationIDs {
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

		// If the registration already has a lot for this auction, reuse it.
		if existing := existingByRegID[regID]; existing != nil {
			if existing.Status == auction.AuctionPlayerUnsold {
				if err := s.lots.UpdateStatus(ctx, existing.ID, auction.AuctionPlayerActive, true); err != nil {
					return nil, fmt.Errorf("activate unsold lot: %w", err)
				}
				existing.Status = auction.AuctionPlayerActive
				existing.IsActive = true
			}
			out = append(out, existing)
			continue
		}

		ap := &auction.AuctionPlayer{
			ID:                             uuid.New().String(),
			AuctionID:                      in.AuctionID,
			TournamentPlayerRegistrationID: regID,
			Status:                         auction.AuctionPlayerPending,
			BasePrice:                      base,
			LotNumber:                      lotNum,
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

// CreateLotsByQuery creates lots by selecting unassigned eligible registrations server-side.
func (s *LotService) CreateLotsByQuery(ctx context.Context, in CreateLotsByQueryInput) ([]*auction.AuctionPlayer, error) {
	if in.AuctionID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}

	a, err := s.auctions.GetByID(ctx, in.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}

	// Allow client to pass tournament/event, but validate consistency with auction if they do.
	tournamentID := a.TournamentID
	tournamentEventID := a.TournamentEventID
	if in.TournamentID != "" && in.TournamentID != tournamentID {
		return nil, &ValidationError{Err: fmt.Errorf("tournament_id does not match auction")}
	}
	if in.TournamentEventID != "" && in.TournamentEventID != tournamentEventID {
		return nil, &ValidationError{Err: fmt.Errorf("tournament_event_id does not match auction")}
	}

	te, err := s.tournamentEvents.GetByID(ctx, tournamentEventID)
	if err != nil {
		return nil, fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil || te.Attrs.TeamEventRules == nil || !te.Attrs.TeamEventRules.IsAuction {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event is not configured for auction")}
	}

	base := in.BasePrice
	if base <= 0 {
		base = a.Rules.MinBidAmount
	}
	startLot := in.StartLotNumber
	if startLot <= 0 {
		startLot = 1
	}

	regs, err := s.regs.ListEligible(ctx, tournamentID, tournamentEventID, in.Filter)
	if err != nil {
		return nil, fmt.Errorf("list eligible registrations: %w", err)
	}
	if in.Limit > 0 && len(regs) > in.Limit {
		regs = regs[:in.Limit]
	}

	// Same reauction logic as bulk: reuse existing unsold lots and activate them.
	existingLots, err := s.lots.ListByAuction(ctx, in.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("list existing lots: %w", err)
	}
	existingByRegID := make(map[string]*auction.AuctionPlayer, len(existingLots))
	for _, l := range existingLots {
		if l == nil {
			continue
		}
		if prev, ok := existingByRegID[l.TournamentPlayerRegistrationID]; !ok || prev.Status != auction.AuctionPlayerUnsold && l.Status == auction.AuctionPlayerUnsold {
			existingByRegID[l.TournamentPlayerRegistrationID] = l
		}
	}

	out := make([]*auction.AuctionPlayer, 0, len(regs))
	lotNum := startLot
	for _, reg := range regs {
		if reg == nil {
			continue
		}
		// Extra safety: service invariant is unassigned registrations.
		if reg.TeamID != "" {
			continue
		}

		// Reuse existing lots for this auction/registration if present.
		if existing := existingByRegID[reg.ID]; existing != nil {
			if existing.Status == auction.AuctionPlayerUnsold {
				if err := s.lots.UpdateStatus(ctx, existing.ID, auction.AuctionPlayerActive, true); err != nil {
					return nil, fmt.Errorf("activate unsold lot: %w", err)
				}
				existing.Status = auction.AuctionPlayerActive
				existing.IsActive = true
			}
			out = append(out, existing)
			continue
		}

		ap := &auction.AuctionPlayer{
			ID:                             uuid.New().String(),
			AuctionID:                      in.AuctionID,
			TournamentPlayerRegistrationID: reg.ID,
			Status:                         auction.AuctionPlayerPending,
			BasePrice:                      base,
			LotNumber:                      lotNum,
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

// GetLotByRegistrationSerial returns the lot for this auction whose player registration has the
// given serial_number (scoped to the auction's tournament event). The lot may be in any status
// (pending, active, sold, unsold, skipped).
func (s *LotService) GetLotByRegistrationSerial(ctx context.Context, auctionID string, serialNumber int) (*auction.AuctionPlayer, *auction.TournamentPlayerRegistration, error) {
	if auctionID == "" || serialNumber < 1 {
		return nil, nil, &ValidationError{Err: fmt.Errorf("auction_id and a positive serial_number are required")}
	}
	a, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return nil, nil, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	ap, err := s.lots.GetByAuctionAndRegistrationSerial(ctx, auctionID, serialNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("get lot by registration serial: %w", err)
	}
	if ap == nil {
		return nil, nil, &ValidationError{Err: fmt.Errorf("no lot for this registration serial in this auction; create the lot first")}
	}
	reg, err := s.regs.GetByID(ctx, ap.TournamentPlayerRegistrationID)
	if err != nil {
		return nil, nil, fmt.Errorf("get registration: %w", err)
	}
	if reg == nil {
		return nil, nil, &ValidationError{Err: fmt.Errorf("registration not found for lot")}
	}
	return ap, reg, nil
}

// SetDisplayAuctionPlayer records which lot the console and relay should show. Empty auctionPlayerID clears (fallback heuristic).
func (s *LotService) SetDisplayAuctionPlayer(ctx context.Context, auctionID, auctionPlayerID string) (*auction.Auction, error) {
	if auctionID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	a, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	lotID := strings.TrimSpace(auctionPlayerID)
	if lotID == "" {
		if err := s.auctions.UpdateDisplayAuctionPlayer(ctx, auctionID, ""); err != nil {
			return nil, err
		}
		return s.auctions.GetByID(ctx, auctionID)
	}
	lot, err := s.lots.GetByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	if lot == nil || lot.AuctionID != auctionID {
		return nil, &ValidationError{Err: fmt.Errorf("auction player not in this auction")}
	}
	if err := s.auctions.UpdateDisplayAuctionPlayer(ctx, auctionID, lotID); err != nil {
		return nil, err
	}
	return s.auctions.GetByID(ctx, auctionID)
}

// RegistrationDisplayMeta returns serial number and player id for a tournament player registration (for relay / UI).
func (s *LotService) RegistrationDisplayMeta(ctx context.Context, registrationID string) (serialNumber int, playerID string, err error) {
	if registrationID == "" {
		return 0, "", nil
	}
	r, err := s.regs.GetByID(ctx, registrationID)
	if err != nil {
		return 0, "", err
	}
	if r == nil {
		return 0, "", nil
	}
	return r.SerialNumber, r.PlayerID, nil
}
