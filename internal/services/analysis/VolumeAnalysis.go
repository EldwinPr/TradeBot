package analysis

import (
	"CryptoTradeBot/internal/models"
	"fmt"
	"math"
)

type VolumeAnalyzer struct {
	weights map[string]float64
}

func NewVolumeAnalyzer() *VolumeAnalyzer {
	return &VolumeAnalyzer{
		weights: map[string]float64{
			models.PriceTimeFrame5m:  0.35,
			models.PriceTimeFrame15m: 0.45,
			models.PriceTimeFrame1h:  0.20,
		},
	}
}

func (a *VolumeAnalyzer) Analyze(prices5m, prices15m, prices1h []models.Price) (*VolumeData, error) {
	m5, err := a.analyzeTimeframe(prices5m, 12)
	if err != nil {
		return nil, err
	}

	m15, err := a.analyzeTimeframe(prices15m, 12)
	if err != nil {
		return nil, err
	}

	m1h, err := a.analyzeTimeframe(prices1h, 6)
	if err != nil {
		return nil, err
	}

	weightedConfidence := (m5.confidence * a.weights[models.PriceTimeFrame5m]) +
		(m15.confidence * a.weights[models.PriceTimeFrame15m]) +
		(m1h.confidence * a.weights[models.PriceTimeFrame1h])

	return &VolumeData{
		VolumeRatio:  m5.volumeRatio,
		TradeCount:   m5.tradeCount,
		AvgTradeSize: m5.avgTradeSize,
		Confidence:   weightedConfidence,
	}, nil
}

func (a *VolumeAnalyzer) analyzeTimeframe(prices []models.Price, window int) (*timeframeVolumeMetrics, error) {
	if len(prices) < window {
		return nil, fmt.Errorf("insufficient data, need %d bars", window)
	}

	recent := prices[len(prices)-window:]
	current := recent[len(recent)-1]

	// Calculate volume trend
	volumeTrend := 0.0
	for i := 1; i < len(recent); i++ {
		if recent[i].Volume > recent[i-1].Volume {
			volumeTrend++
		}
	}
	trendStrength := volumeTrend / float64(len(recent)-1)

	// Progressive volume weighting
	weightedVolume := 0.0
	totalWeight := 0.0
	for i := range recent {
		weight := math.Pow(1.1, float64(i)) // Exponential weight
		weightedVolume += recent[i].Volume * weight
		totalWeight += weight
	}
	avgVolume := weightedVolume / totalWeight

	// Current metrics
	volumeRatio := current.Volume / avgVolume
	tradeRatio := float64(current.TradeCount) / float64(recent[len(recent)-2].TradeCount)
	avgTradeSize := current.Volume / float64(current.TradeCount)

	// Calculate confidence with trend incorporation
	confidence := a.calculateConfidence(volumeRatio, tradeRatio, avgTradeSize, trendStrength)

	return &timeframeVolumeMetrics{
		volumeRatio:    volumeRatio,
		tradeCount:     current.TradeCount,
		avgTradeSize:   avgTradeSize,
		confidence:     confidence,
		volumeTrend:    volumeTrend,
		trendStrength:  trendStrength,
		weightedVolume: weightedVolume,
	}, nil
}

func (a *VolumeAnalyzer) calculateConfidence(volumeRatio, tradeRatio, avgTradeSize, trendStrength float64) float64 {
	minVolumeRatio := 1.3
	minTradeRatio := 1.2

	volComponent := math.Max(0, math.Min((volumeRatio-minVolumeRatio)/(2-minVolumeRatio), 1.0))
	tradeComponent := math.Max(0, math.Min((tradeRatio-minTradeRatio)/(2-minTradeRatio), 1.0))
	sizeComponent := math.Min(avgTradeSize/1000.0, 1.0)

	// Include trend strength in calculation
	trendComponent := trendStrength * 0.3

	return (volComponent * 0.4) + (tradeComponent * 0.2) + (sizeComponent * 0.1) + trendComponent
}
