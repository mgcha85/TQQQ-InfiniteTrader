package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Backfill API: POST /api/market/backfill?start=2024-01-01&end=2024-01-31
func (h *Handler) Backfill(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	if start == "" || end == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start and end dates required (YYYY-MM-DD)"})
		return
	}

	// Trigger async or sync? Backfill takes time.
	// For now, sync or goroutine?
	// User said "Start cron job... curl". Usually sync is okay if timeout is long, but long running requests often timeout.
	// Recommendation: Goroutine and return 202 accepted.
	// But simple implementation: Sync (or Goroutine).
	// Let's do Goroutine to avoid HTTP timeout.

	go func() {
		if err := h.MarketSvc.Backfill(start, end); err != nil {
			// Log error? h.MarketSvc logs internal errors.
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"status": "Backfill triggered",
		"start":  start,
		"end":    end,
	})
}

// GetCandles API: GET /api/market/candles?symbol=AAPL&start=2024-01-01&end=2024-01-02
func (h *Handler) GetCandles(c *gin.Context) {
	symbol := c.Query("symbol")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if symbol == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol, start, end required"})
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date"})
		return
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date"})
		return
	}

	candles, err := h.MarketRepo.QueryCandles(symbol, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"count":  len(candles),
		"data":   candles,
	})
}
