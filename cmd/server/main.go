package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m161awm2/velocity-bulletinBE/internal/auth"
	"github.com/m161awm2/velocity-bulletinBE/internal/config"
	"github.com/m161awm2/velocity-bulletinBE/internal/database"
	"github.com/m161awm2/velocity-bulletinBE/internal/httpapi"
	"github.com/m161awm2/velocity-bulletinBE/internal/service"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
	"github.com/m161awm2/velocity-bulletinBE/internal/upload"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg)
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	st := store.New(db)
	svc := service.New(st)
	uploads, err := upload.New(ctx, cfg.AWSRegion, cfg.S3Bucket, cfg.S3PublicBaseURL)
	if err != nil {
		logger.Error("upload service startup failed", "error", err)
		os.Exit(1)
	}
	router := httpapi.New(svc, st, auth.New(cfg.JWTSecret, cfg.JWTTTL), uploads, logger, cfg.CORSOrigins)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server starting", "config", cfg.String())
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-shutdownSignal.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
