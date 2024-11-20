package indicators

import "math"

// EMAService provides Exponential Moving Average calculations
type EMAService struct {
	maxPeriod int       // Track longest period for validation
	values    []float64 // Store calculated values for reuse
}

// EMAResult holds single-point calculation results
type EMAResult struct {
	Value     float64
	Slope     float64 // Rate of change
	Direction int     // 1 (up), -1 (down), 0 (flat)
	Strength  float64 // 0-1 based on slope
}

// CrossSignal represents EMA crossover status
type CrossSignal struct {
	Crossed   bool    // Whether cross occurred
	Direction int     // 1 (bullish), -1 (bearish)
	Strength  float64 // Strength of crossover
}

// NewEMAService creates a new EMA service instance
func NewEMAService() *EMAService {
	return &EMAService{}
}

// Calculate computes EMA for the entire price series
func (s *EMAService) Calculate(prices []float64, period int) []float64 {
	// Validation
	if !s.validateInputs(prices, period) {
		return nil
	}

	// Initialize result array
	ema := make([]float64, len(prices))
	s.values = ema // Store for potential reuse

	// Calculate smoothing factor
	multiplier := s.getMultiplier(period)

	// Calculate initial SMA
	sma := s.calculateInitialSMA(prices, period)
	ema[period-1] = sma

	// Calculate EMA for remaining points
	for i := period; i < len(prices); i++ {
		ema[i] = s.calculatePoint(prices[i], ema[i-1], multiplier)
	}

	return ema
}

// CalculatePoint calculates EMA for a single point with additional metrics
func (s *EMAService) CalculatePoint(currentPrice, prevEMA float64, period int) *EMAResult {
	if period <= 0 {
		return nil
	}

	multiplier := s.getMultiplier(period)
	emaValue := s.calculatePoint(currentPrice, prevEMA, multiplier)

	// Calculate slope and direction
	slope := (emaValue - prevEMA) / prevEMA
	direction := s.determineDirection(slope)
	strength := s.calculateStrength(slope)

	return &EMAResult{
		Value:     emaValue,
		Slope:     slope,
		Direction: direction,
		Strength:  strength,
	}
}

// CheckCrossover detects and analyzes EMA crossovers
func (s *EMAService) CheckCrossover(fastEMA, slowEMA []float64) *CrossSignal {
	if len(fastEMA) < 2 || len(slowEMA) < 2 {
		return &CrossSignal{Crossed: false}
	}

	// Get current and previous values
	currFast := fastEMA[len(fastEMA)-1]
	prevFast := fastEMA[len(fastEMA)-2]
	currSlow := slowEMA[len(slowEMA)-1]
	prevSlow := slowEMA[len(slowEMA)-2]

	// Check for crossover
	bullishCross := prevFast <= prevSlow && currFast > currSlow
	bearishCross := prevFast >= prevSlow && currFast < currSlow

	if !bullishCross && !bearishCross {
		return &CrossSignal{Crossed: false}
	}

	// Calculate crossover strength
	strength := math.Abs((currFast - currSlow) / currSlow)
	direction := 1
	if bearishCross {
		direction = -1
	}

	return &CrossSignal{
		Crossed:   true,
		Direction: direction,
		Strength:  strength,
	}
}

// GetStandardPeriods returns commonly used EMA periods
func (s *EMAService) GetStandardPeriods() map[string][]int {
	return map[string][]int{
		"scalping": {8, 21},
		"intraday": {9, 20},
		"swing":    {12, 26},
		"position": {50, 200},
	}
}

// Private helper methods

func (s *EMAService) validateInputs(prices []float64, period int) bool {
	if len(prices) == 0 || period <= 0 || len(prices) < period {
		return false
	}
	return true
}

func (s *EMAService) getMultiplier(period int) float64 {
	return 2.0 / float64(period+1)
}

func (s *EMAService) calculateInitialSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func (s *EMAService) calculatePoint(price, prevEMA, multiplier float64) float64 {
	return (price-prevEMA)*multiplier + prevEMA
}

func (s *EMAService) determineDirection(slope float64) int {
	if slope > 0.0001 { // Small threshold to avoid noise
		return 1
	} else if slope < -0.0001 {
		return -1
	}
	return 0
}

func (s *EMAService) calculateStrength(slope float64) float64 {
	// Convert slope to strength value between 0-1
	// Higher absolute slope = stronger trend
	return math.Min(math.Abs(slope)*100, 1.0)
}
