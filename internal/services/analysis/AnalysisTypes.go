package analysis

import (
	"time"
)

type AnalysisResult struct {
	Symbol     string
	Timestamp  time.Time
	IsValid    bool
	Direction  string // "long" or "short"
	EntryPrice float64
	TakeProfit float64
	StopLoss   float64
	Confidence float64
	Reason     string
}

type IndicatorValues struct {
	RSI       float64
	MACD      float64
	Signal    float64
	Histogram float64
	EMA8      float64
	EMA21     float64
	Volume    float64
}
