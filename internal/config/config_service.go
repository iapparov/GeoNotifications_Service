package config

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

func NewAppConfig() (*App, error) {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("warning: could not load .env:", err)
	}

	var cfg App
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse env: %v", err)
	}

	return &cfg, nil
}
