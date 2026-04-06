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

// PlaceBid records a bid with wallet + roster constraints.
// Debit happens only when the player is sold; bid placement checks "feasibility if this bid wins".
func (s *BidService) PlaceBid(ctx context.Context, in PlaceBidInput) (*auction.Bid, error) {
	if in.AuctionPlayerID == "" || in.TeamRegistrationID == "" {
		return nil, &ValidationError{Err: fmt.Errorf("auction_player_id and team_registration_id are required")}
	}
	if in.Amount <= 0 {
		return nil, &ValidationError{Err: fmt.Errorf("amount must be > 0")}
	}

	lot, err := s.lots.GetByID(ctx, in.AuctionPlayerID)
	if err != nil {
		return nil, fmt.Errorf("get lot: %w", err)
	}
	if lot == nil {
		return nil, &ValidationError{Err: fmt.Errorf("lot not found")}
	}
	if lot.Status == auction.AuctionPlayerSold {
		return nil, &ValidationError{Err: fmt.Errorf("lot already sold")}
	}
	if lot.Status == auction.AuctionPlayerSkipped {
		return nil, &ValidationError{Err: fmt.Errorf("lot is skipped")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
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
	if te == nil || te.Attrs.TeamEventRules == nil {
		return nil, &ValidationError{Err: fmt.Errorf("tournament event rules not found")}
	}
	rules := te.Attrs.TeamEventRules

	if in.Amount < max64(lot.BasePrice, rules.BaseBid) {
		return nil, &ValidationError{Err: fmt.Errorf("bid must be >= base bid")}
	}
	if rules.MaxBid > 0 && in.Amount > rules.MaxBid {
		return nil, &ValidationError{Err: fmt.Errorf("bid exceeds max bid")}
	}

	team, err := s.teams.GetByID(ctx, in.TeamRegistrationID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	if team == nil {
		return nil, &ValidationError{Err: fmt.Errorf("team not found")}
	}
	if team.TournamentID != a.TournamentID || team.TournamentEventID != a.TournamentEventID {
		return nil, &ValidationError{Err: fmt.Errorf("team not in this tournament event")}
	}

	w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, in.TeamRegistrationID)
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	if w == nil {
		return nil, &ValidationError{Err: fmt.Errorf("wallet not found")}
	}
	if w.Balance < in.Amount {
		return nil, &ValidationError{Err: fmt.Errorf("insufficient wallet balance")}
	}

	soldCount, err := countSoldToTeam(ctx, s.lots, a.ID, in.TeamRegistrationID)
	if err != nil {
		return nil, fmt.Errorf("count sold: %w", err)
	}
	if rules.MaxPlayersPerTeam > 0 && soldCount >= rules.MaxPlayersPerTeam {
		return nil, &ValidationError{Err: fmt.Errorf("team already reached max players")}
	}

	remainingMinAfter := 0
	if rules.MinPlayersPerTeam > 0 {
		remainingMinAfter = rules.MinPlayersPerTeam - (soldCount + 1)
		if remainingMinAfter < 0 {
			remainingMinAfter = 0
		}
	}
	minReserve := int64(remainingMinAfter) * rules.BaseBid
	if w.Balance-in.Amount < minReserve {
		return nil, &ValidationError{Err: fmt.Errorf("bid would prevent meeting minimum roster with available wallet")}
	}

	b := &auction.Bid{
		ID:                 uuid.New().String(),
		AuctionPlayerID:    in.AuctionPlayerID,
		TeamRegistrationID: in.TeamRegistrationID,
		Amount:             in.Amount,
		RecordedAt:         s.nowMs(),
	}
	if err := s.bids.Create(ctx, b); err != nil {
		return nil, fmt.Errorf("create bid: %w", err)
	}
	return b, nil
}

