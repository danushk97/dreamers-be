package auctionservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/auction"
)

// testAuctionResetCreditPaisa is ₹100,000 in paise (the domain wallet/bid unit).
const testAuctionResetCreditPaisa int64 = 10_000_000

type SettlementService struct {
	auctions         auction.AuctionRepository
	lots             auction.AuctionPlayerRepository
	bids             auction.BidRepository
	regs             auction.RegistrationRepository
	teams            auction.TeamRepository
	tournamentEvents auction.TournamentEventRepository
	wallets          auction.WalletRepository
	nowMs            func() int64
}

func NewSettlementService(d Deps) *SettlementService {
	return &SettlementService{
		auctions:         d.AuctionRepo,
		lots:             d.AuctionPlayerRepo,
		bids:             d.BidRepo,
		regs:             d.RegistrationRepo,
		teams:            d.TeamRepo,
		tournamentEvents: d.TournamentEventRepo,
		wallets:          d.WalletRepo,
		nowMs:            d.now(),
	}
}

type SellInput struct {
	AuctionPlayerID string
	ExpectedBidID   string // optional optimistic check; can be empty
}

type RetainInput struct {
	AuctionPlayerID    string
	TeamRegistrationID string
	Amount             int64
}

type SubstituteInput struct {
	AuctionPlayerID    string
	TeamRegistrationID string
	Amount             int64
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
	if err := requireAuctionRunning(a); err != nil {
		return err
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

	if err := s.lots.MarkSold(ctx, lot.ID, highest.Amount, highest.TeamRegistrationID, auction.AuctionPlayerNotes{}); err != nil {
		return fmt.Errorf("mark sold: %w", err)
	}
	if err := s.regs.AssignToTeam(ctx, lot.TournamentPlayerRegistrationID, highest.TeamRegistrationID); err != nil {
		return fmt.Errorf("assign to team: %w", err)
	}

	now := s.nowMs()
	newBalance := w.Balance - highest.Amount
	tx := &auction.WalletTransaction{
		ID:            uuid.New().String(),
		WalletID:      w.ID,
		Amount:        highest.Amount,
		Type:          auction.WalletTxnDebit,
		ReferenceID:   lot.ID,
		WalletBalance: newBalance,
		CreatedAt:     now,
	}
	if err := s.wallets.CreateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("wallet debit tx: %w", err)
	}
	if err := s.wallets.UpdateBalance(ctx, w.ID, newBalance, now); err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}
	return nil
}

// RetainPlayer directly sells a lot to a team at a provided retain amount.
func (s *SettlementService) RetainPlayer(ctx context.Context, in RetainInput) error {
	if in.AuctionPlayerID == "" || in.TeamRegistrationID == "" {
		return &ValidationError{Err: fmt.Errorf("auction_player_id and team_registration_id are required")}
	}
	if in.Amount <= 0 {
		return &ValidationError{Err: fmt.Errorf("amount must be > 0")}
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
	if lot.Status == auction.AuctionPlayerSkipped {
		return &ValidationError{Err: fmt.Errorf("lot is skipped")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return err
	}
	if a.Rules.MaxRetainPlayerAmount > 0 && in.Amount > a.Rules.MaxRetainPlayerAmount {
		return &ValidationError{Err: fmt.Errorf("retain amount exceeds MaxRetainPlayerAmount")}
	}
	team, err := s.teams.GetByID(ctx, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}
	if team == nil {
		return &ValidationError{Err: fmt.Errorf("team not found")}
	}
	if team.TournamentID != a.TournamentID || team.TournamentEventID != a.TournamentEventID {
		return &ValidationError{Err: fmt.Errorf("team not in this tournament event")}
	}
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil || te.Attrs.TeamEventRules == nil {
		return &ValidationError{Err: fmt.Errorf("tournament event rules not found")}
	}

	w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("get wallet: %w", err)
	}
	if w == nil {
		return &ValidationError{Err: fmt.Errorf("wallet not found")}
	}
	if w.Balance < in.Amount {
		return &ValidationError{Err: fmt.Errorf("insufficient wallet balance to retain")}
	}

	soldCount, err := countSoldToTeam(ctx, s.lots, a.ID, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("count sold: %w", err)
	}
	if te.Attrs.TeamEventRules.MaxPlayersPerTeam > 0 && soldCount >= te.Attrs.TeamEventRules.MaxPlayersPerTeam {
		return &ValidationError{Err: fmt.Errorf("team already reached max players")}
	}

	retainedCount, err := countRetainedToTeam(ctx, s.lots, a.ID, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("count retained: %w", err)
	}
	if a.Rules.MaxRetainPlayers > 0 && retainedCount >= a.Rules.MaxRetainPlayers {
		return &ValidationError{Err: fmt.Errorf("team already reached max retained players")}
	}

	if err := s.lots.MarkSold(ctx, lot.ID, in.Amount, in.TeamRegistrationID, auction.AuctionPlayerNotes{IsRetained: true}); err != nil {
		return fmt.Errorf("mark retained sold: %w", err)
	}
	if err := s.regs.AssignToTeam(ctx, lot.TournamentPlayerRegistrationID, in.TeamRegistrationID); err != nil {
		return fmt.Errorf("assign to team: %w", err)
	}
	now := s.nowMs()
	newBalance := w.Balance - in.Amount
	tx := &auction.WalletTransaction{
		ID:            uuid.New().String(),
		WalletID:      w.ID,
		Amount:        in.Amount,
		Type:          auction.WalletTxnDebit,
		ReferenceID:   lot.ID,
		WalletBalance: newBalance,
		CreatedAt:     now,
	}
	if err := s.wallets.CreateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("wallet debit tx: %w", err)
	}
	if err := s.wallets.UpdateBalance(ctx, w.ID, newBalance, now); err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}
	return nil
}

// SubstitutePlayer marks a lot sold with substitute details and assigns player to team.
// No wallet movement or wallet transactions are created.
func (s *SettlementService) SubstitutePlayer(ctx context.Context, in SubstituteInput) error {
	if in.AuctionPlayerID == "" || in.TeamRegistrationID == "" {
		return &ValidationError{Err: fmt.Errorf("auction_player_id and team_registration_id are required")}
	}
	if in.Amount <= 0 {
		return &ValidationError{Err: fmt.Errorf("amount must be > 0")}
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
	if lot.Status == auction.AuctionPlayerSkipped {
		return &ValidationError{Err: fmt.Errorf("lot is skipped")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return err
	}
	team, err := s.teams.GetByID(ctx, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}
	if team == nil {
		return &ValidationError{Err: fmt.Errorf("team not found")}
	}
	if team.TournamentID != a.TournamentID || team.TournamentEventID != a.TournamentEventID {
		return &ValidationError{Err: fmt.Errorf("team not in this tournament event")}
	}
	te, err := s.tournamentEvents.GetByID(ctx, a.TournamentEventID)
	if err != nil {
		return fmt.Errorf("get tournament event: %w", err)
	}
	if te == nil || te.Attrs.TeamEventRules == nil {
		return &ValidationError{Err: fmt.Errorf("tournament event rules not found")}
	}

	soldCount, err := countSoldToTeam(ctx, s.lots, a.ID, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("count sold: %w", err)
	}
	if te.Attrs.TeamEventRules.MaxPlayersPerTeam > 0 && soldCount < te.Attrs.TeamEventRules.MaxPlayersPerTeam {
		return &ValidationError{Err: fmt.Errorf("team has not auctioned all players")}
	}
	substituteCount, err := countSubstitutedToTeam(ctx, s.lots, a.ID, in.TeamRegistrationID)
	if err != nil {
		return fmt.Errorf("count substitute: %w", err)
	}
	if a.Rules.MaxSubstitutePlayers > 0 && substituteCount >= a.Rules.MaxSubstitutePlayers {
		return &ValidationError{Err: fmt.Errorf("team already reached max substitute players")}
	}

	if err := s.lots.MarkSold(ctx, lot.ID, in.Amount, in.TeamRegistrationID, auction.AuctionPlayerNotes{
		SubstitueDetails: &auction.AuctionPlayerSubstitueDetails{Amount: in.Amount},
	}); err != nil {
		return fmt.Errorf("mark substitute sold: %w", err)
	}
	if err := s.regs.AssignToTeam(ctx, lot.TournamentPlayerRegistrationID, in.TeamRegistrationID); err != nil {
		return fmt.Errorf("assign to team: %w", err)
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
	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return err
	}
	if lot.Status == auction.AuctionPlayerSold {
		return s.RevertSale(ctx, auctionPlayerID, "")
	}
	return s.lots.UpdateStatus(ctx, auctionPlayerID, auction.AuctionPlayerUnsold, false)
}

// RelistUnsold moves an unsold lot back to pending on the same row (same as RevertSale for unsold, without allowing sold lots).
func (s *SettlementService) RelistUnsold(ctx context.Context, auctionPlayerID string) error {
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
	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return err
	}
	if lot.Status != auction.AuctionPlayerUnsold {
		return &ValidationError{Err: fmt.Errorf("lot must be unsold to relist")}
	}
	return s.RevertSale(ctx, auctionPlayerID, "")
}

// RevertSale moves a lot from sold or unsold back to pending: clears sale fields, unassigns the player,
// and for sold lots (except substitute, which never debited the wallet) credits the buyer's wallet.
// Other statuses are rejected.
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
	if lot.Status != auction.AuctionPlayerSold && lot.Status != auction.AuctionPlayerUnsold {
		return &ValidationError{Err: fmt.Errorf("lot must be sold or unsold to revert")}
	}

	a, err := s.auctions.GetByID(ctx, lot.AuctionID)
	if err != nil {
		return fmt.Errorf("get auction: %w", err)
	}
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if err := requireAuctionRunning(a); err != nil {
		return err
	}

	var w *auction.Wallet
	if lot.Status == auction.AuctionPlayerSold {
		if lot.SoldToTeamRegistrationID == "" || lot.FinalPrice <= 0 {
			return &ValidationError{Err: fmt.Errorf("sold lot missing sale fields")}
		}
		// Substitute sales never touch the wallet; skip load and credit.
		if lot.Notes.SubstitueDetails == nil {
			w, err = s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, lot.SoldToTeamRegistrationID)
			if err != nil {
				return fmt.Errorf("get wallet: %w", err)
			}
			if w == nil {
				return &ValidationError{Err: fmt.Errorf("wallet not found")}
			}
		}
	}

	if err := s.lots.ClearSale(ctx, lot.ID); err != nil {
		return fmt.Errorf("clear sale: %w", err)
	}
	if err := s.regs.UnassignTeam(ctx, lot.TournamentPlayerRegistrationID); err != nil {
		return fmt.Errorf("unassign: %w", err)
	}

	if w != nil {
		now := s.nowMs()
		newBalance := w.Balance + lot.FinalPrice
		tx := &auction.WalletTransaction{
			ID:            uuid.New().String(),
			WalletID:      w.ID,
			Amount:        lot.FinalPrice,
			Type:          auction.WalletTxnCredit,
			ReferenceID:   lot.ID,
			WalletBalance: newBalance,
			CreatedAt:     now,
		}
		if err := s.wallets.CreateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("wallet credit tx: %w", err)
		}
		if err := s.wallets.UpdateBalance(ctx, w.ID, newBalance, now); err != nil {
			return fmt.Errorf("update wallet balance: %w", err)
		}
	}
	_ = reason // reserved for audit later
	return nil
}

// revertSoldLotStructurallyForTestReset clears sale fields and unassigns the player (no wallet writes).
func (s *SettlementService) revertSoldLotStructurallyForTestReset(ctx context.Context, lot *auction.AuctionPlayer) error {
	if lot.SoldToTeamRegistrationID == "" || lot.FinalPrice <= 0 {
		return &ValidationError{Err: fmt.Errorf("sold lot missing sale fields")}
	}
	if err := s.lots.ClearSale(ctx, lot.ID); err != nil {
		return fmt.Errorf("clear sale: %w", err)
	}
	if err := s.regs.UnassignTeam(ctx, lot.TournamentPlayerRegistrationID); err != nil {
		return fmt.Errorf("unassign: %w", err)
	}
	return nil
}

// ResetTestAuction clears sold lots, deletes every wallet ledger row (credits and debits) for all team
// wallets in the auction's tournament event, credits each registered team wallet with ₹100,000,
// removes all bids, sets every lot back to pending, clears the display lot, and sets auction status to created.
// Only allowed when the auction runMode is test.
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
			if err := s.revertSoldLotStructurallyForTestReset(ctx, lot); err != nil {
				return err
			}
		}
	}
	if err := s.wallets.DeleteAllWalletTransactionsForTournamentEvent(ctx, a.TournamentID, a.TournamentEventID); err != nil {
		return fmt.Errorf("delete wallet transactions for tournament event: %w", err)
	}
	now := s.nowMs()
	if err := s.wallets.ZeroWalletBalancesForTournamentEvent(ctx, a.TournamentID, a.TournamentEventID, now); err != nil {
		return fmt.Errorf("zero wallet balances for tournament event: %w", err)
	}
	teamRows, err := s.teams.ListByTournamentEvent(ctx, a.TournamentID, a.TournamentEventID)
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}
	newBal := testAuctionResetCreditPaisa
	for _, team := range teamRows {
		if team == nil {
			continue
		}
		w, err := s.wallets.GetByTournamentEventTeam(ctx, a.TournamentID, a.TournamentEventID, team.ID)
		if err != nil {
			return fmt.Errorf("get wallet for team %s: %w", team.ID, err)
		}
		if w == nil {
			continue
		}
		tx := &auction.WalletTransaction{
			ID:            uuid.New().String(),
			WalletID:      w.ID,
			Amount:        testAuctionResetCreditPaisa,
			Type:          auction.WalletTxnCredit,
			ReferenceID:   auctionID,
			WalletBalance: newBal,
			CreatedAt:     now,
		}
		if err := s.wallets.CreateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("wallet test-reset credit: %w", err)
		}
		if err := s.wallets.UpdateBalance(ctx, w.ID, newBal, now); err != nil {
			return fmt.Errorf("update wallet balance: %w", err)
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
	if err := s.auctions.UpdateStatus(ctx, auctionID, auction.AuctionStatusCreated); err != nil {
		return fmt.Errorf("reset auction status: %w", err)
	}
	return nil
}
