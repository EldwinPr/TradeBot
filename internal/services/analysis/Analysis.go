package analysis

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/indicators"
	"fmt"
	"math"
	"time"
)

const (
	MinimumDataPoints = 200  // Minimum candles needed
	MinConfidence     = 0.70 // 70% minimum confidence for trade

	// Fixed targets
	TakeProfit = 0.01  // 1% target
	StopLoss   = 0.006 // 0.6% stop loss
)

type Analysis struct {
	ema  *indicators.EMAService
	rsi  *indicators.RSIService
	macd *indicators.MACDService
}

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
	Indicators *IndicatorValues
}

type IndicatorValues struct {
	RSI       float64
	MACD      float64
	Signal    float64
	Histogram float64
	EMA8      float64
	EMA21     float64
	Volume    float64
	AvgVolume float64
}

func NewAnalysis() *Analysis {
	return &Analysis{
		ema:  indicators.NewEMAService(),
		rsi:  indicators.NewRSIService(),
		macd: indicators.NewMACDService(),
	}
}

func (a *Analysis) Analyze(prices []models.Price) *AnalysisResult {
	// Initial validation
	if len(prices) < MinimumDataPoints {
		return newInvalidResult(prices[len(prices)-1].Symbol, "insufficient data points")
	}

	// Calculate indicators
	indicators, err := a.calculateIndicators(prices)
	if err != nil {
		return newInvalidResult(prices[len(prices)-1].Symbol, fmt.Sprintf("indicator calculation failed: %v", err))
	}

	// Current price info
	currentPrice := prices[len(prices)-1]

	// Determine trading setup
	direction := a.determineDirection(indicators)
	if direction == "" {
		return newInvalidResult(currentPrice.Symbol, "no clear direction")
	}

	// Calculate confidence
	confidence := a.calculateConfidence(indicators, direction)
	if confidence < MinConfidence {
		return newInvalidResult(currentPrice.Symbol, "insufficient confidence")
	}

	// Valid setup found, calculate targets
	return &AnalysisResult{
		Symbol:     currentPrice.Symbol,
		Timestamp:  currentPrice.OpenTime,
		IsValid:    true,
		Direction:  direction,
		EntryPrice: currentPrice.Close,
		TakeProfit: calculateTarget(currentPrice.Close, direction, TakeProfit),
		StopLoss:   calculateTarget(currentPrice.Close, direction, StopLoss),
		Confidence: confidence,
		Reason:     "valid setup found",
		Indicators: indicators,
	}
}

func (a *Analysis) calculateIndicators(prices []models.Price) (*IndicatorValues, error) {
	// Extract price data
	closes := make([]float64, len(prices))
	volumes := make([]float64, len(prices))

	for i, p := range prices {
		closes[i] = p.Close
		volumes[i] = p.Volume
	}

	// Calculate EMAs
	ema8 := a.ema.Calculate(closes, 8)
	if ema8 == nil {
		return nil, fmt.Errorf("EMA8 calculation failed")
	}

	ema21 := a.ema.Calculate(closes, 21)
	if ema21 == nil {
		return nil, fmt.Errorf("EMA21 calculation failed")
	}

	// Calculate RSI
	rsi := a.rsi.Calculate(closes, 14)
	if rsi == nil {
		return nil, fmt.Errorf("RSI calculation failed")
	}

	// Calculate MACD
	macdResult := a.macd.Calculate(closes, 12, 26, 9)
	if macdResult == nil {
		return nil, fmt.Errorf("MACD calculation failed")
	}

	// Calculate average volume (last 20 periods)
	avgVolume := calculateAverageVolume(volumes, 20)

	// Get latest values
	lastIndex := len(prices) - 1

	return &IndicatorValues{
		RSI:       rsi[lastIndex],
		MACD:      macdResult.MACD[lastIndex],
		Signal:    macdResult.Signal[lastIndex],
		Histogram: macdResult.Histogram[lastIndex],
		EMA8:      ema8[lastIndex],
		EMA21:     ema21[lastIndex],
		Volume:    volumes[lastIndex],
		AvgVolume: avgVolume,
	}, nil
}

func (a *Analysis) determineDirection(ind *IndicatorValues) string {
	// Check for long setup
	if isLongSetup(ind) {
		return "long"
	}

	// For now, we're only taking long trades
	return ""
}

func isLongSetup(ind *IndicatorValues) bool {
	// EMA alignment
	if ind.EMA8 <= ind.EMA21 {
		return false
	}

	// RSI conditions (not oversold or overbought)
	if ind.RSI <= 40 || ind.RSI >= 75 {
		return false
	}

	// MACD confirmation
	if ind.MACD <= ind.Signal {
		return false
	}

	// Volume confirmation
	if ind.Volume < ind.AvgVolume {
		return false
	}

	return true
}

func (a *Analysis) calculateConfidence(ind *IndicatorValues, direction string) float64 {
	if direction != "long" {
		return 0
	}

	var confidence float64

	// EMA trend strength (40%)
	emaSpread := (ind.EMA8 - ind.EMA21) / ind.EMA21
	confidence += math.Min(emaSpread*20, 0.4) // Cap at 0.4

	// RSI position (30%)
	if ind.RSI > 40 && ind.RSI < 75 {
		rsiScore := (ind.RSI - 40) / 35 // Normalize to 0-1
		confidence += rsiScore * 0.3
	}

	// MACD momentum (30%)
	if ind.MACD > ind.Signal {
		macdScore := math.Min(ind.Histogram/ind.Signal, 1.0)
		confidence += macdScore * 0.3
	}

	return confidence
}

func calculateTarget(price float64, direction string, percentage float64) float64 {
	if direction == "long" {
		return price * (1 + percentage)
	}
	return price * (1 - percentage)
}

func calculateAverageVolume(volumes []float64, periods int) float64 {
	if len(volumes) < periods {
		return 0
	}

	start := len(volumes) - periods
	var sum float64
	for i := start; i < len(volumes); i++ {
		sum += volumes[i]
	}

	return sum / float64(periods)
}

func newInvalidResult(symbol, reason string) *AnalysisResult {
	return &AnalysisResult{
		Symbol:    symbol,
		Timestamp: time.Now(),
		IsValid:   false,
		Reason:    reason,
	}
}
