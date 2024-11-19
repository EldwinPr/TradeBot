package analysis

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/indicators"
	"math"
	"time"
)

// Constants for analysis
const (
	TargetProfit  = 0.01  // 1% target
	StopLoss      = 0.006 // 0.6% stop loss
	MinConfidence = 0.7   // Minimum confidence for entry

	// Lookback periods
	ShortLook  = 5  // Immediate price action
	MediumLook = 10 // Recent trend
)

type Analysis struct {
	ema  *indicators.EMAService
	rsi  *indicators.RSIService
	macd *indicators.MACDService
}

func NewAnalysis() *Analysis {
	return &Analysis{
		ema:  indicators.NewEMAService(),
		rsi:  indicators.NewRSIService(),
		macd: indicators.NewMACDService(),
	}
}

// Analyze performs quick market analysis optimized for 1% moves
func (a *Analysis) Analyze(prices []models.Price) *AnalysisResult {
	if len(prices) < MediumLook {
		return newInvalidResult(prices[len(prices)-1].Symbol, "insufficient data")
	}

	// Calculate indicators
	indicators := a.calculateIndicators(prices)

	// Quick momentum check
	momentum := a.checkMomentum(prices[len(prices)-ShortLook:])

	// Volume analysis
	volume := a.checkVolume(prices[len(prices)-ShortLook:])

	// Calculate setup confidence
	confidence := a.calculateConfidence(indicators, momentum, volume)

	// Determine direction
	direction := a.determineDirection(indicators, momentum)

	if confidence < MinConfidence {
		return newInvalidResult(prices[len(prices)-1].Symbol, "low confidence")
	}

	currentPrice := prices[len(prices)-1].Close

	return &AnalysisResult{
		Symbol:     prices[len(prices)-1].Symbol,
		Timestamp:  time.Now(),
		IsValid:    true,
		Direction:  direction,
		EntryPrice: currentPrice,
		TakeProfit: calculateTarget(currentPrice, direction),
		StopLoss:   calculateStop(currentPrice, direction),
		Confidence: confidence,
	}
}

// checkMomentum analyzes short-term price movement
func (a *Analysis) checkMomentum(prices []models.Price) float64 {
	if len(prices) < 2 {
		return 0
	}

	// Calculate rapid price changes
	changes := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		changes[i-1] = (prices[i].Close - prices[i-1].Close) / prices[i-1].Close
	}

	// Return recent momentum strength
	return math.Abs(sum(changes))
}

func (a *Analysis) calculateIndicators(prices []models.Price) *IndicatorValues {
	// Extract close prices
	closes := make([]float64, len(prices))
	volumes := make([]float64, len(prices))
	for i, p := range prices {
		closes[i] = p.Close
		volumes[i] = p.Volume
	}

	// Calculate EMAs
	ema8 := a.ema.Calculate(closes, 8)
	ema21 := a.ema.Calculate(closes, 21)

	// Calculate RSI
	rsi := a.rsi.Calculate(closes, 14)

	// Calculate MACD
	macdResult := a.macd.Calculate(closes, 12, 26, 9)

	// Get latest volume
	currentVolume := volumes[len(volumes)-1]

	return &IndicatorValues{
		RSI:       rsi[len(rsi)-1],
		MACD:      macdResult.MACD[len(macdResult.MACD)-1],
		Signal:    macdResult.Signal[len(macdResult.Signal)-1],
		Histogram: macdResult.Histogram[len(macdResult.Histogram)-1],
		EMA8:      ema8[len(ema8)-1],
		EMA21:     ema21[len(ema21)-1],
		Volume:    currentVolume,
	}
}

func (a *Analysis) checkVolume(prices []models.Price) bool {
	if len(prices) < 2 {
		return false
	}

	// Calculate average volume
	var avgVolume float64
	for i := 0; i < len(prices)-1; i++ {
		avgVolume += prices[i].Volume
	}
	avgVolume /= float64(len(prices) - 1)

	// Check if current volume is higher
	return prices[len(prices)-1].Volume > avgVolume*1.2
}

// calculateConfidence determines entry probability
func (a *Analysis) calculateConfidence(ind *IndicatorValues, momentum float64, volume bool) float64 {
	baseConf := 0.0

	// Trend alignment check
	if ind.EMA8 > ind.EMA21 && momentum > 0 {
		baseConf += 0.4
	} else if ind.EMA8 < ind.EMA21 && momentum < 0 {
		baseConf += 0.4
	}

	// RSI check (favor swings back from extremes)
	if ind.RSI > 40 && ind.RSI < 60 {
		baseConf += 0.3
	}

	// MACD confirmation
	if (ind.MACD > ind.Signal && momentum > 0) ||
		(ind.MACD < ind.Signal && momentum < 0) {
		baseConf += 0.3
	}

	// Volume adjustment
	if volume {
		baseConf *= 1.2
	} else {
		baseConf *= 0.8
	}

	return math.Min(baseConf, 1.0)
}

// determineDirection identifies optimal trade direction
func (a *Analysis) determineDirection(ind *IndicatorValues, momentum float64) string {
	// Combine EMA and momentum direction
	if ind.EMA8 > ind.EMA21 && momentum > 0 {
		return "long"
	} else if ind.EMA8 < ind.EMA21 && momentum < 0 {
		return "short"
	}

	return ""
}

// Helper functions for price calculations
func calculateTarget(price float64, direction string) float64 {
	if direction == "long" {
		return price * (1 + TargetProfit)
	}
	return price * (1 - TargetProfit)
}

func calculateStop(price float64, direction string) float64 {
	if direction == "long" {
		return price * (1 - StopLoss)
	}
	return price * (1 + StopLoss)
}

func sum(values []float64) float64 {
	var total float64
	for _, v := range values {
		total += v
	}
	return total
}

func newInvalidResult(symbol, reason string) *AnalysisResult {
	return &AnalysisResult{
		Symbol:     symbol,
		Timestamp:  time.Now(),
		IsValid:    false,
		Reason:     reason,
		Confidence: 0,
	}
}
