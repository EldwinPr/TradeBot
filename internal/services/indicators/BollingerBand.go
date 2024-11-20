package indicators

import "math"

type BBandsService struct{}

type BBandsResult struct {
	Upper  []float64
	Middle []float64
	Lower  []float64
	Width  []float64 // Volatility indicator
}

func NewBBandsService() *BBandsService {
	return &BBandsService{}
}

func (s *BBandsService) Calculate(prices []float64, period int, deviations float64) *BBandsResult {
	if len(prices) < period {
		return nil
	}

	// Initialize result arrays
	upper := make([]float64, len(prices))
	middle := make([]float64, len(prices))
	lower := make([]float64, len(prices))
	width := make([]float64, len(prices))

	// Calculate SMA and Standard Deviation for each point
	for i := period - 1; i < len(prices); i++ {
		// Get subset of prices for this period
		subset := prices[i-period+1 : i+1]

		// Calculate middle band (SMA)
		sum := 0.0
		for _, price := range subset {
			sum += price
		}
		sma := sum / float64(period)
		middle[i] = sma

		// Calculate standard deviation
		squareSum := 0.0
		for _, price := range subset {
			diff := price - sma
			squareSum += diff * diff
		}
		stdDev := math.Sqrt(squareSum / float64(period))

		// Calculate bands
		upper[i] = sma + (deviations * stdDev)
		lower[i] = sma - (deviations * stdDev)

		// Calculate bandwidth
		width[i] = (upper[i] - lower[i]) / middle[i]
	}

	return &BBandsResult{
		Upper:  upper,
		Middle: middle,
		Lower:  lower,
		Width:  width,
	}
}

// CalculateOne calculates Bollinger Bands for a single point
func (s *BBandsService) CalculateOne(prices []float64, period int, deviations float64) (upper, middle, lower, width float64) {
	if len(prices) < period {
		return 0, 0, 0, 0
	}

	// Calculate SMA
	sum := 0.0
	for _, price := range prices[len(prices)-period:] {
		sum += price
	}
	middle = sum / float64(period)

	// Calculate standard deviation
	squareSum := 0.0
	for _, price := range prices[len(prices)-period:] {
		diff := price - middle
		squareSum += diff * diff
	}
	stdDev := math.Sqrt(squareSum / float64(period))

	// Calculate bands
	upper = middle + (deviations * stdDev)
	lower = middle - (deviations * stdDev)

	// Calculate bandwidth
	width = (upper - lower) / middle

	return upper, middle, lower, width
}

// ValidatePeriod checks if we have enough data
func (s *BBandsService) ValidatePeriod(prices []float64, period int) bool {
	return len(prices) >= period && period > 0
}

// GetStandardPeriods returns commonly used BB periods
func (s *BBandsService) GetStandardPeriods() map[string]struct {
	Period     int
	Deviations float64
} {
	return map[string]struct {
		Period     int
		Deviations float64
	}{
		"default": {20, 2.0},
		"short":   {10, 2.0},
		"long":    {50, 2.0},
		"custom":  {20, 2.5}, // More conservative
	}
}
