package strategy

import (
	"CryptoTradeBot/internal/models"
)

type StrategyManager struct {
	long  *LongStrategy
	short *ShortStrategy
	// Minimum confidence difference needed for position reversal
	reversalDelta float64
}

func NewStrategyManager() *StrategyManager {
	return &StrategyManager{
		long:          NewLongStrategy(),
		short:         NewShortStrategy(),
		reversalDelta: 0.1, // 10% higher confidence needed for reversal
	}
}

func (m *StrategyManager) Analyze(
	position *models.Position,
	prices5m, prices15m, prices1h, prices4h []models.Price,
) (*StrategyResult, error) {
	// If no position, analyze both directions
	if position == nil {
		return m.analyzeNewPosition(prices5m, prices15m, prices1h, prices4h)
	}

	// If position exists, check for potential reversal
	return m.analyzeReversal(position, prices5m, prices15m, prices1h, prices4h)
}

func (m *StrategyManager) analyzeNewPosition(
	prices5m, prices15m, prices1h, prices4h []models.Price,
) (*StrategyResult, error) {
	// Get long and short analysis
	longResult, err := m.long.Analyze(prices5m, prices15m, prices1h, prices4h)
	if err != nil {
		return nil, err
	}

	shortResult, err := m.short.Analyze(prices5m, prices15m, prices1h, prices4h)
	if err != nil {
		return nil, err
	}

	// Neither strategy valid
	if !longResult.IsValid && !shortResult.IsValid {
		return &StrategyResult{
			IsValid: false,
			Reason:  "no valid setup found",
		}, nil
	}

	// Return the higher confidence strategy
	if longResult.IsValid && shortResult.IsValid {
		if longResult.Confidence > shortResult.Confidence {
			return longResult, nil
		}
		return shortResult, nil
	}

	// Return whichever is valid
	if longResult.IsValid {
		return longResult, nil
	}
	return shortResult, nil
}

func (m *StrategyManager) analyzeReversal(
	position *models.Position,
	prices5m, prices15m, prices1h, prices4h []models.Price,
) (*StrategyResult, error) {
	// Check opposite direction of current position
	var result *StrategyResult
	var err error

	if position.Side == "long" {
		result, err = m.short.Analyze(prices5m, prices15m, prices1h, prices4h)
	} else {
		result, err = m.long.Analyze(prices5m, prices15m, prices1h, prices4h)
	}

	if err != nil {
		return nil, err
	}

	// No valid reversal setup
	if !result.IsValid {
		return &StrategyResult{
			IsValid: false,
			Reason:  "no reversal setup found",
		}, nil
	}

	// Check if reversal confidence is significantly higher
	// We require higher confidence for reversals to avoid unnecessary switches
	if result.Confidence > position.Confidence+m.reversalDelta {
		return result, nil
	}

	// Not enough confidence for reversal
	return &StrategyResult{
		IsValid: false,
		Reason:  "insufficient confidence for reversal",
	}, nil
}

// Optional helper to get underlying strategies
func (m *StrategyManager) GetStrategies() (*LongStrategy, *ShortStrategy) {
	return m.long, m.short
}

// Optional helper to update reversal threshold
func (m *StrategyManager) SetReversalDelta(delta float64) {
	m.reversalDelta = delta
}
