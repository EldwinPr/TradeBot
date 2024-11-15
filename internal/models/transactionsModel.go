package models

import (
	"time"
)

type Transaction struct {
	ID         uint    `gorm:"primaryKey"`
	PositionID uint    `gorm:"index;not null"`
	Type       string  `gorm:"not null"`
	Amount     float64 `gorm:"type:decimal(20,8);not null"`

	// Time
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// Relationships
	Position Position `gorm:"foreignKey:PositionID"`
}

const (
	TransactionTypeDeposit  = "deposit"
	TransactionTypeWithdraw = "withdraw"
	TransactionTypeTrade    = "trade"
)
