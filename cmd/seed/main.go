package main

import (
	"context"
	"log"
	"os"

	"github.com/m161awm2/velocity-bulletinBE/internal/config"
	"github.com/m161awm2/velocity-bulletinBE/internal/database"
	"github.com/m161awm2/velocity-bulletinBE/internal/service"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
)

func main() {
	cfg, err := config.LoadDatabase()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		log.Fatal("ADMIN_EMAIL and ADMIN_PASSWORD are required")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	admin, err := service.New(store.New(db)).SeedAdmin(ctx, cfg.AdminEmail, "Administrator", cfg.AdminPassword)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("admin ready: %s (%s)\n", admin.Email, admin.ID)
	_ = os.Stdout.Sync()
}
