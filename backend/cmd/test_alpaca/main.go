package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/market"
)

func main() {
	// Try loading from parent dir (project root) where .env usually resides
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Note: .env not found in ../.env, checking current dir")
		godotenv.Load()
	}

	cfg := config.Load()
	if cfg.AlpacaApiKey == "" {
		log.Fatal("Error: ALPACA_API_KEY is missing. Please ensure .env is set.")
	}

	log.Println("Initializing Alpaca Client...")
	client := market.NewAlpacaClient(cfg)
	if client == nil {
		log.Fatal("Failed to create Alpaca client")
	}

	// Symbols to test (Correction validation)
	symbols := []string{"BRK.A", "BRK.B"}

	// Test range: Yesterday
	end := time.Now().UTC()
	start := end.Add(-48 * time.Hour) // Look back 48h to ensure data (weekends etc)

	log.Printf("Testing FetchBars for %v...", symbols)
	candles, err := client.FetchBars(symbols, start, end)
	if err != nil {
		log.Fatalf("❌ Test FAILED: %v", err)
	}

	log.Printf("✅ Test SUCCESS: Retrieved %d candles", len(candles))
	foundSymbols := make(map[string]bool)
	for _, c := range candles {
		foundSymbols[c.Symbol] = true
	}
	log.Printf("Found data for: %v", foundSymbols)
}
