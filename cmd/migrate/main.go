package main

import (
	"context"
	"flag"
	"log"

	"github.com/m161awm2/velocity-bulletinBE/internal/config"
	"github.com/m161awm2/velocity-bulletinBE/internal/database"
)

func main() {
	// Preserve existing deployment invocations using -action up.
	action := flag.String("action", "up", "migration action: up (AutoMigrate only)")
	flag.Parse()
	if *action != "up" || flag.NArg() != 0 {
		log.Fatal("only AutoMigrate is supported; down/steps are no longer available")
	}
	cfg, err := config.LoadDatabase()
	if err != nil {
		log.Fatal(err)
	}
	cfg.DatabaseURL = cfg.MigrationURL()
	db, err := database.Open(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}
	log.Println("auto migration complete")
}
