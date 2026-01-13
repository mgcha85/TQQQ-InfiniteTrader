package service

import (
	"log"
	"math"
	"strconv"
	"time"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/kis"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/model"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/repository"
	"gorm.io/gorm"
)

type Strategy struct {
	DB     *repository.DB
	Client *kis.Client
}

func NewStrategy(db *repository.DB, client *kis.Client) *Strategy {
	return &Strategy{DB: db, Client: client}
}

// logWithTime logs a message with timestamp
func logWithTime(format string, v ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[%s] "+format, append([]interface{}{timestamp}, v...)...)
}

// SyncState updates local DB with real portfolio status
func (s *Strategy) SyncState() error {
	logWithTime("[SYNC] Starting portfolio sync with KIS API...")

	bal, err := s.Client.GetBalance()
	if err != nil {
		logWithTime("[SYNC] ✗ Failed to get balance: %v", err)
		return err
	}

	logWithTime("[SYNC] Received %d holdings from KIS", len(bal.Output1))

	// For each holding, update CycleStatus
	for _, holding := range bal.Output1 {
		qty, _ := strconv.Atoi(holding.Qty)
		avgPrice, _ := strconv.ParseFloat(holding.AvgPrice, 64)

		var cycle model.CycleStatus
		res := s.DB.Where("symbol = ?", holding.Symbol).First(&cycle)

		if res.Error == gorm.ErrRecordNotFound {
			cycle = model.CycleStatus{
				Symbol: holding.Symbol,
			}
			logWithTime("[SYNC] Creating new CycleStatus for %s", holding.Symbol)
		}

		// Update
		cycle.TotalBoughtQty = qty
		cycle.AvgPrice = avgPrice
		cycle.TotalInvested = float64(qty) * avgPrice

		// Logic to reset day if 0
		if qty == 0 {
			cycle.CurrentCycleDay = 0
		}

		s.DB.Save(&cycle)
		logWithTime("[SYNC] Updated %s: Qty=%d, AvgPrice=$%.2f, TotalInvested=$%.2f",
			holding.Symbol, qty, avgPrice, cycle.TotalInvested)
	}

	logWithTime("[SYNC] ✓ Portfolio sync completed")
	return nil
}

// ExecuteDaily runs the strategy for a single day
func (s *Strategy) ExecuteDaily() {
	logWithTime("[EXECUTE] ========================================")
	logWithTime("[EXECUTE] Starting ExecuteDaily...")

	// 0. Force Refresh Token to avoid expiration issues during execution
	if err := s.Client.ForceRefresh(); err != nil {
		logWithTime("[EXECUTE] ⚠ Failed to refresh token: %v (trying to proceed anyway)", err)
	}

	// 1. Load Settings
	var settings model.UserSettings
	if err := s.DB.First(&settings).Error; err != nil {
		logWithTime("[EXECUTE] ✗ No settings found in database, skipping strategy")
		return
	}

	logWithTime("[EXECUTE] Settings loaded - IsActive: %v, Principal: $%.2f, SplitCount: %d, TargetRate: %.2f%%",
		settings.IsActive, settings.Principal, settings.SplitCount, settings.TargetRate*100)

	if !settings.IsActive {
		logWithTime("[EXECUTE] ⚠ Strategy is INACTIVE, skipping execution")
		return
	}

	// Sync State First
	logWithTime("[EXECUTE] Step 1: Syncing portfolio state...")
	if err := s.SyncState(); err != nil {
		logWithTime("[EXECUTE] ✗ Sync State Failed: %v", err)
		return
	}

	symbols := []string{"TQQQ"} // Or parse settings.Symbols
	logWithTime("[EXECUTE] Step 2: Processing symbols: %v", symbols)

	for _, sym := range symbols {
		s.processSymbol(sym, settings)
	}

	logWithTime("[EXECUTE] ========================================")
}

func (s *Strategy) processSymbol(sym string, settings model.UserSettings) {
	logWithTime("[%s] ----------------------------------------", sym)
	logWithTime("[%s] Processing symbol...", sym)

	var cycle model.CycleStatus
	if err := s.DB.Where("symbol = ?", sym).First(&cycle).Error; err != nil {
		// Init if missing
		cycle = model.CycleStatus{Symbol: sym, CurrentCycleDay: 0}
		logWithTime("[%s] No existing cycle found, initializing new cycle (Day 0)", sym)
	} else {
		logWithTime("[%s] Existing cycle: Day %d, TotalQty=%d, AvgPrice=$%.2f, TotalInvested=$%.2f",
			sym, cycle.CurrentCycleDay, cycle.TotalBoughtQty, cycle.AvgPrice, cycle.TotalInvested)
	}

	// 2. Buy Logic
	// Check if already bought today
	today := time.Now().Truncate(24 * time.Hour)
	var existingLog model.TradeLog
	if err := s.DB.Where("symbol = ? AND side = 'BUY' AND created_at >= ?", sym, today).First(&existingLog).Error; err == nil {
		logWithTime("[%s] ⚠ Already bought today (Qty: %d at $%.2f), skipping buy.", sym, existingLog.Qty, existingLog.Price)
		return
	}

	// Amount per buy = Principal / 40
	unitAmount := settings.Principal / float64(settings.SplitCount)
	logWithTime("[%s] Unit amount per buy: $%.2f (Principal $%.2f / %d splits)",
		sym, unitAmount, settings.Principal, settings.SplitCount)

	// Check Price
	logWithTime("[%s] Fetching current price from KIS API...", sym)
	price, err := s.Client.GetCurrentPrice("NAS", sym) // Assuming NAS
	if err != nil {
		logWithTime("[%s] ✗ Price fetch failed: %v", sym, err)
		return
	}
	logWithTime("[%s] ✓ Current price: $%.2f", sym, price)

	// Calculate Qty to buy
	// If unitAmount $250, Price $50 -> 5 shares
	buyQty := int(math.Floor(unitAmount / price))
	if buyQty < 1 {
		buyQty = 1 // Min 1
	}
	logWithTime("[%s] Calculated buy quantity: %d shares ($%.2f / $%.2f)", sym, buyQty, unitAmount, price)

	// Place Buy Order (LOC - Limit On Close, simulated as Limit Order)
	// We use price * 1.05 to ensure fill for now, or just Limit at Price
	logWithTime("[%s] Placing BUY order: %d shares at $%.2f (Limit)...", sym, buyQty, price)
	buyErr := s.Client.PlaceOrder(kis.OrderReq{
		ExchCode: "NASD",
		Symbol:   sym,
		Qty:      buyQty,
		Price:    price, // Use current price
		OrdType:  "00",  // Limit
		Side:     "BUY",
	})

	if buyErr != nil {
		logWithTime("[%s] ✗ Buy order FAILED: %v", sym, buyErr)
	} else {
		logWithTime("[%s] ✓ Buy order PLACED: %d shares at $%.2f (Total: $%.2f)", sym, buyQty, price, float64(buyQty)*price)
		// Update Cycle Day (Optimistic, sync will fix later)
		cycle.CurrentCycleDay++
		s.DB.Save(&cycle)
		logWithTime("[%s] Cycle day updated to: %d", sym, cycle.CurrentCycleDay)

		// Record Log
		s.DB.Create(&model.TradeLog{
			Date:   time.Now(),
			Symbol: sym,
			Side:   "BUY",
			Qty:    buyQty,
			Price:  price,
			Amount: float64(buyQty) * price,
		})
		logWithTime("[%s] Trade log recorded in database", sym)
	}

	// 3. Sell Logic
	// If we have shares, place Sell Limit at AvgPrice * 1.10
	if cycle.TotalBoughtQty > 0 || buyErr == nil {
		logWithTime("[%s] ----------------------------------------", sym)
		logWithTime("[%s] Processing SELL order...", sym)

		// Total shares = Current Locked + New Buy (approx)
		// Note: We can't sell "New Buy" immediately if not settled?
		// Actually KIS allows Day Trading. But safe to sell "Existing" + "New" if accepted.
		// For simplicity, we assume we sell what we think we have.

		totalQty := cycle.TotalBoughtQty + buyQty // Valid assumption for ordering

		// Target Price
		// AvgPrice might change after Buy.
		// Estimated New Avg = (OldTotalVal + NewVal) / TotalQty
		oldVal := cycle.TotalInvested
		newVal := float64(buyQty) * price
		estAvg := (oldVal + newVal) / float64(totalQty)

		targetPrice := estAvg * (1 + settings.TargetRate)

		logWithTime("[%s] Sell calculation: TotalQty=%d, OldInvested=$%.2f, NewInvested=$%.2f",
			sym, totalQty, oldVal, newVal)
		logWithTime("[%s] Estimated Avg: $%.2f, Target Rate: %.2f%%, Target Price: $%.2f",
			sym, estAvg, settings.TargetRate*100, targetPrice)

		logWithTime("[%s] Placing SELL order: %d shares at $%.2f (Limit)...", sym, totalQty, targetPrice)
		sellErr := s.Client.PlaceOrder(kis.OrderReq{
			ExchCode: "NASD",
			Symbol:   sym,
			Qty:      totalQty,
			Price:    targetPrice,
			OrdType:  "00", // Limit
			Side:     "SELL",
		})

		if sellErr != nil {
			logWithTime("[%s] ✗ Sell order FAILED: %v", sym, sellErr)
		} else {
			logWithTime("[%s] ✓ Sell order PLACED: %d shares at Target $%.2f", sym, totalQty, targetPrice)
		}
	}

	logWithTime("[%s] ----------------------------------------", sym)
}
