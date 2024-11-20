package analysis

import "time"

// AnalysisResult represents the complete analysis output
type AnalysisResult struct {
	Symbol     string
	Timestamp  time.Time
	IsValid    bool
	Direction  string  // "long" or "short"
	Confidence float64 // Overall confidence score (0-1)
	Volume     VolumeData
	Technical  TechnicalData
	Price      PriceData
	Reason     string // If invalid, explains why
}

// VolumeData contains volume-based analysis metrics
type VolumeData struct {
	VolumeRatio  float64 // Current/Average volume ratio
	TradeCount   int64   // Number of trades in period
	AvgTradeSize float64 // Average size per trade
	Confidence   float64 // Volume-based confidence (0-1)
}

// TechnicalData holds indicator-based metrics
type TechnicalData struct {
	EMA8       float64
	EMA21      float64
	RSI        float64
	RSISignal  float64
	EMATrend   int     // 1: up, -1: down, 0: neutral
	Confidence float64 // Technical confidence (0-1)
}

// PriceData contains price action metrics
type PriceData struct {
	Current    float64
	Momentum   float64 // Rate of price change
	Volatility float64 // Based on recent price movement
	Confidence float64 // Price-based confidence (0-1)
}

// AnalysisConfig holds minimum confidence requirements
type AnalysisConfig struct {
	MinConfidence float64 // Minimum overall confidence (0.7 default)
	Weights       struct {
		Volume    float64 // 0.3
		Technical float64 // 0.35
		Price     float64 // 0.35
	}
}
