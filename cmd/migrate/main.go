package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/m161awm2/velocity-bulletinBE/internal/config"
)

func main() {
	action := flag.String("action", "up", "migration action: up or down")
	steps := flag.Int("steps", 0, "number of migrations; zero means all")
	flag.Parse()
	cfg, err := config.LoadDatabase()
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.New("file://migrations", cfg.MigrationURL())
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _, _ = m.Close() }()
	switch *action {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps <= 0 {
			log.Fatal("-steps must be positive for down migrations")
		}
		err = m.Steps(-*steps)
	default:
		fmt.Fprintln(os.Stderr, "action must be up or down")
		os.Exit(2)
	}
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
	log.Println("migration complete")
}
