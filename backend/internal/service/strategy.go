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

// SyncState updates local DB with real portfolio status
func (s *Strategy) SyncState() error {
	bal, err := s.Client.GetBalance()
	if err != nil {
		return err
	}

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
	}
	return nil
}

// ExecuteDaily runs the strategy for a single day
func (s *Strategy) ExecuteDaily() {
	// 1. Load Settings
	var settings model.UserSettings
	if err := s.DB.First(&settings).Error; err != nil {
		log.Println("No settings found, skipping strategy")
		return
	}
	if !settings.IsActive {
		return
	}

	// Sync State First
	if err := s.SyncState(); err != nil {
		log.Printf("Sync State Failed: %v", err)
		return
	}

	symbols := []string{"TQQQ"} // Or parse settings.Symbols

	for _, sym := range symbols {
		s.processSymbol(sym, settings)
	}
}

func (s *Strategy) processSymbol(sym string, settings model.UserSettings) {
	var cycle model.CycleStatus
	if err := s.DB.Where("symbol = ?", sym).First(&cycle).Error; err != nil {
		// Init if missing
		cycle = model.CycleStatus{Symbol: sym, CurrentCycleDay: 0}
	}

	// 2. Buy Logic
	// Check if already bought today
	today := time.Now().Truncate(24 * time.Hour)
	var existingLog model.TradeLog
	if err := s.DB.Where("symbol = ? AND side = 'BUY' AND created_at >= ?", sym, today).First(&existingLog).Error; err == nil {
		log.Printf("[%s] Already bought today (Qty: %d), skipping buy.", sym, existingLog.Qty)
		return
	}
	// Amount per buy = Principal / 40
	unitAmount := settings.Principal / float64(settings.SplitCount)

	// Check Price
	price, err := s.Client.GetCurrentPrice("NAS", sym) // Assuming NAS
	if err != nil {
		log.Printf("[%s] Price fetch failed: %v", sym, err)
		return
	}

	// Calculate Qty to buy
	// If unitAmount $250, Price $50 -> 5 shares
	buyQty := int(math.Floor(unitAmount / price))
	if buyQty < 1 {
		buyQty = 1 // Min 1
	}

	// Place Buy Order (LOC - Limit On Close, simulated as Limit Order)
	// We use price * 1.05 to ensure fill for now, or just Limit at Price
	buyErr := s.Client.PlaceOrder(kis.OrderReq{
		ExchCode: "NAS",
		Symbol:   sym,
		Qty:      buyQty,
		Price:    price, // Use current price
		OrdType:  "00",  // Limit
		Side:     "BUY",
	})

	if buyErr != nil {
		log.Printf("[%s] Buy failed: %v", sym, buyErr)
	} else {
		log.Printf("[%s] Buy Order Placed: %d shares at $%.2f", sym, buyQty, price)
		// Update Cycle Day (Optimistic, sync will fix later)
		cycle.CurrentCycleDay++
		s.DB.Save(&cycle)

		// Record Log
		s.DB.Create(&model.TradeLog{
			Date:   time.Now(),
			Symbol: sym,
			Side:   "BUY",
			Qty:    buyQty,
			Price:  price,
			Amount: float64(buyQty) * price,
		})
	}

	// 3. Sell Logic
	// If we have shares, place Sell Limit at AvgPrice * 1.10
	if cycle.TotalBoughtQty > 0 || buyErr == nil {
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

		sellErr := s.Client.PlaceOrder(kis.OrderReq{
			ExchCode: "NAS",
			Symbol:   sym,
			Qty:      totalQty,
			Price:    targetPrice,
			OrdType:  "00", // Limit
			Side:     "SELL",
		})

		if sellErr != nil {
			log.Printf("[%s] Sell Order Failed: %v", sym, sellErr)
		} else {
			log.Printf("[%s] Sell Order Placed: %d shares at Target $%.2f", sym, totalQty, targetPrice)
		}
	}
}
