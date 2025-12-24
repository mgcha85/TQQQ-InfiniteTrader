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
