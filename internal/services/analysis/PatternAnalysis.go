package analysis

import (
	"CryptoTradeBot/internal/models"
	"math"
)

type PatternAnalyzer struct {
	minHeight float64
}

func NewPatternAnalyzer() *PatternAnalyzer {
	return &PatternAnalyzer{
		minHeight: 0.001,
	}
}

func (a *PatternAnalyzer) Analyze(candles []models.Price) *PatternResult {
	if len(candles) < 3 {
		return nil
	}

	c2 := candles[len(candles)-3]
	c1 := candles[len(candles)-2]
	c0 := candles[len(candles)-1]

	// Three-bar pattern
	if pattern := a.checkThreeBar(c2, c1, c0); pattern != nil {
		return pattern
	}

	// Two-bar patterns
	if pattern := a.checkEngulfing(c1, c0); pattern != nil {
		return pattern
	}

	// Single-bar patterns
	if pattern := a.checkPinbar(c0); pattern != nil {
		return pattern
	}

	return nil
}

func (a *PatternAnalyzer) checkThreeBar(c2, c1, c0 models.Price) *PatternResult {
	// Higher lows pattern
	if c0.Low > c1.Low && c1.Low > c2.Low {
		strength := (c0.Low - c2.Low) / c2.Low
		return &PatternResult{
			Type:     "HigherLows",
			Signal:   1,
			Strength: math.Min(strength*10, 1.0),
		}
	}

	// Lower highs pattern
	if c0.High < c1.High && c1.High < c2.High {
		strength := (c2.High - c0.High) / c2.High
		return &PatternResult{
			Type:     "LowerHighs",
			Signal:   -1,
			Strength: math.Min(strength*10, 1.0),
		}
	}

	return nil
}

func (a *PatternAnalyzer) checkEngulfing(prev, curr models.Price) *PatternResult {
	prevSize := math.Abs(prev.Close - prev.Open)
	currSize := math.Abs(curr.Close - curr.Open)

	if currSize < a.minHeight {
		return nil
	}

	// Bullish engulfing
	if curr.Open < prev.Close && curr.Close > prev.Open {
		return &PatternResult{
			Type:     "BullishEngulfing",
			Signal:   1,
			Strength: math.Min(currSize/prevSize, 1.0),
		}
	}

	// Bearish engulfing
	if curr.Open > prev.Close && curr.Close < prev.Open {
		return &PatternResult{
			Type:     "BearishEngulfing",
			Signal:   -1,
			Strength: math.Min(currSize/prevSize, 1.0),
		}
	}

	return nil
}

func (a *PatternAnalyzer) checkPinbar(candle models.Price) *PatternResult {
	bodySize := math.Abs(candle.Close - candle.Open)
	upperWick := candle.High - math.Max(candle.Open, candle.Close)
	lowerWick := math.Min(candle.Open, candle.Close) - candle.Low
	totalSize := candle.High - candle.Low

	if totalSize < a.minHeight {
		return nil
	}

	// Bullish pinbar
	if lowerWick > (totalSize*0.6) && bodySize < (totalSize*0.3) {
		return &PatternResult{
			Type:     "BullishPinbar",
			Signal:   1,
			Strength: lowerWick / totalSize,
		}
	}

	// Bearish pinbar
	if upperWick > (totalSize*0.6) && bodySize < (totalSize*0.3) {
		return &PatternResult{
			Type:     "BearishPinbar",
			Signal:   -1,
			Strength: upperWick / totalSize,
		}
	}

	return nil
}
