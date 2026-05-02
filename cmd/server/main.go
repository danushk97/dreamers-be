package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	ginhandler "github.com/dreamers-be/internal/adapter/http/gin"
	"github.com/dreamers-be/internal/adapter/persistence/postgres"
	"github.com/dreamers-be/internal/adapter/storage/s3"
	appconfig "github.com/dreamers-be/internal/config"
	"github.com/dreamers-be/internal/domain/storage"
	playersrv "github.com/dreamers-be/internal/server/player"
	uploadsrv "github.com/dreamers-be/internal/server/upload"
	auctionservice "github.com/dreamers-be/internal/service/auction"
	playersvc "github.com/dreamers-be/internal/service/player"
	uploadsvc "github.com/dreamers-be/internal/service/upload"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("Config loaded, server port=%s", cfg.Server.Port)
	log.Printf("S3 config: bucket=%s region=%s", cfg.S3.Bucket, cfg.S3.Region)

	log.Printf("Connecting to database...")
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	log.Print("Database connected")

	maxMB := cfg.S3.MaxSizeMB
	if maxMB <= 0 {
		maxMB = 2
	}

	// S3 uploader (optional - noop if bucket not configured)
	var uploader storage.FileUploader
	if cfg.S3.Bucket != "" {
		ctx := context.Background()
		s3u, err := s3.NewFileUploader(ctx, s3.Config{
			Bucket:    cfg.S3.Bucket,
			Region:    cfg.S3.Region,
			BaseURL:   cfg.S3.BaseURL,
			MaxSizeMB: maxMB,
			AccessKey: cfg.S3.AccessKey,
			SecretKey: cfg.S3.SecretKey,
		})
		if err != nil {
			log.Fatalf("s3 uploader: %v", err)
		}
		uploader = s3u
		log.Printf("S3 uploader configured, bucket=%s region=%s", cfg.S3.Bucket, cfg.S3.Region)
	} else {
		uploader = &noopUploader{}
		log.Printf("S3 not configured, using noop uploader")
	}

	playerRepo := postgres.NewPlayerRepository(db)
	playerSvc := playersvc.NewPlayerService(playerRepo)
	uploadSvc := uploadsvc.NewUploadService(uploader, maxMB)

	playerSrv := playersrv.NewPlayerServer(playerSvc)
	uploadSrv := uploadsrv.NewUploadServer(uploadSvc)

	var presigner storage.Presigner
	if p, ok := uploader.(storage.Presigner); ok {
		presigner = p
		log.Printf("Presigner available for presigned URLs")
	} else {
		log.Printf("Presigner not available (noop uploader), presigned URLs disabled")
	}
	ph := ginhandler.NewPlayerHandler(playerSrv, presigner)
	uh := ginhandler.NewUploadHandler(uploadSrv, presigner)
	log.Printf("Use cases and handlers initialized")

	// Auction core flow wiring (MVP)
	tournamentRepo := postgres.NewTournamentRepository(db)
	tournamentEventRepo := postgres.NewTournamentEventRepository(db)
	auctionRepo := postgres.NewAuctionRepository(db)
	teamRepo := postgres.NewTeamRepository(db)
	registrationRepo := postgres.NewRegistrationRepository(db)
	auctionPlayerRepo := postgres.NewAuctionPlayerRepository(db)
	bidRepo := postgres.NewBidRepository(db)
	walletRepo := postgres.NewWalletRepository(db)

	deps := auctionservice.Deps{
		AuctionRepo:         auctionRepo,
		AuctionPlayerRepo:   auctionPlayerRepo,
		TournamentRepo:      tournamentRepo,
		TournamentEventRepo: tournamentEventRepo,
		RegistrationRepo:    registrationRepo,
		TeamRepo:            teamRepo,
		BidRepo:             bidRepo,
		WalletRepo:          walletRepo,
		NowMs:               func() int64 { return time.Now().UnixMilli() },
	}
	auctionSvc := auctionservice.NewAuctionService(deps)
	lotSvc := auctionservice.NewLotService(deps)
	bidSvc := auctionservice.NewBidService(deps)
	settlementSvc := auctionservice.NewSettlementService(deps)

	auctionHandler := ginhandler.NewAuctionHandler(
		auctionSvc,
		lotSvc,
		bidSvc,
		settlementSvc,
		auctionPlayerRepo,
		bidRepo,
		walletRepo,
		registrationRepo,
		playerRepo,
		presigner,
	)

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), corsMiddleware())

	adminAuth := ginhandler.BasicAuth(ginhandler.BasicAuthCredentials)

	api := r.Group("/v1")
	{
		api.POST("/upload", uh.Upload)
		api.POST("/players", ph.Create)
		api.GET("/tournaments/:tournamentId", auctionHandler.GetTournament)
		api.GET("/tournaments/:tournamentId/events/:tournamentEventId/teams", auctionHandler.ListRegisteredTeams)
		api.GET("/auctions/:auctionId", auctionHandler.GetAuction)
		api.GET("/auctions/:auctionId/relay/stream", auctionHandler.StreamAuctionRelay)
		api.GET("/auctions/:auctionId/lots/by-registration-serial/:serialNumber", auctionHandler.GetLotByRegistrationSerial)
		api.GET("/auctions/:auctionId/lots", auctionHandler.ListLotsByAuction)
		api.GET("/auction-players/:auctionPlayerId/bids", auctionHandler.ListBidsByAuctionPlayer)
		api.GET("/wallets", auctionHandler.GetWallet)
		api.GET("/wallets/:walletId/transactions/:transactionType", auctionHandler.ListWalletTransactions)
		api.POST("/auctions", adminAuth, auctionHandler.CreateAuction)
		api.PATCH("/auctions/:auctionId", adminAuth, auctionHandler.PatchAuction)
		api.PUT("/auctions/:auctionId/display-lot", adminAuth, auctionHandler.PutAuctionDisplayLot)
		api.POST("/auctions/:auctionId/reset", adminAuth, auctionHandler.ResetTestAuction)
		api.POST("/auctions/:auctionId/eligible", adminAuth, auctionHandler.ListEligibleRegistrations)
		api.POST("/auctions/:auctionId/lots/bulk", adminAuth, auctionHandler.CreateLotsBulk)
		api.POST("/auction-players/:auctionPlayerId/bids", adminAuth, auctionHandler.PlaceBid)
		api.POST("/auction-players/:auctionPlayerId/bids/revert", adminAuth, auctionHandler.RevertLatestBid)
		api.POST("/auction-players/:auctionPlayerId/retain", adminAuth, auctionHandler.RetainPlayer)
		api.POST("/auction-players/:auctionPlayerId/substitute", adminAuth, auctionHandler.SubstitutePlayer)
		api.POST("/auction-players/:auctionPlayerId/sell", adminAuth, auctionHandler.SellCurrentLot)
		api.POST("/auction-players/:auctionPlayerId/unsold", adminAuth, auctionHandler.MarkUnsold)
		api.POST("/auction-players/:auctionPlayerId/relist", adminAuth, auctionHandler.RelistUnsold)
		api.POST("/auction-players/:auctionPlayerId/revert", adminAuth, auctionHandler.RevertSale)
		// api.GET("/players", ginhandler.BasicAuth(ginhandler.BasicAuthCredentials), ph.List)
		api.GET("/players", ph.List)
		// Single player by id: public profile; send Authorization: Basic (admin) to include aadharCardImageURL.
		api.GET("/players/:id", ph.Get)
	}
	log.Printf("API routes registered")

	addr := ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
