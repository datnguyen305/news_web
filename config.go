package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBURL           string
	AIServiceURL    string
	ScrapURL        string
	Port            string
	ScraperInterval time.Duration
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DBURL:           os.Getenv("DB_URL"),
		AIServiceURL:    os.Getenv("AI_SERVICE_URL"),
		ScrapURL:        os.Getenv("SCRAP_URL"),
		Port:            os.Getenv("PORT"),
		ScraperInterval: time.Hour,
	}

	if cfg.DBURL == "" {
		return Config{}, fmt.Errorf("DB_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("PORT must be a number: %w", err)
	}
	if interval := os.Getenv("SCRAPER_INTERVAL"); interval != "" {
		parsed, err := time.ParseDuration(interval)
		if err != nil {
			return Config{}, fmt.Errorf("SCRAPER_INTERVAL must be a duration: %w", err)
		}
		cfg.ScraperInterval = parsed
	}

	return cfg, nil
}
