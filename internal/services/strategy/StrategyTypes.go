package strategy

import (
	"CryptoTradeBot/internal/services/analysis"
	"time"
)

// StrategyResult represents the output of a strategy analysis
type StrategyResult struct {
	// Core fields
	IsValid   bool
	Direction string // "long" or "short"
	Reason    string // If invalid, explains why

	// Price levels
	EntryPrice float64
	StopLoss   float64
	TakeProfit float64

	// Confidence and Analysis
	Confidence float64
	Volume     analysis.VolumeData
	Technical  analysis.TechnicalData
	Price      analysis.PriceData
}

type PositionRequest struct {
	Action     string
	Symbol     string
	Side       string
	EntryPrice float64
	StopLoss   float64
	TakeProfit float64
	Confidence float64

	// Analysis results
	Volume    analysis.VolumeData
	Technical analysis.TechnicalData
	Price     analysis.PriceData

	Timestamp time.Time
}

// Helper function for invalid results
func newInvalidResult(reason string) *StrategyResult {
	return &StrategyResult{
		IsValid: false,
		Reason:  reason,
	}
}
