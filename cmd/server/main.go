package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	ginhandler "github.com/dreamers-be/internal/adapter/http/gin"
	"github.com/dreamers-be/internal/adapter/persistence/postgres"
	"github.com/dreamers-be/internal/adapter/storage/gdrive"
	"github.com/dreamers-be/internal/domain/storage"
	appconfig "github.com/dreamers-be/internal/config"
	"github.com/dreamers-be/internal/usecase/player"
	uploaduc "github.com/dreamers-be/internal/usecase/upload"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	maxMB := cfg.GDrive.MaxSizeMB
	if maxMB <= 0 {
		maxMB = 2
	}

	// Google Drive uploader (optional - skip if no credentials)
	var uploader storage.FileUploader
	if len(cfg.GDriveCredentials) > 0 {
		ctx := context.Background()
		gdu, err := gdrive.NewFileUploader(ctx, gdrive.Config{
			CredentialsJSON: cfg.GDriveCredentials,
			FolderID:        cfg.GDrive.FolderID,
			MaxSizeMB:       maxMB,
		})
		if err != nil {
			log.Fatalf("gdrive uploader: %v", err)
		}
		uploader = gdu
	} else {
		uploader = &noopUploader{}
	}

	playerRepo := postgres.NewPlayerRepository(db)
	createUC := player.NewCreateUseCase(playerRepo, uploader)
	listUC := player.NewListUseCase(playerRepo)
	uploadUC := uploaduc.NewUploadUseCase(uploader, maxMB)

	ph := ginhandler.NewPlayerHandler(createUC, listUC)
	uh := ginhandler.NewUploadHandler(uploadUC)

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), corsMiddleware())

	api := r.Group("/api/v1")
	{
		api.POST("/upload", uh.Upload)
		api.POST("/players", ph.Create)
		api.GET("/players", ph.List)
	}

	addr := ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
