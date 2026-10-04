package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m161awm2/velocity-bulletinBE/internal/auth"
	"github.com/m161awm2/velocity-bulletinBE/internal/cache"
	"github.com/m161awm2/velocity-bulletinBE/internal/config"
	"github.com/m161awm2/velocity-bulletinBE/internal/database"
	"github.com/m161awm2/velocity-bulletinBE/internal/httpapi"
	"github.com/m161awm2/velocity-bulletinBE/internal/service"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	cfg, err := config.Load()
	if err != nil {
		logger.Println("invalid configuration:", err)
		os.Exit(1)
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg)
	if err != nil {
		logger.Println("database startup failed:", err)
		os.Exit(1)
	}
	st := store.New(db)
	redisCache, err := cache.NewRedis(cfg.RedisURL)
	if err != nil {
		logger.Println("invalid Redis configuration; cache disabled:", err)
	}
	var responseCache cache.Cache
	if redisCache != nil {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := redisCache.Ping(pingCtx); err != nil {
			logger.Println("Redis unavailable; requests will use PostgreSQL until Redis recovers:", err)
		} else {
			logger.Println("Redis cache connected")
		}
		cancel()
		defer redisCache.Close()
		responseCache = redisCache
	}
	svc := service.NewWithCache(st, responseCache, 30*time.Second)
	router := httpapi.New(svc, st, auth.New(cfg.JWTSecret, cfg.JWTTTL), logger, cfg.CORSOrigins)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Println("server starting", "config", cfg.String())
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-shutdownSignal.Done():
		logger.Println("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Println("server stopped unexpectedly:", err)
			os.Exit(1)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Println("graceful shutdown failed:", err)
		os.Exit(1)
	}
	logger.Println("server stopped")
}
