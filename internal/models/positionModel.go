package models

import "time"

type Position struct {
	ID         uint    `gorm:"primaryKey"`
	Symbol     string  `gorm:"index;not null"`
	Side       string  `gorm:"not null"`
	Size       float64 `gorm:"type:decimal(20,8);not null"`
	Leverage   int     `gorm:"not null"`
	EntryPrice float64 `gorm:"type:decimal(20,8);not null"`

	StopLossPrice   float64 `gorm:"type:decimal(20,8);not null"`
	TakeProfitPrice float64 `gorm:"type:decimal(20,8);not null"`

	PnL float64 `gorm:"type:decimal(20,8)"`

	OpenTime  time.Time `gorm:"index;not null"`
	CloseTime time.Time `gorm:"index"`
	Status    string    `gorm:"not null"`

	Confidence float64 `gorm:"type:decimal(20,8)"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	DeletedAt time.Time `gorm:"index"`
}

const (
	PositionStatusOpen   = "open"
	PositionStatusClosed = "closed"

	PositionSideLong  = "long"
	PositionSideShort = "short"
)
