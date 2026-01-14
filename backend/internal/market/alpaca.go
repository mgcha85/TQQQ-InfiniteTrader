package market

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
)

type AlpacaClient struct {
	Client *marketdata.Client
}

type Candle struct {
	Symbol    string  `parquet:"symbol,dict"`
	Timestamp int64   `parquet:"timestamp"` // Unix Milli
	Open      float64 `parquet:"open"`
	High      float64 `parquet:"high"`
	Low       float64 `parquet:"low"`
	Close     float64 `parquet:"close"`
	Volume    uint64  `parquet:"volume"`
}

func NewAlpacaClient(cfg *config.Config) *AlpacaClient {
	if cfg.AlpacaApiKey == "" || cfg.AlpacaSecret == "" {
		return nil
	}

	client := marketdata.NewClient(marketdata.ClientOpts{
		APIKey:    cfg.AlpacaApiKey,
		APISecret: cfg.AlpacaSecret,
		Feed:      marketdata.IEX, // Free Tier
	})

	return &AlpacaClient{Client: client}
}

// FetchBars fetches 1-min bars for multiple symbols within a time range
func (c *AlpacaClient) FetchBars(symbols []string, start, end time.Time) ([]Candle, error) {
	var candles []Candle

	// IEX feed is set in ClientOpts
	req := marketdata.GetBarsRequest{
		TimeFrame: marketdata.OneMin,
		Start:     start,
		End:       end,
	}

	// GetMultiBars handles pagination automatically
	barsMap, err := c.Client.GetMultiBars(symbols, req)
	if err != nil {
		return nil, fmt.Errorf("alpaca GetMultiBars failed: %v", err)
	}

	for symbol, bars := range barsMap {
		for _, b := range bars {
			candles = append(candles, Candle{
				Symbol:    symbol,
				Timestamp: b.Timestamp.UnixMilli(),
				Open:      b.Open,
				High:      b.High,
				Low:       b.Low,
				Close:     b.Close,
				Volume:    b.Volume,
			})
		}
	}

	return candles, nil
}
