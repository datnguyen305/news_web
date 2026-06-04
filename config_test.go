package main

import (
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/news")
	t.Setenv("AI_SERVICE_URL", "")
	t.Setenv("SCRAP_URL", "")
	t.Setenv("PORT", "")
	t.Setenv("SCRAPER_INTERVAL", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.ScraperInterval != time.Hour {
		t.Fatalf("ScraperInterval = %v, want %v", cfg.ScraperInterval, time.Hour)
	}
}

func TestLoadConfigRequiresDBURL(t *testing.T) {
	t.Setenv("DB_URL", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfigParsesCustomInterval(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/news")
	t.Setenv("PORT", "9090")
	t.Setenv("SCRAPER_INTERVAL", "15m")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.ScraperInterval != 15*time.Minute {
		t.Fatalf("ScraperInterval = %v, want 15m", cfg.ScraperInterval)
	}
}
