package models

import (
	"time"
)

type Price struct {
	ID         uint      `gorm:"primaryKey"`
	Symbol     string    `gorm:"index;not null"`
	TimeFrame  string    `gorm:"not null"`
	OpenTime   time.Time `gorm:"index;not null"`
	CloseTime  time.Time `gorm:"index"`
	Open       float64   `gorm:"type:decimal(20,8)"`
	Close      float64   `gorm:"type:decimal(20,8)"`
	High       float64   `gorm:"type:decimal(20,8)"`
	Low        float64   `gorm:"type:decimal(20,8)"`
	Volume     float64   `gorm:"type:decimal(20,8)"`
	TradeCount int64
}

const (
	PriceTimeFrame5m  = "5m"
	PriceTimeFrame15m = "15m"
	PriceTimeFrame1h  = "1h"
	PriceTimeFrame4h  = "4h"
)

// TableName sets the table name for Price model
func (Price) TableName() string {
	return "prices"
}
