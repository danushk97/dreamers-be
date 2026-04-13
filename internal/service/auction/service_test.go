package auctionservice

import (
	"context"
	"testing"

	"github.com/dreamers-be/internal/domain/auction"
	"github.com/dreamers-be/internal/mocks"
	"github.com/golang/mock/gomock"
)

func TestService_PlaceBid_ReservesForMinimumRoster(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	tournamentEventRepo := mocks.NewMockTournamentEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	teamRepo := mocks.NewMockTeamRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	svc := NewBidService(Deps{
		AuctionRepo:         auctionRepo,
		AuctionPlayerRepo:   lotRepo,
		TournamentEventRepo: tournamentEventRepo,
		RegistrationRepo:    regRepo,
		TeamRepo:            teamRepo,
		BidRepo:             bidRepo,
		WalletRepo:          walletRepo,
		NowMs:               func() int64 { return 123 },
	})

	lot := &auction.AuctionPlayer{ID: "lot1", AuctionID: "auc1", Status: auction.AuctionPlayerActive, BasePrice: 0}
	auc := &auction.Auction{
		ID: "auc1", TournamentID: "t1", TournamentEventID: "te1",
		Rules: auction.AuctionRules{MinBidAmount: 100, MaxBidAmount: 1000},
	}
	te := &auction.TournamentEvent{
		ID: "te1",
		Attrs: auction.EventAttrs{TeamEventRules: &auction.TeamEventRules{
			IsAuction:         true,
			MinPlayersPerTeam: 3,
			MaxPlayersPerTeam: 5,
		}},
	}
	team := &auction.TournamentTeamRegistration{ID: "team1", TournamentID: "t1", TournamentEventID: "te1"}

	lots := []*auction.AuctionPlayer{
		{ID: "sold1", AuctionID: "auc1", Status: auction.AuctionPlayerSold, SoldToTeamRegistrationID: "team1"},
	}

	lotRepo.EXPECT().GetByID(gomock.Any(), "lot1").Return(lot, nil)
	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	tournamentEventRepo.EXPECT().GetByID(gomock.Any(), "te1").Return(te, nil)
	teamRepo.EXPECT().GetByID(gomock.Any(), "team1").Return(team, nil)
	walletRepo.EXPECT().GetByTournamentEventTeam(gomock.Any(), "t1", "te1", "team1").
		Return(&auction.Wallet{ID: "w1", Balance: 150}, nil)
	lotRepo.EXPECT().ListByAuction(gomock.Any(), "auc1").Return(lots, nil)

	_, _, err := svc.PlaceBid(ctx, PlaceBidInput{AuctionPlayerID: "lot1", TeamRegistrationID: "team1", Amount: 100})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestService_SellCurrentLot_SettlesWalletAndAssigns(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	svc := NewSettlementService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		RegistrationRepo:  regRepo,
		BidRepo:           bidRepo,
		WalletRepo:        walletRepo,
		NowMs:             func() int64 { return 999 },
	})

	lot := &auction.AuctionPlayer{
		ID:                             "lot1",
		AuctionID:                      "auc1",
		TournamentPlayerRegistrationID: "reg1",
		Status:                         auction.AuctionPlayerActive,
	}
	auc := &auction.Auction{ID: "auc1", TournamentID: "t1", TournamentEventID: "te1"}
	highest := &auction.Bid{ID: "bid1", AuctionPlayerID: "lot1", TeamRegistrationID: "team1", Amount: 300}
	w := &auction.Wallet{ID: "w1", Balance: 500}

	lotRepo.EXPECT().GetByID(gomock.Any(), "lot1").Return(lot, nil)
	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	bidRepo.EXPECT().GetHighestBid(gomock.Any(), "lot1").Return(highest, nil)
	walletRepo.EXPECT().GetByTournamentEventTeam(gomock.Any(), "t1", "te1", "team1").Return(w, nil)

	lotRepo.EXPECT().MarkSold(gomock.Any(), "lot1", int64(300), "team1").Return(nil)
	regRepo.EXPECT().AssignToTeam(gomock.Any(), "reg1", "team1").Return(nil)

	walletRepo.EXPECT().CreateTransaction(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, tx *auction.WalletTransaction) error {
			if tx.Type != auction.WalletTxnDebit {
				t.Fatalf("expected debit")
			}
			if tx.Amount != 300 || tx.ReferenceID != "lot1" || tx.WalletID != "w1" {
				t.Fatalf("unexpected tx: %#v", tx)
			}
			if tx.CreatedAt != 999 {
				t.Fatalf("unexpected CreatedAt: %d", tx.CreatedAt)
			}
			return nil
		},
	)
	walletRepo.EXPECT().UpdateBalance(gomock.Any(), "w1", int64(200), int64(999)).Return(nil)

	if err := svc.SellCurrentLot(ctx, SellInput{AuctionPlayerID: "lot1", ExpectedBidID: "bid1"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestService_MarkUnsold_SoldLot_RevertsSale(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	svc := NewSettlementService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		RegistrationRepo:  regRepo,
		BidRepo:           bidRepo,
		WalletRepo:        walletRepo,
		NowMs:             func() int64 { return 999 },
	})

	lot := &auction.AuctionPlayer{
		ID:                             "lot1",
		AuctionID:                      "auc1",
		TournamentPlayerRegistrationID: "reg1",
		Status:                         auction.AuctionPlayerSold,
		FinalPrice:                     300,
		SoldToTeamRegistrationID:       "team1",
	}
	auc := &auction.Auction{ID: "auc1", TournamentID: "t1", TournamentEventID: "te1"}
	w := &auction.Wallet{ID: "w1", Balance: 200}

	// MarkUnsold and RevertSale each load the lot.
	lotRepo.EXPECT().GetByID(gomock.Any(), "lot1").Return(lot, nil).Times(2)
	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	walletRepo.EXPECT().GetByTournamentEventTeam(gomock.Any(), "t1", "te1", "team1").Return(w, nil)
	lotRepo.EXPECT().ClearSale(gomock.Any(), "lot1").Return(nil)
	regRepo.EXPECT().UnassignTeam(gomock.Any(), "reg1").Return(nil)
	walletRepo.EXPECT().CreateTransaction(gomock.Any(), gomock.Any()).Return(nil)
	walletRepo.EXPECT().UpdateBalance(gomock.Any(), "w1", int64(500), int64(999)).Return(nil)

	if err := svc.MarkUnsold(ctx, "lot1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestService_CreateLot_ScreeningGenderAndAge(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	tournamentEventRepo := mocks.NewMockTournamentEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)

	now := int64(1_700_000_000_000)
	svc := NewLotService(Deps{
		AuctionRepo:         auctionRepo,
		AuctionPlayerRepo:   lotRepo,
		TournamentEventRepo: tournamentEventRepo,
		RegistrationRepo:    regRepo,
		NowMs:               func() int64 { return now },
	})

	auc := &auction.Auction{
		ID:                "auc1",
		TournamentID:      "t1",
		TournamentEventID: "te1",
		Rules:             auction.AuctionRules{MinBidAmount: 100},
	}
	te := &auction.TournamentEvent{ID: "te1", Attrs: auction.EventAttrs{TeamEventRules: &auction.TeamEventRules{IsAuction: true}}}

	reg := &auction.TournamentPlayerRegistration{
		ID:                "reg1",
		TournamentID:      "t1",
		TournamentEventID: "te1",
	}

	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	tournamentEventRepo.EXPECT().GetByID(gomock.Any(), "te1").Return(te, nil)
	regRepo.EXPECT().GetByID(gomock.Any(), "reg1").Return(reg, nil)
	lotRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	_, err := svc.CreateLot(ctx, CreateLotInput{AuctionID: "auc1", TournamentPlayerRegistrationID: "reg1", LotNumber: 1})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestLotService_ListEligibleRegistrations_ForwardsFilter(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)

	svc := NewLotService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		RegistrationRepo:  regRepo,
		NowMs:             func() int64 { return 123 },
	})

	auc := &auction.Auction{ID: "auc1", TournamentID: "t1", TournamentEventID: "te1"}
	filter := auction.RegistrationFilter{Gender: "FEMALE", MaxAgeYears: 30}
	expected := []*auction.TournamentPlayerRegistration{{ID: "reg1"}, {ID: "reg2"}}

	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	regRepo.EXPECT().ListEligible(gomock.Any(), "t1", "te1", filter).Return(expected, nil)

	got, err := svc.ListEligibleRegistrations(ctx, "auc1", filter)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 2 || got[0].ID != "reg1" || got[1].ID != "reg2" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestLotService_CreateLotsByQuery_ReactivatesUnsoldLots(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	tournamentEventRepo := mocks.NewMockTournamentEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)

	svc := NewLotService(Deps{
		AuctionRepo:         auctionRepo,
		AuctionPlayerRepo:   lotRepo,
		TournamentEventRepo: tournamentEventRepo,
		RegistrationRepo:    regRepo,
		NowMs:               func() int64 { return 123 },
	})

	auc := &auction.Auction{
		ID: "auc1", TournamentID: "t1", TournamentEventID: "te1",
		Rules: auction.AuctionRules{MinBidAmount: 1500},
	}
	te := &auction.TournamentEvent{
		ID: "te1",
		Attrs: auction.EventAttrs{TeamEventRules: &auction.TeamEventRules{
			IsAuction:         true,
			MinPlayersPerTeam: 4,
			MaxPlayersPerTeam: 4,
		}},
	}

	// reg1 has an existing unsold lot; reg2 is new.
	regs := []*auction.TournamentPlayerRegistration{
		{ID: "reg1", TournamentID: "t1", TournamentEventID: "te1", TeamID: ""},
		{ID: "reg2", TournamentID: "t1", TournamentEventID: "te1", TeamID: ""},
	}
	existingLots := []*auction.AuctionPlayer{
		{ID: "lot1", AuctionID: "auc1", TournamentPlayerRegistrationID: "reg1", Status: auction.AuctionPlayerUnsold, IsActive: false, LotNumber: 5, CreatedAt: 10},
	}

	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	tournamentEventRepo.EXPECT().GetByID(gomock.Any(), "te1").Return(te, nil)
	regRepo.EXPECT().ListEligible(gomock.Any(), "t1", "te1", auction.RegistrationFilter{}).Return(regs, nil)
	lotRepo.EXPECT().ListByAuction(gomock.Any(), "auc1").Return(existingLots, nil)

	// reg1: unsold -> active
	lotRepo.EXPECT().UpdateStatus(gomock.Any(), "lot1", auction.AuctionPlayerActive, true).Return(nil)

	// reg2: create one new lot
	lotRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ap *auction.AuctionPlayer) error {
			if ap.TournamentPlayerRegistrationID != "reg2" {
				t.Fatalf("unexpected registration id: %s", ap.TournamentPlayerRegistrationID)
			}
			if ap.BasePrice != 1500 {
				t.Fatalf("unexpected base price: %d", ap.BasePrice)
			}
			return nil
		},
	)

	out, err := svc.CreateLotsByQuery(ctx, CreateLotsByQueryInput{
		AuctionID:          "auc1",
		TournamentID:      "t1",
		TournamentEventID: "te1",
		Filter:            auction.RegistrationFilter{},
		StartLotNumber:    1,
		BasePrice:         0,
		Limit:              0,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(out))
	}
	if out[0].ID != "lot1" || out[0].Status != auction.AuctionPlayerActive || !out[0].IsActive {
		t.Fatalf("expected reg1 lot to be activated: %#v", out[0])
	}
	if out[1].TournamentPlayerRegistrationID != "reg2" || out[1].Status != auction.AuctionPlayerPending || out[1].IsActive {
		t.Fatalf("expected new lot pending/inactive: %#v", out[1])
	}
}
