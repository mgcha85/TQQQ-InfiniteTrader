package worker

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/market"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Cron      *cron.Cron
	Strat     *service.Strategy
	MarketSvc *market.MarketDataService
	Location  *time.Location
}

func NewScheduler(strat *service.Strategy, marketSvc *market.MarketDataService) *Scheduler {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal("Failed to load America/New_York location:", err)
	}
	c := cron.New(cron.WithLocation(loc))
	return &Scheduler{Cron: c, Strat: strat, MarketSvc: marketSvc, Location: loc}
}

func (s *Scheduler) Start() {
	// Log startup information
	nowET := time.Now().In(s.Location)
	log.Println("========================================")
	log.Println("[SCHEDULER] Starting TQQQ-InfiniteTrader Scheduler")
	log.Printf("[SCHEDULER] Current Time (ET): %s", nowET.Format("2006-01-02 15:04:05 MST"))
	log.Printf("[SCHEDULER] Current Time (UTC): %s", time.Now().UTC().Format("2006-01-02 15:04:05 MST"))
	log.Printf("[SCHEDULER] Current Time (Local): %s", time.Now().Format("2006-01-02 15:04:05 MST"))
	log.Println("========================================")

	// 1. Daily Market Data Sync (Backfill)
	// Schedule: 20:00 ET (8 PM) - Mon-Fri
	// This ensures we download today's closing data.
	marketDataSpec := "0 20 * * 1-5"
	_, err := s.Cron.AddFunc(marketDataSpec, func() {
		execTime := time.Now().In(s.Location)
		dateStr := execTime.Format("2006-01-02")
		log.Println("========================================")
		log.Printf("[MARKET] ▶ Starting Daily Market Data Sync for %s (20:00 ET)", dateStr)
		log.Println("========================================")

		// Download just today's data (start=today, end=today)
		if err := s.MarketSvc.Backfill(dateStr, dateStr); err != nil {
			log.Printf("[MARKET] ✗ Daily Sync Failed: %v", err)
		} else {
			log.Printf("[MARKET] ✓ Daily Sync Completed for %s", dateStr)
		}
	})
	if err != nil {
		log.Printf("[SCHEDULER] ⚠ Failed to register Market Data job: %v", err)
	} else {
		log.Printf("[SCHEDULER] Registered Daily Market Data Sync at 20:00 ET (Mon-Fri)")
	}

	// 2. Monthly Rebalancing Schedule: 26th of every month
	// Time: Configured via SCHEDULE_TIME (default 15:50 ET)
	scheduleTime := s.Strat.Client.Config.ScheduleTime
	parts := strings.Split(scheduleTime, ":")
	if len(parts) != 2 {
		log.Printf("[SCHEDULER] ⚠ Invalid SCHEDULE_TIME format (%s), defaulting to 15:50", scheduleTime)
		parts = []string{"15", "50"}
	}
	hour := parts[0]
	min := parts[1]
	// Cron: Min Hour Dom Month Dow
	cronSpec := fmt.Sprintf("%s %s 26 * *", min, hour)

	log.Printf("[SCHEDULER] Schedule registered: %s ET on 26th (Cron: %s)", scheduleTime, cronSpec)

	entryID, err := s.Cron.AddFunc(cronSpec, func() {
		execTime := time.Now().In(s.Location)
		log.Println("========================================")
		log.Printf("[STRATEGY] ▶ Starting Monthly Rebalance Execution at %s", execTime.Format("2006-01-02 15:04:05 MST"))
		log.Println("========================================")

		if err := s.Strat.ExecuteRebalance(false); err != nil {
			log.Printf("[STRATEGY] ✗ Rebalance Execution Failed: %v", err)
		}

		endTime := time.Now().In(s.Location)
		log.Println("========================================")
		log.Printf("[STRATEGY] ✓ Monthly Rebalance Execution Completed at %s (Duration: %v)", endTime.Format("2006-01-02 15:04:05 MST"), endTime.Sub(execTime))
		log.Println("========================================")
	})
	if err != nil {
		log.Fatal("Error adding cron job:", err)
	}

	s.Cron.Start()

	// Calculate and log next scheduled execution
	entry := s.Cron.Entry(entryID)
	nextRun := entry.Next
	log.Printf("[SCHEDULER] ✓ Scheduler started successfully")
	log.Printf("[SCHEDULER] Next Rebalance: %s", nextRun.In(s.Location).Format("2006-01-02 15:04:05 MST"))
	log.Printf("[SCHEDULER] Time until rebalance: %v", time.Until(nextRun).Round(time.Second))
	log.Println("========================================")

	// Start heartbeat goroutine to log status every 30 minutes
	go s.heartbeat(entryID)
}

// heartbeat logs the scheduler status periodically
func (s *Scheduler) heartbeat(entryID cron.EntryID) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		entry := s.Cron.Entry(entryID)
		nextRun := entry.Next
		nowET := time.Now().In(s.Location)

		log.Printf("[SCHEDULER] 💓 Heartbeat - Current Time (ET): %s | Next Execution: %s | Time Until: %v",
			nowET.Format("15:04:05"),
			nextRun.In(s.Location).Format("2006-01-02 15:04:05"),
			time.Until(nextRun).Round(time.Second))
	}
}
