package model

import (
	"time"

	"gorm.io/gorm"
)

type UserSettings struct {
	gorm.Model
	Principal  float64 // Total Investment (e.g. 10000)
	SplitCount int     // Default 40
	TargetRate float64 // Default 0.10 (10%)
	Symbols    string  // Comma separated, e.g., "TQQQ,SOXL"
	IsActive   bool    // Logic On/Off
}

type TradeLog struct {
	gorm.Model
	Date   time.Time
	Symbol string
	Side   string // BUY, SELL
	Type   string // LIMIT, LOC, MARKET
	Qty    int
	Price  float64
	Amount float64
	Profit float64 // Only for SELL
}

// Daily status ensuring we track the 40-day cycle
type CycleStatus struct {
	gorm.Model
	Symbol          string `gorm:"uniqueIndex"`
	CurrentCycleDay int    // 1 to 40
	TotalBoughtQty  int
	AvgPrice        float64
	TotalInvested   float64
}
