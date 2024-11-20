package strategy

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/analysis"
	"fmt"
	"math"
)

type ShortStrategy struct {
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

func NewShortStrategy() *ShortStrategy {
	return &ShortStrategy{
		targetProfit:      0.01,
		stopLoss:          0.006,
		minConfidence:     0.70,
		volumeWeight:      0.30,
		technicalWeight:   0.35,
		priceWeight:       0.35,
		volumeAnalyzer:    analysis.NewVolumeAnalyzer(),
		technicalAnalyzer: analysis.NewTechnicalAnalyzer(),
		priceAnalyzer:     analysis.NewPriceAnalyzer(),
	}
}

func (s *ShortStrategy) Analyze(prices5m, prices15m, prices1h, prices4h []models.Price) (*StrategyResult, error) {
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

	// Check if short conditions are met
	if !s.validateShortSetup(volAnalysis, techAnalysis, priceAnalysis) {
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
		Direction:  "short",
		EntryPrice: currentPrice,
		StopLoss:   currentPrice * (1 + s.stopLoss),     // Note: Reversed for short
		TakeProfit: currentPrice * (1 - s.targetProfit), // Note: Reversed for short
		Confidence: confidence,
		Volume:     *volAnalysis,
		Technical:  *techAnalysis,
		Price:      *priceAnalysis,
	}, nil
}

func (s *ShortStrategy) validateShortSetup(vol *analysis.VolumeData, tech *analysis.TechnicalData, price *analysis.PriceData) bool {
	// Volume conditions
	volumeValid := vol.VolumeRatio > 1.2 && vol.Confidence > 0.6

	// Technical conditions - Note reversed conditions from long
	technicalValid := tech.EMA.Direction < 0 &&
		tech.RSI.Value < 60 && tech.RSI.Value > 30 &&
		tech.Signal < 0

	// Price conditions - Note reversed from long
	priceValid := price.Momentum < 0 && price.Signal < 0

	return volumeValid && technicalValid && priceValid
}

func (s *ShortStrategy) calculateConfidence(vol *analysis.VolumeData, tech *analysis.TechnicalData, price *analysis.PriceData) float64 {
	volScore := vol.Confidence * s.volumeWeight
	techScore := tech.Confidence * s.technicalWeight
	priceScore := price.Confidence * s.priceWeight

	// Base confidence
	confidence := volScore + techScore + priceScore

	// Apply modifiers
	confidence = s.applyModifiers(confidence, vol, tech, price)

	return math.Min(confidence, 1.0)
}

func (s *ShortStrategy) applyModifiers(base float64, vol *analysis.VolumeData, tech *analysis.TechnicalData, price *analysis.PriceData) float64 {
	conf := base

	// Volume modifiers
	if vol.VolumeRatio > 2.0 {
		conf *= 1.1 // Boost confidence on very high volume
	}

	// Technical modifiers
	if tech.RSI.CrossBelow && tech.EMA.Direction < 0 {
		conf *= 1.1 // Boost on RSI cross with downtrend
	}

	// Price modifiers
	if price.Volatility > 0.8 {
		conf *= 0.9 // Reduce confidence in high volatility
	}

	return conf
}

func (s *ShortStrategy) newInvalidResult(reason string) *StrategyResult {
	return &StrategyResult{
		IsValid:    false,
		Reason:     reason,
		Confidence: 0,
	}
}
