package analysis

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/indicators"
	"log"
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

	// Calculate EMAs
	ema8Values := a.ema.Calculate(closes, 8)
	ema21Values := a.ema.Calculate(closes, 21)

	// Get latest values
	lastIndex := len(closes) - 1
	ema8 := ema8Values[lastIndex]
	ema21 := ema21Values[lastIndex]

	// Calculate EMA direction once
	emaDirection := 0
	if ema8 > ema21 {
		emaDirection = 1
	} else if ema8 < ema21 {
		emaDirection = -1
	}

	// Calculate EMA slope (trend strength)
	emaSlope := 0.0
	if len(ema8Values) > 1 {
		emaSlope = (ema8Values[lastIndex] - ema8Values[lastIndex-1]) / ema8Values[lastIndex-1]
	}

	// Calculate RSI
	rsiResult := a.rsi.Calculate(closes, 14, 3)
	currentRSI := rsiResult.RSI[len(rsiResult.RSI)-1]
	currentSignal := rsiResult.Signal[len(rsiResult.Signal)-1]
	currentHistogram := currentRSI - currentSignal

	// Determine signal based on both EMAs and RSI
	signal := 0
	if emaDirection > 0 && currentRSI > 50 {
		signal = 1
	} else if emaDirection < 0 && currentRSI < 50 {
		signal = -1
	}

	// Calculate RSI trend
	rsiTrend := 0
	if currentRSI > currentSignal {
		rsiTrend = 1
	} else if currentRSI < currentSignal {
		rsiTrend = -1
	}

	log.Printf("Technical Analysis - EMA8: %.4f, EMA21: %.4f, Direction: %d, RSI: %.2f, Signal: %d",
		ema8, ema21, emaDirection, currentRSI, signal)

	td := &TechnicalData{
		EMA: struct {
			Values    map[int]float64
			Direction int
			Slope     float64
			Strength  float64
		}{
			Values: map[int]float64{
				8:  ema8,
				21: ema21,
			},
			Direction: emaDirection,
			Slope:     emaSlope,
			Strength:  math.Abs(emaSlope),
		},
		RSI: struct {
			Value      float64
			Signal     float64
			Histogram  float64
			Divergence float64
			Trend      int
			Strength   float64
			CrossAbove bool
			CrossBelow bool
		}{
			Value:      currentRSI,
			Signal:     currentSignal,
			Histogram:  currentHistogram,
			Trend:      rsiTrend,
			Strength:   math.Abs(currentRSI-50) / 50,
			CrossAbove: len(rsiResult.RSI) > 1 && rsiResult.RSI[lastIndex-1] <= currentSignal && currentRSI > currentSignal,
			CrossBelow: len(rsiResult.RSI) > 1 && rsiResult.RSI[lastIndex-1] >= currentSignal && currentRSI < currentSignal,
		},
	}

	// Calculate signal and confidence
	td.Signal = signal
	td.Confidence = calculateConfidence(emaDirection, emaSlope, currentRSI)

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

	// RSI confirmation
	if td.RSI.Value > 70 {
		signal = -1
	} else if td.RSI.Value < 30 {
		signal = 1
	}

	return signal
}

func calculateConfidence(emaDirection int, emaSlope float64, rsi float64) float64 {
	confidence := 0.0

	// EMA contribution
	if emaDirection != 0 {
		confidence += 0.4 * math.Abs(emaSlope) * 100
	}

	// RSI contribution
	if rsi > 70 || rsi < 30 {
		confidence += 0.3
	} else if rsi > 60 || rsi < 40 {
		confidence += 0.2
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
