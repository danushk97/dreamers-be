package auctionservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

type BidService struct {
	auctions         auction.AuctionRepository
	lots             auction.AuctionPlayerRepository
	tournamentEvents auction.TournamentEventRepository
	teams            auction.TeamRepository
	bids             auction.BidRepository
	wallets          auction.WalletRepository
	nowMs            func() int64
}

func NewBidService(d Deps) *BidService {
	return &BidService{
		auctions:         d.AuctionRepo,
		lots:             d.AuctionPlayerRepo,
		tournamentEvents: d.TournamentEventRepo,
		teams:            d.TeamRepo,
		bids:             d.BidRepo,
		wallets:          d.WalletRepo,
		nowMs:            d.now(),
	}
}

type PlaceBidInput struct {
	AuctionPlayerID    string
	TeamRegistrationID string
	Amount             int64
}

type RevertBidInput struct {
	AuctionPlayerID string
}

// PlaceBid records a bid with wallet + roster constraints.
// Debit happens only when the player is sold; bid placement checks "feasibility if this bid wins".
// On success, hints reflect limits for the next raise (balance unchanged until sale).
func (s *BidService) PlaceBid(ctx context.Context, in PlaceBidInput) (*auction.Bid, BidPlacementHints, error) {
	if in.AuctionPlayerID == "" || in.TeamRegistrationID == "" {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("auction_player_id and team_registration_id are required")}
	}
	if in.Amount <= 0 {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("amount must be > 0")}
	}

	lot, err := s.lots.GetByID(ctx, in.AuctionPlayerID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get lot: %w", err)
	}
	if lot == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("lot not found")}
	}
	if lot.Status == auction.AuctionPlayerSold {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("lot already sold")}
	}
	if lot.Status == auction.AuctionPlayerSkipped {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("lot is skipped")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return nil, BidPlacementHints{}, err
	}
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil || te.Attrs.TeamEventRules == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("tournament event rules not found")}
	}
	teamRules := te.Attrs.TeamEventRules

	minFloor := max64(lot.BasePrice, a.Rules.MinBidAmount)

	team, err := s.teams.GetByID(ctx, in.TeamRegistrationID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get team: %w", err)
	}
	if team == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("team not found")}
	}
	if team.TournamentID != a.TournamentID || team.TournamentEventID != a.TournamentEventID {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("team not in this tournament event")}
	}

	w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, in.TeamRegistrationID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get wallet: %w", err)
	}
	if w == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("wallet not found")}
	}
	if w.MaxBidAmount > 0 && !a.Rules.TeamWalletMaxBidOK(w.MaxBidAmount) {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("team max bid amount must be between auction MinBidAmount and MaxBidAmount")}
	}

	soldCount, err := countSoldToTeam(ctx, s.lots, a.ID, in.TeamRegistrationID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("count sold: %w", err)
	}
	if teamRules.MaxPlayersPerTeam > 0 && soldCount >= teamRules.MaxPlayersPerTeam {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("team already reached max players")}
	}

	derived := auction.DerivedMaxBidAmount(w.Balance, soldCount, teamRules.MinPlayersPerTeam, a.Rules.MinBidAmount, a.Rules.MaxBidAmount, w.MaxBidAmount)
	remainingPlayers := 0
	if teamRules.MaxPlayersPerTeam > 0 {
		remainingPlayers = teamRules.MaxPlayersPerTeam - soldCount
		if remainingPlayers < 0 {
			remainingPlayers = 0
		}
	}
	hints := BidPlacementHints{
		DerivedMaxBidAmount: derived,
		MinBidAmount:        minFloor,
		MaxBidAmount:        w.MaxBidAmount,
		RemainingPlayersToPick: remainingPlayers,
	}

	if in.Amount < minFloor {
		return nil, hints, &ValidationError{Err: fmt.Errorf("bid must be >= minimum bid"), Hints: &hints}
	}
	if derived < minFloor {
		return nil, hints, &ValidationError{Err: fmt.Errorf("cannot bid: wallet cannot cover minimum roster at minimum bid after this purchase"), Hints: &hints}
	}
	if in.Amount > derived {
		return nil, hints, &ValidationError{Err: fmt.Errorf("bid exceeds maximum allowed for current balance and roster reserve"), Hints: &hints}
	}

	b := &auction.Bid{
		ID:                 uuid.New().String(),
		AuctionPlayerID:    in.AuctionPlayerID,
		TeamRegistrationID: in.TeamRegistrationID,
		Amount:             in.Amount,
		RecordedAt:         s.nowMs(),
	}
	if err := s.bids.Create(ctx, b); err != nil {
		return nil, hints, fmt.Errorf("create bid: %w", err)
	}
	return b, hints, nil
}

// RevertLatestBid removes the latest bid recorded for a lot.
func (s *BidService) RevertLatestBid(ctx context.Context, in RevertBidInput) (*auction.Bid, BidPlacementHints, error) {
	if in.AuctionPlayerID == "" {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("auction_player_id is required")}
	}

	lot, err := s.lots.GetByID(ctx, in.AuctionPlayerID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get lot: %w", err)
	}
	if lot == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("lot not found")}
	}
	if lot.Status == auction.AuctionPlayerSold {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("cannot revert bid after lot is sold")}
	}
	if lot.Status == auction.AuctionPlayerSkipped {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("lot is skipped")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return nil, BidPlacementHints{}, err
	}

	reverted, err := s.bids.DeleteLatestByAuctionPlayer(ctx, in.AuctionPlayerID)
	if err != nil {
		return nil, BidPlacementHints{}, fmt.Errorf("delete latest bid: %w", err)
	}
	if reverted == nil {
		return nil, BidPlacementHints{}, &ValidationError{Err: fmt.Errorf("no bids found to revert")}
	}

	w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, reverted.TeamRegistrationID)
	if err != nil {
		return reverted, BidPlacementHints{}, fmt.Errorf("get wallet: %w", err)
	}
	if w == nil {
		return reverted, BidPlacementHints{}, nil
	}
	derived, derr := s.DerivedMaxBidForTeam(ctx, a.ID, reverted.TeamRegistrationID, w)
	if derr != nil {
		return reverted, BidPlacementHints{}, nil
	}
	hints := BidPlacementHints{
		DerivedMaxBidAmount: derived,
		MinBidAmount:        max64(lot.BasePrice, a.Rules.MinBidAmount),
		MaxBidAmount:        w.MaxBidAmount,
	}
	return reverted, hints, nil
}
