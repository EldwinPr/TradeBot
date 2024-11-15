package models

import (
	"time"

	"gorm.io/gorm"
)

type Price struct {
	ID        uint           `gorm:"primaryKey"`
	Symbol    string         `gorm:"index;not null"`
	Price     float64        `gorm:"type:decimal(20,8);not null"`
	TimeFrame string         `gorm:"not null"`
	OpenTime  time.Time      `gorm:"index;not null"`
	CloseTime time.Time      `gorm:"index"`
	Open      float64        `gorm:"type:decimal(20,8)"`
	High      float64        `gorm:"type:decimal(20,8)"`
	Low       float64        `gorm:"type:decimal(20,8)"`
	Volume    float64        `gorm:"type:decimal(20,8)"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

const (
	PriceTimeFrame1m  = "1m"
	PriceTimeFrame15m = "15m"
	PriceTimeFrame1h  = "1h"
	PriceTimeFrame4h  = "4h"
	PriceTimeFrame1d  = "1d"
)

// TableName sets the table name for Price model
func (Price) TableName() string {
	return "prices"
}
