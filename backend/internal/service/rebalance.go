package service

import (
	"fmt"
	"math"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/kis"
)

// RebalancePlan holds the result of a rebalance calculation
type RebalancePlan struct {
	TotalValue    float64         `json:"total_value"`
	Cash          float64         `json:"cash"`
	Items         []RebalanceItem `json:"items"`
	EstimatedTax  float64         `json:"estimated_tax"`
	ActionSummary string          `json:"action_summary"`
}

type RebalanceItem struct {
	Symbol       string  `json:"symbol"`
	CurrentQty   int     `json:"current_qty"`
	CurrentPrice float64 `json:"current_price"`
	CurrentVal   float64 `json:"current_val"`
	CurrentWt    float64 `json:"current_wt"` // Percentage (0.0-1.0)

	TargetWt  float64 `json:"target_wt"`
	TargetVal float64 `json:"target_val"`
	TargetQty int     `json:"target_qty"`
	Action    string  `json:"action"` // BUY, SELL, HOLD
	ActionQty int     `json:"action_qty"`

	MA130      float64 `json:"ma_130"`
	MA130Prev  float64 `json:"ma_130_prev"`
	Condition1 bool    `json:"cond_price_under_ma"` // Price < MA
	Condition2 bool    `json:"cond_ma_down"`        // MA Slope < 0
	KillSwitch bool    `json:"kill_switch"`         // 2-Strike (PFIX/TMF)
}

// CalculateRebalancePlan generates a plan without executing trades
func (s *Strategy) CalculateRebalancePlan() (*RebalancePlan, error) {
	logWithTime("[REBALANCE] Starting calculation...")

	// 1. Define Strategy Constants
	baseWeights := map[string]float64{
		"TQQQ": 0.50,
		"PFIX": 0.15,
		"SCHD": 0.20,
		"TMF":  0.15,
	}
	exchCodes := map[string]string{
		"TQQQ": "NAS",
		"PFIX": "AMS",
		"SCHD": "AMS",
		"TMF":  "AMS",
	}

	// 2. Fetch Portfolio State
	// Get Balance/BuyingPower
	bp, err := s.Client.GetBuyingPower()
	if err != nil {
		return nil, fmt.Errorf("failed to get buying power: %v", err)
	}
	// Parse Cash
	var cash float64
	fmt.Sscanf(bp.Output.OvrsOrdPsblAmt, "%f", &cash)

	// Get Holdings (GetBalance logic)
	bal, err := s.Client.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %v", err)
	}

	type HoldingInfo struct {
		Qty      int
		AvgPrice float64
		Price    float64
	}
	holdingsMap := make(map[string]HoldingInfo)
	for _, h := range bal.Output1 {
		var q int
		var avg, now float64
		fmt.Sscanf(h.Qty, "%d", &q)
		fmt.Sscanf(h.AvgPrice, "%f", &avg)
		fmt.Sscanf(h.NowPrice, "%f", &now)
		holdingsMap[h.Symbol] = HoldingInfo{Qty: q, AvgPrice: avg, Price: now}
	}

	// 3. Process Each Asset (Fetch Data & Calc Logic)
	// We also need to map "Kill Switch" status to apply cross-asset logic (PFIX <-> TMF)
	killStatus := make(map[string]bool)

	// Temporary storage for calculated item before final weight adjustment
	type TempItem struct {
		Symbol string
		Price  float64
		MA130  float64
		MAPrev float64
		RawWt  float64
		Cond1  bool
		Cond2  bool
		IsKill bool
	}
	var tempItems []TempItem
	totalEquity := cash

	for sym, baseWt := range baseWeights {
		exch := exchCodes[sym]

		// A. Get Price History (131 days)
		prices, err := s.Client.GetDailyPrice(exch, sym, 131)
		if err != nil {
			logWithTime("⚠ Failed to get history for %s: %v", sym, err)
			return nil, err
		}

		if len(prices) < 130 {
			return nil, fmt.Errorf("insufficient history for %s: got %d, need 130+", sym, len(prices))
		}

		currentPrice := prices[0].Close // Latest

		// B. Calculate MA130
		// MA Today: 0..129
		sum := 0.0
		for i := 0; i < 130; i++ {
			sum += prices[i].Close
		}
		ma130 := sum / 130.0

		// MA Yesterday: 1..130
		sumPrev := 0.0
		if len(prices) >= 131 {
			for i := 1; i < 131; i++ {
				sumPrev += prices[i].Close
			}
		} else {
			// Fallback if exactly 130 (rare if we asked for 131)
			sumPrev = sum - prices[0].Close + prices[129].Close
		}
		maPrev := sumPrev / 130.0

		// C. Apply Logic
		wt := baseWt
		cond1 := currentPrice < ma130 // Price < MA
		cond2 := ma130 < maPrev       // MA Falling (Slope < 0)
		isKill := false

		if cond1 {
			wt *= 0.5
		}
		if cond2 {
			wt *= 0.5
		}

		// Kill Switch for PFIX / TMF
		if sym == "PFIX" || sym == "TMF" {
			if cond1 && cond2 {
				wt = 0
				isKill = true
				killStatus[sym] = true
			}
		}

		// Add to Equity
		h, exists := holdingsMap[sym]
		if exists {
			if h.Price > 0 {
				currentPrice = h.Price
			}
			totalEquity += float64(h.Qty) * currentPrice
		}

		tempItems = append(tempItems, TempItem{
			Symbol: sym,
			Price:  currentPrice,
			MA130:  ma130,
			MAPrev: maPrev,
			RawWt:  wt,
			Cond1:  cond1,
			Cond2:  cond2,
			IsKill: isKill,
		})
	}

	// 4. Cross-Asset Logic (PFIX <-> TMF)
	for i := range tempItems {
		item := &tempItems[i]
		if item.Symbol == "PFIX" && killStatus["TMF"] {
			if !item.IsKill {
				item.RawWt *= 2
				logWithTime("[REBALANCE] TMF Killed -> Doubling PFIX weight")
			}
		}
		if item.Symbol == "TMF" && killStatus["PFIX"] {
			if !item.IsKill {
				item.RawWt *= 2
				logWithTime("[REBALANCE] PFIX Killed -> Doubling TMF weight")
			}
		}
	}

	// 5. Finalize Items
	var rebalItems []RebalanceItem
	var totalTax float64

	for _, tmp := range tempItems {
		hInfo, exists := holdingsMap[tmp.Symbol]
		currentQty := 0
		avgPrice := 0.0
		if exists {
			currentQty = hInfo.Qty
			avgPrice = hInfo.AvgPrice
		}

		currentVal := float64(currentQty) * tmp.Price
		currentWt := 0.0
		if totalEquity > 0 {
			currentWt = currentVal / totalEquity
		}

		targetVal := totalEquity * tmp.RawWt
		targetQty := int(math.Floor(targetVal / tmp.Price))

		action := "HOLD"
		actionQty := 0
		estTax := 0.0

		if targetQty > currentQty {
			action = "BUY"
			actionQty = targetQty - currentQty
		} else if targetQty < currentQty {
			action = "SELL"
			actionQty = currentQty - targetQty

			// Tax Estimate
			profit := (tmp.Price - avgPrice) * float64(actionQty)
			if profit > 0 {
				estTax = profit * 0.22
			}
		}
		totalTax += estTax

		rebalItems = append(rebalItems, RebalanceItem{
			Symbol:       tmp.Symbol,
			CurrentQty:   currentQty,
			CurrentPrice: tmp.Price,
			CurrentVal:   currentVal,
			CurrentWt:    currentWt,
			TargetWt:     tmp.RawWt,
			TargetVal:    targetVal,
			TargetQty:    targetQty,
			Action:       action,
			ActionQty:    actionQty,
			MA130:        tmp.MA130,
			MA130Prev:    tmp.MAPrev,
			Condition1:   tmp.Cond1,
			Condition2:   tmp.Cond2,
			KillSwitch:   tmp.IsKill,
		})
	}

	plan := &RebalancePlan{
		TotalValue:    totalEquity,
		Cash:          cash,
		Items:         rebalItems,
		EstimatedTax:  totalTax,
		ActionSummary: fmt.Sprintf("Equity: $%.2f, Est. Tax: $%.2f", totalEquity, totalTax),
	}

	logWithTime("[REBALANCE] Plan calculated. Total Equity: $%.2f", totalEquity)
	return plan, nil
}

// ExecuteRebalance executes the plan
func (s *Strategy) ExecuteRebalance(dryRun bool) error {
	plan, err := s.CalculateRebalancePlan()
	if err != nil {
		return err
	}

	logWithTime("[REBALANCE] Executing Plan (DryRun=%v)...", dryRun)
	logWithTime("[REBALANCE] %s", plan.ActionSummary)

	var sells, buys []RebalanceItem
	for _, item := range plan.Items {
		if item.Action == "SELL" {
			sells = append(sells, item)
		} else if item.Action == "BUY" {
			buys = append(buys, item)
		}
	}

	// 1. Sells
	for _, item := range sells {
		if item.ActionQty == 0 {
			continue
		}
		s.placeRebalanceOrder(item, dryRun)
	}

	// 2. Buys
	for _, item := range buys {
		if item.ActionQty == 0 {
			continue
		}
		s.placeRebalanceOrder(item, dryRun)
	}

	logWithTime("[REBALANCE] Execution Completed.")
	return nil
}

func (s *Strategy) placeRebalanceOrder(item RebalanceItem, dryRun bool) {
	logWithTime("[REBALANCE] %s %d shares of %s (Target: %d, Current: %d)",
		item.Action, item.ActionQty, item.Symbol, item.TargetQty, item.CurrentQty)

	if dryRun {
		return
	}

	exch := "NYS"
	if item.Symbol == "TQQQ" {
		exch = "NASD"
	}

	orderReq := kis.OrderReq{
		ExchCode: exch,
		Symbol:   item.Symbol,
		Qty:      item.ActionQty,
		Price:    item.CurrentPrice,
		OrdType:  "00", // Limit
		Side:     item.Action,
	}

	if err := s.Client.PlaceOrder(orderReq); err != nil {
		logWithTime("[REBALANCE] ✗ Failed to %s %s: %v", item.Action, item.Symbol, err)
	} else {
		logWithTime("[REBALANCE] ✓ %s Order PLACED for %s", item.Action, item.Symbol)
	}
}
