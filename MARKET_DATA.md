# Market Data Service (Alpaca + DuckDB)

This service fetches 1-minute candle data from Alpaca API (Free Tier), stores it in Parquet format with Hive Partitioning, and serves it via DuckDB.

## 🏗 Architecture

1.  **Collector**: Fetches 1-min candles for 66 symbols using Alpaca API.
2.  **Storage**: Parquet files stored in `data/market_data/resolution=1min/year=YYYY/month=MM/data_YYYYMMDD.parquet`.
3.  **Serving**: DuckDB executes SQL queries directly on Parquet files for high performance.

## ⚙️ Configuration

Start by adding your Alpaca API credentials to `.env`:

```env
ALPACA_API_KEY=your_key
ALPACA_SECRET_KEY=your_secret
```

Symbols are configured in `backend/config/symbols.json`.

## 🚀 API Usage

### 1. Backfill Data (Download)
Trigger a download for a specific date range.

```bash
POST /api/market/backfill?start=YYYY-MM-DD&end=YYYY-MM-DD
```

**Example:**
```bash
curl -X POST "http://localhost:8082/api/market/backfill?start=2024-01-01&end=2024-01-31"
```

### 2. Query Candles (Serving)
Retrieve candle data for a symbol.

```bash
GET /api/market/candles?symbol=TQQQ&start=YYYY-MM-DD&end=YYYY-MM-DD
```

**Example:**
```bash
curl "http://localhost:8082/api/market/candles?symbol=AAPL&start=2024-01-01&end=2024-01-02" | jq
```

## ⏰ Daily Cron Setup

To backfill yesterday's data every day at 02:00 AM, add this to your crontab:

```bash
0 2 * * * curl -X POST "http://localhost:8082/api/market/backfill?start=$(date -d 'yesterday' +\%Y-\%m-\%d)&end=$(date -d 'yesterday' +\%Y-\%m-\%d)" >> /var/log/market_backfill.log 2>&1
```

## 🛠 Troubleshooting

- **429 Rate Limit**: The service automatically sleeps (300ms) between day loops, but Alpaca Free Tier has a limit (200/min). If you hit limits, try smaller ranges or increase the sleep duration in `service.go`.
- **Missing Data**: Free Tier uses IEX feed. Some low-volume symbols might have gaps.
