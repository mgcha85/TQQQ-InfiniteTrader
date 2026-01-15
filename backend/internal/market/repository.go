package market

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

type MarketRepository struct {
	DB *sql.DB
}

func NewMarketRepository() (*MarketRepository, error) {
	// In-memory DuckDB, we will query parquet files directly from disk
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}
	return &MarketRepository{DB: db}, nil
}

func (r *MarketRepository) QueryCandles(symbol string, start, end time.Time) ([]Candle, error) {
	// Query Parquet files using DuckDB's read_parquet or Glob check
	// We scan all parquet files in data/market_data. DuckDB is smart enough to prune if partitioned?
	// Hive partitioning is supported.
	// Query: SELECT * FROM read_parquet('data/market_data/resolution=1min/**/*.parquet', hive_partitioning=true)
	// WHERE symbol = ? AND timestamp >= ? AND timestamp <= ?

	// Timestamp in Parquet is UnixMilli (int64).
	startMilli := start.UnixMilli()
	endMilli := end.UnixMilli()

	query := `
		SELECT symbol, epoch_ms(timestamp), open, high, low, close, volume
		FROM read_parquet('data/market_data/resolution=1min/**/*.parquet', hive_partitioning=true)
		WHERE symbol = ? AND epoch_ms(timestamp) >= ? AND epoch_ms(timestamp) <= ?
		ORDER BY timestamp ASC
	`

	rows, err := r.DB.Query(query, symbol, startMilli, endMilli)
	if err != nil {
		return nil, fmt.Errorf("duckdb query failed: %v", err)
	}
	defer rows.Close()

	var candles []Candle
	for rows.Next() {
		var c Candle
		if err := rows.Scan(&c.Symbol, &c.Timestamp, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, nil
}
