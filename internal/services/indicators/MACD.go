package indicators

type MACDService struct {
	ema *EMAService
}

type MACDResult struct {
	MACD      []float64
	Signal    []float64
	Histogram []float64
}

func NewMACDService() *MACDService {
	return &MACDService{
		ema: NewEMAService(),
	}
}

// Calculate returns MACD line, signal line, and histogram
// Default periods: fast=12, slow=26, signal=9
func (s *MACDService) Calculate(prices []float64, fastPeriod, slowPeriod, signalPeriod int) *MACDResult {
	if !s.ValidatePeriods(prices, fastPeriod, slowPeriod, signalPeriod) {
		return nil
	}

	// Calculate fast and slow EMAs
	fastEMA := s.ema.Calculate(prices, fastPeriod)
	slowEMA := s.ema.Calculate(prices, slowPeriod)

	// Calculate MACD line (fast EMA - slow EMA)
	macdLine := make([]float64, len(prices))
	for i := slowPeriod - 1; i < len(prices); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// Calculate signal line (EMA of MACD line)
	signalLine := s.ema.Calculate(macdLine, signalPeriod)

	// Calculate histogram (MACD line - signal line)
	histogram := make([]float64, len(prices))
	for i := slowPeriod + signalPeriod - 2; i < len(prices); i++ {
		histogram[i] = macdLine[i] - signalLine[i]
	}

	return &MACDResult{
		MACD:      macdLine,
		Signal:    signalLine,
		Histogram: histogram,
	}
}

// CalculateOne calculates single MACD value using previous values
func (s *MACDService) CalculateOne(currentPrice float64, prevFastEMA, prevSlowEMA, prevSignal float64,
	fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64) {

	// Calculate new EMAs
	newFastEMA := s.ema.CalculateOne(currentPrice, prevFastEMA, fastPeriod)
	newSlowEMA := s.ema.CalculateOne(currentPrice, prevSlowEMA, slowPeriod)

	// Calculate MACD
	macd := newFastEMA - newSlowEMA

	// Calculate signal line
	signal := s.ema.CalculateOne(macd, prevSignal, signalPeriod)

	// Calculate histogram
	histogram := macd - signal

	return macd, signal, histogram
}

func (s *MACDService) ValidatePeriods(prices []float64, fastPeriod, slowPeriod, signalPeriod int) bool {
	minLength := slowPeriod + signalPeriod - 1
	return len(prices) >= minLength &&
		fastPeriod > 0 &&
		slowPeriod > fastPeriod &&
		signalPeriod > 0
}
