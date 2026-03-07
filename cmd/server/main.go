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
	"github.com/dreamers-be/internal/adapter/storage/s3"
	appconfig "github.com/dreamers-be/internal/config"
	"github.com/dreamers-be/internal/domain/storage"
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
	} else {
		uploader = &noopUploader{}
	}

	playerRepo := postgres.NewPlayerRepository(db)
	createUC := player.NewCreateUseCase(playerRepo, uploader)
	listUC := player.NewListUseCase(playerRepo)
	getUC := player.NewGetUseCase(playerRepo)
	uploadUC := uploaduc.NewUploadUseCase(uploader, maxMB)

	var presigner storage.Presigner
	if p, ok := uploader.(storage.Presigner); ok {
		presigner = p
	}
	ph := ginhandler.NewPlayerHandler(createUC, listUC, getUC, presigner)
	uh := ginhandler.NewUploadHandler(uploadUC, presigner)

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), corsMiddleware())

	api := r.Group("/v1")
	{
		api.POST("/upload", uh.Upload)
		api.POST("/players", ph.Create)
		api.GET("/players", ginhandler.BasicAuth(ginhandler.BasicAuthCredentials), ph.List)
		api.GET("/players/:id", ginhandler.BasicAuth(ginhandler.BasicAuthCredentials), ph.Get)
	}

	addr := ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
