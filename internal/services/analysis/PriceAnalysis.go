package analysis

import (
	"CryptoTradeBot/internal/models"
	"math"
)

type PriceAnalyzer struct {
	weights map[string]float64
}

func NewPriceAnalyzer() *PriceAnalyzer {
	return &PriceAnalyzer{
		weights: map[string]float64{
			models.PriceTimeFrame5m:  0.15,
			models.PriceTimeFrame15m: 0.25,
			models.PriceTimeFrame1h:  0.35,
			models.PriceTimeFrame4h:  0.25,
		},
	}
}

func (a *PriceAnalyzer) Analyze(prices5m, prices15m, prices1h, prices4h []models.Price) (*PriceData, error) {
	// Calculate weighted momentum across timeframes
	m5 := a.calculateTimeframeMomentum(prices5m, 12)
	m15 := a.calculateTimeframeMomentum(prices15m, 12)
	m1h := a.calculateTimeframeMomentum(prices1h, 6)
	m4h := a.calculateTimeframeMomentum(prices4h, 6)

	weightedMomentum := (m5 * a.weights[models.PriceTimeFrame5m]) +
		(m15 * a.weights[models.PriceTimeFrame15m]) +
		(m1h * a.weights[models.PriceTimeFrame1h]) +
		(m4h * a.weights[models.PriceTimeFrame4h])

	volatility := a.calculateVolatility(prices5m)
	confidence := a.calculateConfidence(m5, m15, m1h, m4h, volatility)

	// Determine signal direction
	signal := 0
	if weightedMomentum > 0.001 { // Small threshold to avoid noise
		signal = 1
	} else if weightedMomentum < -0.001 {
		signal = -1
	}

	return &PriceData{
		Current:    prices5m[len(prices5m)-1].Close,
		Momentum:   weightedMomentum,
		Volatility: volatility,
		Confidence: confidence,
		Signal:     signal,
	}, nil
}

func (a *PriceAnalyzer) calculateTimeframeMomentum(prices []models.Price, window int) float64 {
	recent := prices[len(prices)-window:]

	var momentum float64
	weight := 1.0
	totalWeight := 0.0

	for i := 1; i < len(recent); i++ {
		change := (recent[i].Close - recent[i-1].Close) / recent[i-1].Close
		momentum += change * weight
		totalWeight += weight
		weight *= 0.9
	}

	return momentum / totalWeight
}

func (a *PriceAnalyzer) calculateConfidence(m5, m15, m1h, m4h, volatility float64) float64 {
	// Check momentum alignment across timeframes
	alignmentScore := 0.0
	if (m5 > 0 && m15 > 0 && m1h > 0 && m4h > 0) ||
		(m5 < 0 && m15 < 0 && m1h < 0 && m4h < 0) {
		alignmentScore = 1.0
	} else if (m15 > 0 && m1h > 0 && m4h > 0) ||
		(m15 < 0 && m1h < 0 && m4h < 0) {
		alignmentScore = 0.8
	} else if (m1h > 0 && m4h > 0) || (m1h < 0 && m4h < 0) {
		alignmentScore = 0.6
	}

	// Adjust for volatility
	volatilityScore := math.Max(0, 1-volatility)

	return alignmentScore * volatilityScore
}

func (a *PriceAnalyzer) calculateVolatility(prices []models.Price) float64 {
	window := prices[len(prices)-12:] // Last hour for 5m

	var sumSquares float64
	mean := window[len(window)-1].Close

	for _, p := range window {
		diff := (p.Close - mean) / mean
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(window)))
}
