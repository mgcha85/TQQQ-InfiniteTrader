package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/api"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/kis"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/repository"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/worker"
)

func main() {
	// 1. Config
	cfg := config.Load()

	// 2. DB
	db, err := repository.NewDB("data/db.sqlite")
	if err != nil {
		log.Fatal("DB init failed:", err)
	}

	// 3. KIS Client
	client := kis.NewClient(cfg)

	// 4. Strategy
	strat := service.NewStrategy(db, client)

	// 5. Handler
	handler := api.NewHandler(db, strat)

	// 6. Scheduler
	scheduler := worker.NewScheduler(strat)
	scheduler.Start()

	// 7. Startup Balance Check (for debugging via docker logs)
	log.Println("========================================")
	log.Println("[STARTUP] Checking KIS API connection...")
	balance, err := client.GetBalance()
	if err != nil {
		log.Printf("[STARTUP] ✗ Balance check failed: %v", err)
	} else {
		log.Printf("[STARTUP] ✓ API connection successful!")
		log.Printf("[STARTUP] Account: %s", cfg.KisAccountNum)
		log.Printf("[STARTUP] Holdings: %d", len(balance.Output1))
		for i, h := range balance.Output1 {
			log.Printf("[STARTUP]   [%d] %s:%s - Qty: %s, AvgPrice: %s, CurrentPrice: %s",
				i+1, h.ExchCode, h.Symbol, h.Qty, h.AvgPrice, h.NowPrice)
		}
		log.Printf("[STARTUP] Total Evaluation: $%s", balance.Output2.TotalAmt)
	}
	log.Println("========================================")

	r := gin.Default()

	// API Group
	v1 := r.Group("/api")
	{
		v1.GET("/dashboard", handler.GetDashboard)
		v1.GET("/settings", handler.GetSettings)
		v1.POST("/settings", handler.UpdateSettings)
		v1.POST("/sync", handler.TriggerSync)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
