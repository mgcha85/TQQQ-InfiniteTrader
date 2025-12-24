package repository

import (
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
}

func NewDB(path string) (*DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto Migrate
	err = db.AutoMigrate(
		&model.UserSettings{},
		&model.TradeLog{},
		&model.CycleStatus{},
	)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}
