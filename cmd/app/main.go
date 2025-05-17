package main

import (
	"log"

	_ "github.com/joho/godotenv/autoload"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
