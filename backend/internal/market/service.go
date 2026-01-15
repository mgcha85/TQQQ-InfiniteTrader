package market

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
	"github.com/parquet-go/parquet-go"
)

type MarketDataService struct {
	Config *config.Config
	Alpaca *AlpacaClient
}

func NewMarketDataService(cfg *config.Config, alpaca *AlpacaClient) *MarketDataService {
	return &MarketDataService{
		Config: cfg,
		Alpaca: alpaca,
	}
}

// Backfill downloads data for the given range and saves to Parquet
// Date format: "2006-01-02"
func (s *MarketDataService) Backfill(startDateStr, endDateStr string) error {
	log.Printf("[MARKET] Starting Backfill task: %s to %s", startDateStr, endDateStr)
	if s.Alpaca == nil {
		log.Printf("[MARKET] ✗ Backfill aborted: Alpaca API client not initialized (Check ALPACA_API_KEY in .env)")
		return fmt.Errorf("alpaca API client not initialized")
	}

	// 1. Load Symbols
	symbols, err := s.loadSymbols()
	if err != nil {
		return fmt.Errorf("failed to load symbols: %v", err)
	}
	log.Printf("[MARKET] Loaded %d symbols for backfill", len(symbols))

	// 2. Parse Dates
	start, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return fmt.Errorf("invalid start date: %v", err)
	}
	end, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return fmt.Errorf("invalid end date: %v", err)
	}

	// 3. Loop Dates
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		// Skip weekends if desired? Alpaca returns empty for weekends anyway.
		// Crypto? No, symbols are stocks.
		// Check weekday to save API calls?
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		log.Printf("[MARKET] Processing %s...", d.Format("2006-01-02"))

		// End of the day (Alpaca expects time range)
		// We ask for the full day 00:00 to 23:59 (UTC? Alpaca defaults to feed logic. SIP is 9:30-16:00 usually but extended hours exist)
		// We'll ask for full UTC day.
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := dayStart.Add(24 * time.Hour)

		// Fetch
		candles, err := s.Alpaca.FetchBars(symbols, dayStart, dayEnd)
		if err != nil {
			log.Printf("[MARKET] ✗ Failed to fetch %s: %v", d.Format("2006-01-02"), err)
			continue
		}

		if len(candles) == 0 {
			log.Printf("[MARKET] ⚠ No data for %s", d.Format("2006-01-02"))
			continue
		}

		// Write Parquet
		if err := s.writeParquet(candles, d); err != nil {
			log.Printf("[MARKET] ✗ Failed to write parquet %s: %v", d.Format("2006-01-02"), err)
			continue
		}

		log.Printf("[MARKET] ✓ Saved %d candles for %s", len(candles), d.Format("2006-01-02"))

		// Rate limit kindness
		time.Sleep(300 * time.Millisecond)
	}

	return nil
}

func (s *MarketDataService) loadSymbols() ([]string, error) {
	// Path relative to execution? Or absolute?
	// Assuming execution from project root or robust finding.
	// We'll try "backend/config/symbols.json" and "config/symbols.json"

	paths := []string{
		"backend/config/symbols.json",
		"config/symbols.json",
		"../config/symbols.json",
	}

	var content []byte
	var err error
	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	var symbols []string
	if err := json.Unmarshal(content, &symbols); err != nil {
		return nil, err
	}
	return symbols, nil
}

func (s *MarketDataService) writeParquet(candles []Candle, date time.Time) error {
	// Path: data/market_data/resolution=1min/year=YYYY/month=MM/data_YYYYMMDD.parquet
	year := date.Format("2006")
	month := date.Format("01")
	dateStr := date.Format("20060102")

	dir := filepath.Join("data", "market_data", "resolution=1min", fmt.Sprintf("year=%s", year), fmt.Sprintf("month=%s", month))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fpath := filepath.Join(dir, fmt.Sprintf("data_%s.parquet", dateStr))

	// Write
	if err := parquet.WriteFile(fpath, candles); err != nil {
		return err
	}
	return nil
}
