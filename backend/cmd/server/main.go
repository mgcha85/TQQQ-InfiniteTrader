package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/api"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/kis"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/market"
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

	// 3.1 Market Data (Alpaca + DuckDB)
	alpacaClient := market.NewAlpacaClient(cfg)
	marketSvc := market.NewMarketDataService(cfg, alpacaClient)
	marketRepo, err := market.NewMarketRepository()
	if err != nil {
		log.Printf("⚠ MarketRepository (DuckDB) init failed: %v", err)
	}

	// 5. Handler
	handler := api.NewHandler(db, strat, marketSvc, marketRepo)

	// 6. Scheduler
	scheduler := worker.NewScheduler(strat, marketSvc)
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
		log.Println("[STARTUP] ----------------------------------------")

		// Cash balance inquiry
		cashBalance, cashErr := client.GetBuyingPower()
		if cashErr != nil {
			log.Printf("[STARTUP] Cash Balance: (조회 실패: %v)", cashErr)
		} else {
			log.Printf("[STARTUP] 💵 Available Cash: $%s", cashBalance.Output.OvrsOrdPsblAmt)
		}

		log.Printf("[STARTUP] Total Invested: $%s", balance.Output2.TotalPurchase)
		log.Printf("[STARTUP] Total Evaluation: $%s", balance.Output2.TotalAmt)
		log.Printf("[STARTUP] Total P/L: $%s (%s%%)", balance.Output2.TotalPL, balance.Output2.TotalPLRate)
		log.Printf("[STARTUP] Realized P/L: $%s", balance.Output2.RealizedPL)
		log.Println("[STARTUP] ----------------------------------------")
		log.Printf("[STARTUP] Holdings: %d", len(balance.Output1))
		for i, h := range balance.Output1 {
			log.Printf("[STARTUP]   [%d] %s:%s - Qty: %s, AvgPrice: $%s, Now: $%s",
				i+1, h.ExchCode, h.Symbol, h.Qty, h.AvgPrice, h.NowPrice)
		}
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

		// Rebalance API
		v1.GET("/rebalance/preview", handler.GetRebalancePreview)
		v1.POST("/rebalance/execute", handler.ExecuteRebalance)

		// Market Data API
		v1.POST("/market/backfill", handler.Backfill)
		v1.GET("/market/candles", handler.GetCandles)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
