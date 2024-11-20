package analysis

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/indicators"
	"math"
)

type TechnicalAnalyzer struct {
	weights map[string]float64
	ema     *indicators.EMAService
	rsi     *indicators.RSIService
}

func NewTechnicalAnalyzer() *TechnicalAnalyzer {
	return &TechnicalAnalyzer{
		weights: map[string]float64{
			models.PriceTimeFrame5m:  0.30,
			models.PriceTimeFrame15m: 0.35,
			models.PriceTimeFrame1h:  0.20,
			models.PriceTimeFrame4h:  0.15,
		},
		ema: indicators.NewEMAService(),
		rsi: indicators.NewRSIService(),
	}
}

func (a *TechnicalAnalyzer) Analyze(prices5m, prices15m, prices1h, prices4h []models.Price) (*TechnicalData, error) {
	m5 := a.analyzeTimeframe(prices5m)
	m15 := a.analyzeTimeframe(prices15m)
	m1h := a.analyzeTimeframe(prices1h)
	m4h := a.analyzeTimeframe(prices4h)

	// Weight signals
	signal := a.weightSignals(m5, m15, m1h, m4h)

	// Weight confidences
	confidence := a.weightConfidences(m5, m15, m1h, m4h)

	return &TechnicalData{
		Signal:     signal,
		Confidence: confidence,
		EMA:        m5.EMA, // Use shortest timeframe for current values
		RSI:        m5.RSI,
	}, nil
}

func (a *TechnicalAnalyzer) analyzeTimeframe(prices []models.Price) *TechnicalData {
	closes := extractCloses(prices)

	// Calculate EMAs (unchanged)
	ema8Values := a.ema.Calculate(closes, 8)
	ema21Values := a.ema.Calculate(closes, 21)

	// Get latest EMAs (unchanged)
	ema8Result := a.ema.CalculatePoint(
		closes[len(closes)-1],
		ema8Values[len(ema8Values)-2],
		8,
	)

	// Check crossovers (unchanged)
	crossSignal := a.ema.CheckCrossover(ema8Values, ema21Values)

	// Calculate RSI with signal
	rsi := a.rsi.Calculate(closes, 14, 3)

	// Need: price, prevPrice, prevGain, prevLoss, prevRSI, prevSignal, period, smoothPeriod
	rsiPoint := a.rsi.CalculatePoint(
		closes[len(closes)-1],         // price
		closes[len(closes)-2],         // prevPrice
		0.0,                           // prevGain
		0.0,                           // prevLoss
		rsi.RSI[len(rsi.RSI)-2],       // prevRSI
		rsi.Signal[len(rsi.Signal)-2], // prevSignal
		14,                            // period
		3,                             // smoothPeriod
	)

	td := &TechnicalData{
		EMA: struct {
			Values    map[int]float64
			Direction int
			Slope     float64
			Strength  float64
		}{
			Values: map[int]float64{
				8:  ema8Values[len(ema8Values)-1],
				21: ema21Values[len(ema21Values)-1],
			},
			Direction: ema8Result.Direction,
			Slope:     ema8Result.Slope,
			Strength:  crossSignal.Strength,
		},
		RSI: struct {
			Value     float64
			Signal    float64
			Histogram float64
			Trend     int
			Strength  float64
		}{
			Value:     rsiPoint.Value,
			Signal:    rsiPoint.Signal,
			Histogram: rsiPoint.Histogram,
			Trend:     rsiPoint.Trend,
			Strength:  rsiPoint.Strength,
		},
	}

	td.Signal = a.calculateSignal(td, crossSignal)
	td.Confidence = a.calculateConfidence(td)

	return td
}

func (a *TechnicalAnalyzer) weightSignals(m5, m15, m1h, m4h *TechnicalData) int {
	weightedSignal := float64(m5.Signal)*a.weights[models.PriceTimeFrame5m] +
		float64(m15.Signal)*a.weights[models.PriceTimeFrame15m] +
		float64(m1h.Signal)*a.weights[models.PriceTimeFrame1h] +
		float64(m4h.Signal)*a.weights[models.PriceTimeFrame4h]

	if weightedSignal > 0.2 {
		return 1
	} else if weightedSignal < -0.2 {
		return -1
	}
	return 0
}

func (a *TechnicalAnalyzer) weightConfidences(m5, m15, m1h, m4h *TechnicalData) float64 {
	return m5.Confidence*a.weights[models.PriceTimeFrame5m] +
		m15.Confidence*a.weights[models.PriceTimeFrame15m] +
		m1h.Confidence*a.weights[models.PriceTimeFrame1h] +
		m4h.Confidence*a.weights[models.PriceTimeFrame4h]
}

func (a *TechnicalAnalyzer) calculateSignal(td *TechnicalData, crossSignal *indicators.CrossSignal) int {
	// Base signal from EMAs
	signal := td.EMA.Direction

	// Strong signal on crossover with RSI confirmation
	if crossSignal.Crossed {
		if (crossSignal.Direction == 1 && td.RSI.Value > 50) ||
			(crossSignal.Direction == -1 && td.RSI.Value < 50) {
			return crossSignal.Direction
		}
	}

	// RSI trend confirmation or override
	if td.RSI.Trend != 0 {
		if td.RSI.Trend == signal {
			return signal
		} else if td.RSI.Strength > 0.7 {
			return td.RSI.Trend
		}
	}

	// Filter extreme RSI
	if (td.RSI.Value > 70 && signal == 1) ||
		(td.RSI.Value < 30 && signal == -1) {
		return 0
	}

	return signal
}

func (a *TechnicalAnalyzer) calculateConfidence(td *TechnicalData) float64 {
	// Base confidence from indicators
	emaConf := td.EMA.Strength * 0.4
	rsiConf := td.RSI.Strength * 0.4

	// Trend alignment bonus
	alignmentConf := 0.0
	if td.EMA.Direction == td.RSI.Trend {
		alignmentConf = 0.2
	}

	// Calculate total
	confidence := emaConf + rsiConf + alignmentConf

	// Reduce confidence in RSI middle zone
	if td.RSI.Value > 45 && td.RSI.Value < 55 {
		confidence *= 0.8
	}

	return math.Min(confidence, 1.0)
}

func extractCloses(prices []models.Price) []float64 {
	closes := make([]float64, len(prices))
	for i, p := range prices {
		closes[i] = p.Close
	}
	return closes
}
