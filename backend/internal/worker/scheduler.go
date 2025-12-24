package worker

import (
	"log"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Cron  *cron.Cron
	Strat *service.Strategy
}

func NewScheduler(strat *service.Strategy) *Scheduler {
	c := cron.New()
	return &Scheduler{Cron: c, Strat: strat}
}

func (s *Scheduler) Start() {
	// Schedule for 23:50 KST (approx market open/close timing needs checking)
	// User can configure this. For now, daily at a fixed time.
	// US Market Open is 09:30 ET. Close 16:00 ET.
	// LOC orders should be placed before close (e.g., 15:50 ET).
	// 15:50 ET = 05:50 KST (Next Day).
	// Let's set it to run every 10 minutes to verify "Current Time vs Market Time" or just trigger
	// Strategy logic usually guards against duplicate buys.
	// But let's set a fixed time: "0 5 * * 2-6" (05:00 AM Tue-Sat KST => 15:00 PM Mon-Fri ET approx)

	_, err := s.Cron.AddFunc("0 5 * * 2-6", func() {
		log.Println("Running Daily Strategy...")
		s.Strat.ExecuteDaily()
	})
	if err != nil {
		log.Fatal("Error adding cron job:", err)
	}

	s.Cron.Start()
	log.Println("Scheduler started (05:00 KST Tue-Sat)")
}
