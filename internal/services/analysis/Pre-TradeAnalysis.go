package analysis

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/services/indicators"
	"fmt"
	"math"
	"time"
)

type PreTradeAnalysis struct {
	rsi  *indicators.RSIService
	ema  *indicators.EMAService
	macd *indicators.MACDService
}

type AnalysisResult struct {
	TradeType     string // "long" or "short"
	Symbol        string
	EntryPrice    float64
	StopLoss      float64
	TakeProfits   []float64
	Probability   float64 // 0-1
	RiskRatio     float64
	TrendStrength int // 1-5
	Time          time.Time
	Message       string
}

const (
	TradeLong  = "long"
	TradeShort = "short"

	MinProbability = 0.6
	MinRiskRatio   = 2.0

	TP1 = 0.50 // 50% ROI target
	TP2 = 0.75 // 75% ROI target
	TP3 = 1.00 // 100% ROI target
)

func NewPreTradeAnalysis() *PreTradeAnalysis {
	return &PreTradeAnalysis{
		rsi:  indicators.NewRSIService(),
		ema:  indicators.NewEMAService(),
		macd: indicators.NewMACDService(),
	}
}

func (a *PreTradeAnalysis) AnalyzeSetup(symbol string, prices []models.Price) *AnalysisResult {
	if len(prices) < 50 {
		return nil
	}

	// Get latest price and indicators
	current := a.getPriceData(prices)

	// Check long setup
	if longSetup := a.analyzeLongSetup(symbol, current); longSetup != nil {
		return longSetup
	}

	// Check short setup
	if shortSetup := a.analyzeShortSetup(symbol, current); shortSetup != nil {
		return shortSetup
	}

	return nil
}

type priceData struct {
	price     models.Price
	rsi       float64
	ema20     float64
	ema50     float64
	macd      float64
	signal    float64
	histogram float64
}

func (a *PreTradeAnalysis) getPriceData(prices []models.Price) priceData {
	closePrices := make([]float64, len(prices))
	for i, p := range prices {
		closePrices[i] = p.Close
	}

	rsiValues := a.rsi.Calculate(closePrices, 14)
	ema20 := a.ema.Calculate(closePrices, 20)
	ema50 := a.ema.Calculate(closePrices, 50)
	macdResult := a.macd.Calculate(closePrices, 12, 26, 9)

	last := len(prices) - 1
	return priceData{
		price:     prices[last],
		rsi:       rsiValues[last],
		ema20:     ema20[last],
		ema50:     ema50[last],
		macd:      macdResult.MACD[last],
		signal:    macdResult.Signal[last],
		histogram: macdResult.Histogram[last],
	}
}

func (a *PreTradeAnalysis) analyzeLongSetup(symbol string, data priceData) *AnalysisResult {
	// Check long conditions
	if !isLongSetup(data) {
		return nil
	}

	probability := calculateLongProbability(data)
	riskRatio := calculateLongRiskRatio(data)
	trendStrength := calculateLongTrendStrength(data)

	if probability < MinProbability || riskRatio < MinRiskRatio {
		return nil
	}

	return &AnalysisResult{
		TradeType:     TradeLong,
		Symbol:        symbol,
		EntryPrice:    data.price.Close,
		StopLoss:      calculateLongStopLoss(data),
		TakeProfits:   calculateLongTakeProfits(data),
		Probability:   probability,
		RiskRatio:     riskRatio,
		TrendStrength: trendStrength,
		Time:          data.price.CloseTime,
		Message:       buildLongMessage(data, probability, riskRatio),
	}
}

func (a *PreTradeAnalysis) analyzeShortSetup(symbol string, data priceData) *AnalysisResult {
	if !isShortSetup(data) {
		return nil
	}

	probability := calculateShortProbability(data)
	riskRatio := calculateShortRiskRatio(data)
	trendStrength := calculateShortTrendStrength(data)

	if probability < MinProbability || riskRatio < MinRiskRatio {
		return nil
	}

	return &AnalysisResult{
		TradeType:     TradeShort,
		Symbol:        symbol,
		EntryPrice:    data.price.Close,
		StopLoss:      calculateShortStopLoss(data),
		TakeProfits:   calculateShortTakeProfits(data),
		Probability:   probability,
		RiskRatio:     riskRatio,
		TrendStrength: trendStrength,
		Time:          data.price.CloseTime,
		Message:       buildShortMessage(data, probability, riskRatio),
	}
}

func isLongSetup(data priceData) bool {
	return data.rsi < 30 && // Oversold
		data.ema20 > data.ema50 && // Uptrend
		data.macd > data.signal && // Bullish MACD
		data.histogram > 0
}

func isShortSetup(data priceData) bool {
	return data.rsi > 70 && // Overbought
		data.ema20 < data.ema50 && // Downtrend
		data.macd < data.signal && // Bearish MACD
		data.histogram < 0
}

func calculateLongProbability(data priceData) float64 {
	prob := 0.5

	if data.rsi < 30 {
		prob += 0.1
	}
	if data.rsi < 20 {
		prob += 0.1
	}
	if data.ema20 > data.ema50 {
		prob += 0.1
	}
	if data.macd > data.signal && data.histogram > 0 {
		prob += 0.1
	}

	return prob
}

func calculateShortProbability(data priceData) float64 {
	prob := 0.5

	if data.rsi > 70 {
		prob += 0.1
	}
	if data.rsi > 80 {
		prob += 0.1
	}
	if data.ema20 < data.ema50 {
		prob += 0.1
	}
	if data.macd < data.signal && data.histogram < 0 {
		prob += 0.1
	}

	return prob
}

func calculateLongRiskRatio(data priceData) float64 {
	atr := calculateATR(data.price) // You'll need to implement ATR calculation
	stopLoss := data.price.Close - (2 * atr)
	takeProfit := data.price.Close + (4 * atr)

	risk := data.price.Close - stopLoss
	reward := takeProfit - data.price.Close

	if risk == 0 {
		return 0
	}
	return reward / risk
}

func calculateShortRiskRatio(data priceData) float64 {
	atr := calculateATR(data.price) // You'll need to implement ATR calculation
	stopLoss := data.price.Close + (2 * atr)
	takeProfit := data.price.Close - (4 * atr)

	risk := stopLoss - data.price.Close
	reward := data.price.Close - takeProfit

	if risk == 0 {
		return 0
	}
	return reward / risk
}

func calculateLongTakeProfits(data priceData) []float64 {
	return []float64{
		data.price.Close * (1 + TP1),
		data.price.Close * (1 + TP2),
		data.price.Close * (1 + TP3),
	}
}

func calculateShortTakeProfits(data priceData) []float64 {
	return []float64{
		data.price.Close * (1 - TP1),
		data.price.Close * (1 - TP2),
		data.price.Close * (1 - TP3),
	}
}

func calculateLongStopLoss(data priceData) float64 {
	atr := calculateATR(data.price)
	return data.price.Close - (2 * atr)
}

func calculateShortStopLoss(data priceData) float64 {
	atr := calculateATR(data.price)
	return data.price.Close + (2 * atr)
}

func buildLongMessage(data priceData, prob, rr float64) string {
	return fmt.Sprintf("Long setup: RSI(%0.2f) oversold, EMA bullish, MACD positive. Probability: %0.2f, R/R: %0.2f",
		data.rsi, prob, rr)
}

func buildShortMessage(data priceData, prob, rr float64) string {
	return fmt.Sprintf("Short setup: RSI(%0.2f) overbought, EMA bearish, MACD negative. Probability: %0.2f, R/R: %0.2f",
		data.rsi, prob, rr)
}

func calculateLongTrendStrength(data priceData) int {
	strength := 1

	// EMA trend strength
	if data.ema20 > data.ema50 {
		strength++
		if (data.ema20/data.ema50 - 1) > 0.02 {
			strength++
		}
	}

	// MACD strength
	if data.macd > data.signal && data.histogram > 0 {
		strength++
		if data.histogram > data.macd*0.5 {
			strength++
		}
	}

	return strength
}

func calculateShortTrendStrength(data priceData) int {
	strength := 1

	// EMA trend strength
	if data.ema20 < data.ema50 {
		strength++
		if (1 - data.ema20/data.ema50) > 0.02 {
			strength++
		}
	}

	// MACD strength
	if data.macd < data.signal && data.histogram < 0 {
		strength++
		if math.Abs(data.histogram) > math.Abs(data.macd)*0.5 {
			strength++
		}
	}

	return strength
}

// You'll need to implement this
func calculateATR(price models.Price) float64 {
	// Implement ATR calculation
	return 0
}
