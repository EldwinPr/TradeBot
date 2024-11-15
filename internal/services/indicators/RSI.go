package indicators

import "math"

type RSIService struct {
	ema *EMAService
}

func NewRSIService() *RSIService {
	return &RSIService{
		ema: NewEMAService(),
	}
}

func (s *RSIService) Calculate(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return nil
	}

	gains := make([]float64, len(prices))
	losses := make([]float64, len(prices))

	// Calculate price changes and separate gains/losses
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = math.Abs(change)
		}
	}

	// Calculate EMAs of gains and losses
	emaGains := s.ema.Calculate(gains, period)
	emaLosses := s.ema.Calculate(losses, period)

	// Calculate RSI
	rsi := make([]float64, len(prices))
	for i := period; i < len(prices); i++ {
		if emaLosses[i] == 0 {
			rsi[i] = 100
		} else {
			rs := emaGains[i] / emaLosses[i]
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}

	return rsi
}

func (s *RSIService) CalculateOne(currentPrice, prevPrice, prevGainEMA, prevLossEMA float64, period int) float64 {
	var currentGain, currentLoss float64
	change := currentPrice - prevPrice

	if change > 0 {
		currentGain = change
		currentLoss = 0
	} else {
		currentGain = 0
		currentLoss = math.Abs(change)
	}

	gainEMA := s.ema.CalculateOne(currentGain, prevGainEMA, period)
	lossEMA := s.ema.CalculateOne(currentLoss, prevLossEMA, period)

	if lossEMA == 0 {
		return 100
	}

	rs := gainEMA / lossEMA
	return 100 - (100 / (1 + rs))
}

func (s *RSIService) ValidatePeriod(prices []float64, period int) bool {
	return len(prices) >= period+1 && period > 0
}
