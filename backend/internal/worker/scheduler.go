package worker

import (
	"log"
	"time"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Cron     *cron.Cron
	Strat    *service.Strategy
	Location *time.Location
}

func NewScheduler(strat *service.Strategy) *Scheduler {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal("Failed to load America/New_York location:", err)
	}
	c := cron.New(cron.WithLocation(loc))
	return &Scheduler{Cron: c, Strat: strat, Location: loc}
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

	// US Market Close: 16:00 ET
	// LOC orders should be placed before close (e.g., 15:50 ET)
	// Schedule: Mon-Fri 15:50 ET

	entryID, err := s.Cron.AddFunc("50 15 * * 1-5", func() {
		execTime := time.Now().In(s.Location)
		log.Println("========================================")
		log.Printf("[STRATEGY] ▶ Starting Daily Strategy Execution at %s", execTime.Format("2006-01-02 15:04:05 MST"))
		log.Println("========================================")

		s.Strat.ExecuteDaily()

		endTime := time.Now().In(s.Location)
		log.Println("========================================")
		log.Printf("[STRATEGY] ✓ Daily Strategy Execution Completed at %s (Duration: %v)", endTime.Format("2006-01-02 15:04:05 MST"), endTime.Sub(execTime))
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
	log.Printf("[SCHEDULER] Schedule: 15:50 ET (Mon-Fri)")
	log.Printf("[SCHEDULER] Next scheduled execution: %s", nextRun.In(s.Location).Format("2006-01-02 15:04:05 MST (Mon)"))
	log.Printf("[SCHEDULER] Time until next execution: %v", time.Until(nextRun).Round(time.Second))
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
