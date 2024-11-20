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

/*---------------- volume analysis ----------------*/
// VolumeData contains volume-based analysis metrics
type VolumeData struct {
	VolumeRatio  float64 // Current/Average volume ratio
	TradeCount   int64   // Number of trades in period
	AvgTradeSize float64 // Average size per trade
	Confidence   float64 // Volume-based confidence (0-1)
}

type timeframeVolumeMetrics struct { // VolumeData
	volumeRatio    float64
	tradeCount     int64
	avgTradeSize   float64
	confidence     float64
	volumeTrend    float64
	trendStrength  float64
	weightedVolume float64
}

/*---------------- technical analysis ----------------*/
// TechnicalData holds indicator-based metrics
type TechnicalData struct {
	Signal     int     // 1: bullish, -1: bearish, 0: neutral
	Confidence float64 // 0-1
	EMA        struct {
		Values    map[int]float64
		Direction int // 1: up, -1: down, 0: neutral
		Slope     float64
		Strength  float64
	}
	RSI struct {
		Value     float64
		Signal    float64
		Histogram float64
		Trend     int // 1: bullish, -1: bearish, 0: neutral
		Strength  float64
	}
}

/*---------------- price analysis ----------------*/
// PriceData contains price action metrics
type PriceData struct {
	Current    float64
	Momentum   float64 // Rate of price change
	Volatility float64 // Based on recent price movement
	Signal     int     // 1: bullish, -1: bearish, 0: neutral
	Confidence float64 // Price-based confidence (0-1)
}

/*---------------- pattern analysis ----------------*/
// PatternResult represents a detected pattern
type PatternResult struct {
	Type     string
	Signal   int     // 1: bullish, -1: bearish, 0: neutral
	Strength float64 // 0-1
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
