package auctionservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

type SettlementService struct {
	auctions auction.AuctionRepository
	lots     auction.AuctionPlayerRepository
	bids     auction.BidRepository
	regs     auction.RegistrationRepository
	wallets  auction.WalletRepository
	nowMs    func() int64
}

func NewSettlementService(d Deps) *SettlementService {
	return &SettlementService{
		auctions: d.AuctionRepo,
		lots:     d.AuctionPlayerRepo,
		bids:     d.BidRepo,
		regs:     d.RegistrationRepo,
		wallets:  d.WalletRepo,
		nowMs:    d.now(),
	}
}

type SellInput struct {
	AuctionPlayerID string
	ExpectedBidID   string // optional optimistic check; can be empty
}

// SellCurrentLot marks the lot as sold to the highest bid, assigns the registration to the team,
// and debits the team's wallet (ledger + balance update).
func (s *SettlementService) SellCurrentLot(ctx context.Context, in SellInput) error {
	if in.AuctionPlayerID == "" {
		return &ValidationError{Err: fmt.Errorf("auction_player_id is required")}
	}
	lot, err := s.lots.GetByID(ctx, in.AuctionPlayerID)
	if err != nil {
		return fmt.Errorf("get lot: %w", err)
	}
	if lot == nil {
		return &ValidationError{Err: fmt.Errorf("lot not found")}
	}
	if lot.Status == auction.AuctionPlayerSold {
		return &ValidationError{Err: fmt.Errorf("lot already sold")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}

	highest, err := s.bids.GetHighestBid(ctx, lot.ID)
	if err != nil {
		return fmt.Errorf("get highest bid: %w", err)
	}
	if highest == nil {
		return &ValidationError{Err: fmt.Errorf("no bids to sell")}
	}
	if in.ExpectedBidID != "" && highest.ID != in.ExpectedBidID {
		return &ValidationError{Err: fmt.Errorf("highest bid changed")}
	}

	w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, highest.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("get wallet: %w", err)
	}
	if w == nil {
		return &ValidationError{Err: fmt.Errorf("wallet not found")}
	}
	if w.Balance < highest.Amount {
		return &ValidationError{Err: fmt.Errorf("insufficient wallet balance to settle")}
	}

	if err := s.lots.MarkSold(ctx, lot.ID, highest.Amount, highest.TeamRegistrationID); err != nil {
		return fmt.Errorf("mark sold: %w", err)
	}
	if err := s.regs.AssignToTeam(ctx, lot.TournamentPlayerRegistrationID, highest.TeamRegistrationID); err != nil {
		return fmt.Errorf("assign to team: %w", err)
	}

	now := s.nowMs()
	tx := &auction.WalletTransaction{
		ID:          uuid.New().String(),
		WalletID:    w.ID,
		Amount:      highest.Amount,
		Type:        auction.WalletTxnDebit,
		ReferenceID: lot.ID,
		CreatedAt:   now,
	}
	if err := s.wallets.CreateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("wallet debit tx: %w", err)
	}
	if err := s.wallets.UpdateBalance(ctx, w.ID, w.Balance-highest.Amount, now); err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}
	return nil
}

// MarkUnsold moves an open lot to unsold (no wallet movement). If the lot is already sold,
// it performs the same reversal as RevertSale (credit wallet, unassign player, clear sale).
func (s *SettlementService) MarkUnsold(ctx context.Context, auctionPlayerID string) error {
	if auctionPlayerID == "" {
		return &ValidationError{Err: fmt.Errorf("auction_player_id is required")}
	}
	lot, err := s.lots.GetByID(ctx, auctionPlayerID)
	if err != nil {
		return fmt.Errorf("get lot: %w", err)
	}
	if lot == nil {
		return &ValidationError{Err: fmt.Errorf("lot not found")}
	}
	if lot.Status == auction.AuctionPlayerSold {
		return s.RevertSale(ctx, auctionPlayerID, "")
	}
	return s.lots.UpdateStatus(ctx, auctionPlayerID, auction.AuctionPlayerUnsold, false)
}

// RevertSale reverses a mistaken sale: unassign player, credit wallet back, and clear lot sale fields.
func (s *SettlementService) RevertSale(ctx context.Context, auctionPlayerID string, reason string) error {
	if auctionPlayerID == "" {
		return &ValidationError{Err: fmt.Errorf("auction_player_id is required")}
	}
	lot, err := s.lots.GetByID(ctx, auctionPlayerID)
	if err != nil {
		return fmt.Errorf("get lot: %w", err)
	}
	if lot == nil {
		return &ValidationError{Err: fmt.Errorf("lot not found")}
	}
	if lot.Status != auction.AuctionPlayerSold {
		return &ValidationError{Err: fmt.Errorf("lot is not sold")}
	}
	if lot.SoldToTeamRegistrationID == "" || lot.FinalPrice <= 0 {
		return &ValidationError{Err: fmt.Errorf("sold lot missing sale fields")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, lot.SoldToTeamRegistrationID)
	if err != nil {
		return fmt.Errorf("get wallet: %w", err)
	}
	if w == nil {
		return &ValidationError{Err: fmt.Errorf("wallet not found")}
	}

	if err := s.lots.ClearSale(ctx, lot.ID); err != nil {
		return fmt.Errorf("clear sale: %w", err)
	}
	if err := s.regs.UnassignTeam(ctx, lot.TournamentPlayerRegistrationID); err != nil {
		return fmt.Errorf("unassign: %w", err)
	}

	now := s.nowMs()
	tx := &auction.WalletTransaction{
		ID:          uuid.New().String(),
		WalletID:    w.ID,
		Amount:      lot.FinalPrice,
		Type:        auction.WalletTxnCredit,
		ReferenceID: lot.ID,
		CreatedAt:   now,
	}
	if err := s.wallets.CreateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("wallet credit tx: %w", err)
	}
	if err := s.wallets.UpdateBalance(ctx, w.ID, w.Balance+lot.FinalPrice, now); err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}
	_ = reason // reserved for audit later
	return nil
}

// ResetTestAuction reverts all sold lots (wallet credit, unassign), removes all bids, and sets every
// lot back to pending. Only allowed when the auction run mode is test.
func (s *SettlementService) ResetTestAuction(ctx context.Context, auctionID string) error {
	if auctionID == "" {
		return &ValidationError{Err: fmt.Errorf("auction_id is required")}
	}
	a, err := s.auctions.GetByID(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if a.RunMode != auction.AuctionRunModeTest {
		return &ValidationError{Err: fmt.Errorf("reset is only allowed when auction runMode is test")}
	}

	lots, err := s.lots.ListByAuction(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("list lots: %w", err)
	}
	for _, lot := range lots {
		if lot == nil {
			continue
		}
		if lot.Status == auction.AuctionPlayerSold {
			if err := s.RevertSale(ctx, lot.ID, "test reset"); err != nil {
				return err
			}
		}
	}
	if err := s.bids.DeleteByAuction(ctx, auctionID); err != nil {
		return fmt.Errorf("delete bids: %w", err)
	}
	if err := s.lots.ResetAllLotsToPending(ctx, auctionID); err != nil {
		return fmt.Errorf("reset lots: %w", err)
	}
	if err := s.auctions.UpdateDisplayAuctionPlayer(ctx, auctionID, ""); err != nil {
		return fmt.Errorf("clear display lot: %w", err)
	}
	return nil
}

