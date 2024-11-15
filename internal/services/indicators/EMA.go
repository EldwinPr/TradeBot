package indicators

type EMAService struct{}

func NewEMAService() *EMAService {
	return &EMAService{}
}

func (s *EMAService) Calculate(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}

	multiplier := 2.0 / float64(period+1)
	ema := make([]float64, len(prices))

	// Initial SMA for first EMA value
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// Calculate EMA: EMA = (Price - Previous EMA) × Multiplier + Previous EMA
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
}

// CalculateOne calculates single EMA value using previous EMA
func (s *EMAService) CalculateOne(currentPrice, previousEMA float64, period int) float64 {
	multiplier := 2.0 / float64(period+1)
	return (currentPrice-previousEMA)*multiplier + previousEMA
}

// ValidatePeriod checks if the period is valid for the given prices
func (s *EMAService) ValidatePeriod(prices []float64, period int) bool {
	return len(prices) >= period && period > 0
}
