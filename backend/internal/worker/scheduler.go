package worker

import (
	"log"
	"time"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Cron  *cron.Cron
	Strat *service.Strategy
}

func NewScheduler(strat *service.Strategy) *Scheduler {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal("Failed to load America/New_York location:", err)
	}
	c := cron.New(cron.WithLocation(loc))
	return &Scheduler{Cron: c, Strat: strat}
}

func (s *Scheduler) Start() {
	// US Market Close: 16:00 ET
	// LOC orders should be placed before close (e.g., 15:50 ET)
	// Schedule: Mon-Fri 15:50 ET

	_, err := s.Cron.AddFunc("50 15 * * 1-5", func() {
		log.Println("Running Daily Strategy...")
		s.Strat.ExecuteDaily()
	})
	if err != nil {
		log.Fatal("Error adding cron job:", err)
	}

	s.Cron.Start()
	log.Println("Scheduler started (15:50 ET Mon-Fri)")
}
