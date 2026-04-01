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
	eventRepo := mocks.NewMockEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	teamRepo := mocks.NewMockTeamRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	svc := NewBidService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		EventRepo:         eventRepo,
		RegistrationRepo:  regRepo,
		TeamRepo:          teamRepo,
		BidRepo:           bidRepo,
		WalletRepo:        walletRepo,
		NowMs:             func() int64 { return 123 },
	})

	lot := &auction.AuctionPlayer{ID: "lot1", AuctionID: "auc1", Status: auction.AuctionPlayerActive, BasePrice: 0}
	auc := &auction.Auction{ID: "auc1", TournamentID: "t1", EventID: "e1"}
	ev := &auction.Event{
		ID: "e1",
		Attrs: auction.EventAttrs{TeamEventRules: &auction.TeamEventRules{
			IsAuction:         true,
			MinPlayersPerTeam: 3,
			MaxPlayersPerTeam: 5,
			BaseBid:           100,
			MaxBid:            1000,
		}},
	}
	team := &auction.TournamentTeamRegistration{ID: "team1", TournamentID: "t1", EventID: "e1"}

	// Already sold 1 player to team; remaining min after winning this lot would be 1.
	lots := []*auction.AuctionPlayer{
		{ID: "sold1", AuctionID: "auc1", Status: auction.AuctionPlayerSold, SoldToTeamRegistrationID: "team1"},
	}

	lotRepo.EXPECT().GetByID(gomock.Any(), "lot1").Return(lot, nil)
	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	eventRepo.EXPECT().GetByID(gomock.Any(), "e1").Return(ev, nil)
	teamRepo.EXPECT().GetByID(gomock.Any(), "team1").Return(team, nil)
	walletRepo.EXPECT().GetByTournamentEventTeam(gomock.Any(), "t1", "e1", "team1").
		Return(&auction.Wallet{ID: "w1", Balance: 150 /* not enough to keep 1*BaseBid after bidding 100 */}, nil)
	lotRepo.EXPECT().ListByAuction(gomock.Any(), "auc1").Return(lots, nil)

	_, err := svc.PlaceBid(ctx, PlaceBidInput{AuctionPlayerID: "lot1", TeamRegistrationID: "team1", Amount: 100})
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
	eventRepo := mocks.NewMockEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	teamRepo := mocks.NewMockTeamRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	svc := NewSettlementService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		EventRepo:         eventRepo,
		RegistrationRepo:  regRepo,
		TeamRepo:          teamRepo,
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
	auc := &auction.Auction{ID: "auc1", TournamentID: "t1", EventID: "e1"}
	highest := &auction.Bid{ID: "bid1", AuctionPlayerID: "lot1", TeamRegistrationID: "team1", Amount: 300}
	w := &auction.Wallet{ID: "w1", Balance: 500}

	lotRepo.EXPECT().GetByID(gomock.Any(), "lot1").Return(lot, nil)
	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	bidRepo.EXPECT().GetHighestBid(gomock.Any(), "lot1").Return(highest, nil)
	walletRepo.EXPECT().GetByTournamentEventTeam(gomock.Any(), "t1", "e1", "team1").Return(w, nil)

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

func TestService_CreateLot_ScreeningGenderAndAge(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	auctionRepo := mocks.NewMockAuctionRepository(ctrl)
	lotRepo := mocks.NewMockAuctionPlayerRepository(ctrl)
	eventRepo := mocks.NewMockEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	teamRepo := mocks.NewMockTeamRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	now := int64(1_700_000_000_000) // ms
	svc := NewLotService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		EventRepo:         eventRepo,
		RegistrationRepo:  regRepo,
		TeamRepo:          teamRepo,
		BidRepo:           bidRepo,
		WalletRepo:        walletRepo,
		NowMs:             func() int64 { return now },
	})

	auc := &auction.Auction{
		ID:           "auc1",
		TournamentID: "t1",
		EventID:      "e1",
	}
	ev := &auction.Event{ID: "e1", Attrs: auction.EventAttrs{TeamEventRules: &auction.TeamEventRules{IsAuction: true, BaseBid: 100}}}

	reg := &auction.TournamentPlayerRegistration{
		ID:           "reg1",
		TournamentID: "t1",
		EventID:      "e1",
	}

	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	eventRepo.EXPECT().GetByID(gomock.Any(), "e1").Return(ev, nil)
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
	eventRepo := mocks.NewMockEventRepository(ctrl)
	regRepo := mocks.NewMockRegistrationRepository(ctrl)
	teamRepo := mocks.NewMockTeamRepository(ctrl)
	bidRepo := mocks.NewMockBidRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)

	svc := NewLotService(Deps{
		AuctionRepo:       auctionRepo,
		AuctionPlayerRepo: lotRepo,
		EventRepo:         eventRepo,
		RegistrationRepo:  regRepo,
		TeamRepo:          teamRepo,
		BidRepo:           bidRepo,
		WalletRepo:        walletRepo,
		NowMs:             func() int64 { return 123 },
	})

	auc := &auction.Auction{ID: "auc1", TournamentID: "t1", EventID: "e1"}
	filter := auction.RegistrationFilter{Gender: "FEMALE", MaxAgeYears: 30}
	expected := []*auction.TournamentPlayerRegistration{{ID: "reg1"}, {ID: "reg2"}}

	auctionRepo.EXPECT().GetByID(gomock.Any(), "auc1").Return(auc, nil)
	regRepo.EXPECT().ListEligible(gomock.Any(), "t1", "e1", filter).Return(expected, nil)

	got, err := svc.ListEligibleRegistrations(ctx, "auc1", filter)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 2 || got[0].ID != "reg1" || got[1].ID != "reg2" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

