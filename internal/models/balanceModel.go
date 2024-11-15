package models

import (
	"time"
)

type Balance struct {
	ID      uint    `gorm:"primaryKey"`
	Symbol  string  `gorm:"index;not null"`
	Balance float64 `gorm:"type:decimal(20,8);not null"`

	LastUpdated time.Time `gorm:"index;not null"`
}
