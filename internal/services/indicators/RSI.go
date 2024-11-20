package indicators

import "math"

type RSIService struct {
	ema *EMAService
}

type RSIResult struct {
	RSI        []float64 // Main RSI line
	Signal     []float64 // Smoothed RSI line (like a signal line)
	Histogram  []float64 // Difference between RSI and signal
	Divergence []float64 // Price/RSI divergence indicator
}

// RSIPoint represents single-point RSI analysis
type RSIPoint struct {
	Value        float64
	Signal       float64
	Histogram    float64
	Trend        int     // 1 (bullish), -1 (bearish), 0 (neutral)
	Strength     float64 // 0-1 based on distance from neutral (50)
	IsOverbought bool
	IsOversold   bool
	CrossAbove   bool // RSI crossed above signal
	CrossBelow   bool // RSI crossed below signal
}

func NewRSIService() *RSIService {
	return &RSIService{
		ema: NewEMAService(),
	}
}

func (s *RSIService) Calculate(prices []float64, period int, smoothPeriod int) *RSIResult {
	if len(prices) < period+1 {
		return nil
	}

	// Initialize arrays
	rsi := make([]float64, len(prices))
	gains := make([]float64, len(prices))
	losses := make([]float64, len(prices))

	// Calculate initial gains and losses
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = math.Abs(change)
		}
	}

	// Calculate EMAs of gains and losses
	avgGain := s.ema.Calculate(gains, period)
	avgLoss := s.ema.Calculate(losses, period)

	// Calculate RSI
	for i := period; i < len(prices); i++ {
		if avgLoss[i] == 0 {
			rsi[i] = 100
		} else {
			rs := avgGain[i] / avgLoss[i]
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}

	// Calculate signal line (smoothed RSI)
	signal := s.ema.Calculate(rsi, smoothPeriod)

	// Calculate histogram
	histogram := make([]float64, len(prices))
	for i := period + smoothPeriod; i < len(prices); i++ {
		histogram[i] = rsi[i] - signal[i]
	}

	// Calculate divergence
	divergence := s.calculateDivergence(prices, rsi)

	return &RSIResult{
		RSI:        rsi,
		Signal:     signal,
		Histogram:  histogram,
		Divergence: divergence,
	}
}

// CalculatePoint provides detailed analysis for the latest point
func (s *RSIService) CalculatePoint(price, prevPrice float64, prevGain, prevLoss float64,
	prevRSI, prevSignal float64, period int, smoothPeriod int) *RSIPoint {

	// Calculate current gain/loss
	var currentGain, currentLoss float64
	change := price - prevPrice
	if change > 0 {
		currentGain = change
		currentLoss = 0
	} else {
		currentGain = 0
		currentLoss = math.Abs(change)
	}

	// Calculate smoothed gain/loss using EMA
	newGain := s.ema.Calculate([]float64{prevGain, currentGain}, period)[1]
	newLoss := s.ema.Calculate([]float64{prevLoss, currentLoss}, period)[1]

	// Calculate RSI
	var rsiValue float64
	if newLoss == 0 {
		rsiValue = 100
	} else {
		rs := newGain / newLoss
		rsiValue = 100 - (100 / (1 + rs))
	}

	// Calculate signal line
	signalValue := s.ema.Calculate([]float64{prevSignal, rsiValue}, smoothPeriod)[1]

	// Calculate histogram
	histogram := rsiValue - signalValue

	// Determine trend and strength
	trend := s.determineTrend(rsiValue, signalValue, histogram)
	strength := s.calculateStrength(rsiValue)

	return &RSIPoint{
		Value:        rsiValue,
		Signal:       signalValue,
		Histogram:    histogram,
		Trend:        trend,
		Strength:     strength,
		IsOverbought: rsiValue >= 70,
		IsOversold:   rsiValue <= 30,
		CrossAbove:   prevRSI <= prevSignal && rsiValue > signalValue,
		CrossBelow:   prevRSI >= prevSignal && rsiValue < signalValue,
	}
}

// GetOptimalParameters returns RSI settings optimized for scalping
func (s *RSIService) GetOptimalParameters() struct {
	Period          int
	SmoothPeriod    int
	OverboughtLevel float64
	OversoldLevel   float64
} {
	return struct {
		Period          int
		SmoothPeriod    int
		OverboughtLevel float64
		OversoldLevel   float64
	}{
		Period:          14,
		SmoothPeriod:    3,
		OverboughtLevel: 70,
		OversoldLevel:   30,
	}
}

// Private helper methods

func (s *RSIService) calculateDivergence(prices, rsi []float64) []float64 {
	divergence := make([]float64, len(prices))

	// Need at least 5 points to calculate meaningful divergence
	if len(prices) < 5 {
		return divergence
	}

	for i := 4; i < len(prices); i++ {
		// Compare price and RSI movements over last 5 candles
		priceDelta := prices[i] - prices[i-4]
		rsiDelta := rsi[i] - rsi[i-4]

		// Normalize and compare movements
		if priceDelta > 0 && rsiDelta < 0 {
			// Bearish divergence
			divergence[i] = -1
		} else if priceDelta < 0 && rsiDelta > 0 {
			// Bullish divergence
			divergence[i] = 1
		}
	}

	return divergence
}

func (s *RSIService) determineTrend(rsi, signal, histogram float64) int {
	if rsi > signal && histogram > 0 {
		return 1 // Bullish
	} else if rsi < signal && histogram < 0 {
		return -1 // Bearish
	}
	return 0 // Neutral
}

func (s *RSIService) calculateStrength(rsi float64) float64 {
	// Convert RSI to strength (0-1)
	// Maximum strength at extremes (0 or 100)
	if rsi >= 50 {
		return (rsi - 50) / 50
	}
	return (50 - rsi) / 50
}
