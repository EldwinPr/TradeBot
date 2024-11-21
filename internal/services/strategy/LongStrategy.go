package strategy

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/analysis"
	"fmt"
	"math"
)

type LongStrategy struct {
	// Core settings
	targetProfit  float64 // 1%
	stopLoss      float64 // 0.6%
	minConfidence float64 // Minimum confidence for entry

	// Analysis weights
	volumeWeight    float64 // 30%
	technicalWeight float64 // 35%
	priceWeight     float64 // 35%

	// Analysis services
	volumeAnalyzer    *analysis.VolumeAnalyzer
	technicalAnalyzer *analysis.TechnicalAnalyzer
	priceAnalyzer     *analysis.PriceAnalyzer
}

func NewLongStrategy() *LongStrategy {
	return &LongStrategy{
		targetProfit:      0.01,
		stopLoss:          0.006,
		minConfidence:     0.7,
		volumeWeight:      0.30,
		technicalWeight:   0.35,
		priceWeight:       0.35,
		volumeAnalyzer:    analysis.NewVolumeAnalyzer(),
		technicalAnalyzer: analysis.NewTechnicalAnalyzer(),
		priceAnalyzer:     analysis.NewPriceAnalyzer(),
	}
}

// Main analysis function
func (s *LongStrategy) Analyze(prices5m, prices15m, prices1h, prices4h []models.Price) (*StrategyResult, error) {
	// Get all analysis results
	volAnalysis, err := s.volumeAnalyzer.Analyze(prices5m, prices15m, prices1h)
	if err != nil {
		return nil, fmt.Errorf("volume analysis failed: %w", err)
	}

	techAnalysis, err := s.technicalAnalyzer.Analyze(prices5m, prices15m, prices1h, prices4h)
	if err != nil {
		return nil, fmt.Errorf("technical analysis failed: %w", err)
	}

	priceAnalysis, err := s.priceAnalyzer.Analyze(prices5m, prices15m, prices1h, prices4h)
	if err != nil {
		return nil, fmt.Errorf("price analysis failed: %w", err)
	}

	// Check if long conditions are met
	if !s.validateLongSetup(volAnalysis, techAnalysis, priceAnalysis) {
		return s.newInvalidResult("conditions not met"), nil
	}

	// Calculate overall confidence
	confidence := s.calculateConfidence(volAnalysis, techAnalysis, priceAnalysis)
	if confidence < s.minConfidence {
		return s.newInvalidResult("low confidence"), nil
	}

	// Get current price from most recent 5m candle
	currentPrice := prices5m[len(prices5m)-1].Close

	return &StrategyResult{
		IsValid:    true,
		Direction:  "long",
		EntryPrice: currentPrice,
		StopLoss:   currentPrice * (1 - s.stopLoss),
		TakeProfit: currentPrice * (1 + s.targetProfit),
		Confidence: confidence,
		Volume:     *volAnalysis,
		Technical:  *techAnalysis,
		Price:      *priceAnalysis,
	}, nil
}

// Validate long setup conditions
func (s *LongStrategy) validateLongSetup(vol *analysis.VolumeData, tech *analysis.TechnicalData, price *analysis.PriceData) bool {
	// Just check basic conditions for possible setup
	technicalValid := tech.RSI.Value < 75 && tech.RSI.Value > 25 // Wide RSI range
	priceValid := tech.EMA.Direction >= 0 || tech.EMA.Slope > 0  // Any upward movement
	volumeValid := vol.VolumeRatio > 0.45                        // Basic volume check

	return technicalValid && priceValid && volumeValid
}

// Calculate overall confidence score
func (s *LongStrategy) calculateConfidence(vol *analysis.VolumeData, tech *analysis.TechnicalData, price *analysis.PriceData) float64 {
	// Base confidence much lower
	confidence := 0.3 // Start with 30% base

	// Add small boosts for good conditions
	if vol.VolumeRatio > 1.0 {
		confidence += 0.1
	}
	if tech.EMA.Direction > 0 {
		confidence += 0.1
	}
	if tech.RSI.Value > 40 && tech.RSI.Value < 60 {
		confidence += 0.1
	}

	s.minConfidence = 0.3 // Lower minimum confidence threshold

	return math.Min(confidence, 1.0)
}

// Apply confidence modifiers based on conditions
func (s *LongStrategy) applyModifiers(base float64, vol *analysis.VolumeData, tech *analysis.TechnicalData, price *analysis.PriceData) float64 {
	conf := base

	// Volume modifiers
	if vol.VolumeRatio > 2.0 {
		conf *= 1.1 // Boost confidence on very high volume
	}

	// Technical modifiers
	if tech.RSI.CrossAbove && tech.EMA.Direction > 0 {
		conf *= 1.1 // Boost on RSI cross with trend
	}

	// Price modifiers
	if price.Volatility > 0.8 {
		conf *= 0.9 // Reduce confidence in high volatility
	}

	return conf
}

// Create invalid result
func (s *LongStrategy) newInvalidResult(reason string) *StrategyResult {
	return &StrategyResult{
		IsValid:    false,
		Reason:     reason,
		Confidence: 0,
	}
}
