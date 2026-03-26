package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

// NewAppConfig loads configuration from .env and environment variables.
func NewAppConfig() (*App, error) {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("warning: could not load .env:", err)
	}

	var cfg App
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	return &cfg, nil
}
