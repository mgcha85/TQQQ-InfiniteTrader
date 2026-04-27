package api

import (
	"log"
	"net/http"

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
