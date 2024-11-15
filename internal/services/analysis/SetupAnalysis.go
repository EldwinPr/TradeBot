package analysis

import "errors"

type PositionDetails struct {
	Size     float64
	Leverage int
	CostUSD  float64
}

type SetupAnalysis struct {
	riskPerTrade float64 // $1-2 per trade
	btcLeverage  int     // 75-100x
	altLeverage  int     // 50x
}

type SetupResult struct {
	PreTradeResult *AnalysisResult
	Leverage       int
	PositionSize   float64 // In asset units
	CostUSD        float64 // Always around $1-2
	StopLoss       float64
	TakeProfit     float64 // Changed from TakeProfits []float64 to single TakeProfit
}

// NewSetupAnalysis creates a new setup analyzer with default values
func NewSetupAnalysis(riskPerTrade float64, btcLeverage, altLeverage int) *SetupAnalysis {
	return &SetupAnalysis{
		riskPerTrade: riskPerTrade,
		btcLeverage:  btcLeverage,
		altLeverage:  altLeverage,
	}
}

// AnalyzeSetup performs full setup analysis including position sizing
func (s *SetupAnalysis) AnalyzeSetup(preTrade *AnalysisResult, lastPrice float64) (*SetupResult, error) {
	if preTrade == nil {
		return nil, errors.New("pre-trade analysis result is required")
	}

	if lastPrice <= 0 {
		return nil, errors.New("invalid last price")
	}

	position := s.calculatePosition(preTrade, lastPrice)

	// Calculate targets based on position
	stopLoss := preTrade.StopLoss
	takeProfit := preTrade.TakeProfits[0] * position.Size // Changed to single takeProfit

	return &SetupResult{
		PreTradeResult: preTrade,
		Leverage:       position.Leverage,
		PositionSize:   position.Size,
		CostUSD:        position.CostUSD,
		StopLoss:       stopLoss,
		TakeProfit:     takeProfit,
	}, nil
}

// ValidateSetup checks if the setup meets minimum criteria
func (s *SetupAnalysis) ValidateSetup(setup *SetupResult) bool {
	if setup == nil || setup.PreTradeResult == nil {
		return false
	}

	// Check minimum probability and risk ratio
	if setup.PreTradeResult.Probability < MinProbability {
		return false
	}

	if setup.PreTradeResult.RiskRatio < MinRiskRatio {
		return false
	}

	// Validate position size and leverage
	if setup.PositionSize <= 0 || setup.Leverage <= 0 {
		return false
	}

	return true
}

func (s *SetupAnalysis) calculatePosition(preTrade *AnalysisResult, lastPrice float64) PositionDetails {
	leverage := s.getLeverage(preTrade.Symbol)

	// Calculate position size based on fixed cost
	positionSize := (s.riskPerTrade * float64(leverage)) / lastPrice

	return PositionDetails{
		Size:     positionSize,
		Leverage: leverage,
		CostUSD:  s.riskPerTrade,
	}
}

func (s *SetupAnalysis) getLeverage(symbol string) int {
	switch symbol {
	case "BTCUSDT", "ETHUSDT":
		return s.btcLeverage
	default:
		return s.altLeverage
	}
}
