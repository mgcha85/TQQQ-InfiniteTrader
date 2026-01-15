package api

import (
	"log"
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

	// 1. Log request immediately
	log.Printf("[API] Backfill request received: start=%s, end=%s", start, end)

	// 2. Trigger async Goroutine
	go func() {
		log.Printf("[API] Starting Backfill Goroutine for range %s to %s...", start, end)
		if err := h.MarketSvc.Backfill(start, end); err != nil {
			log.Printf("[API] ✗ Backfill failed: %v", err)
		} else {
			log.Printf("[API] ✓ Backfill completed successfully")
		}
	}()

	// 3. Return 202
	c.JSON(http.StatusAccepted, gin.H{
		"status": "Backfill triggered",
		"start":  start,
		"end":    end,
		"note":   "Check server logs for progress (docker logs -f tqqq-backend)",
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
