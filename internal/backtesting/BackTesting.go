package backtesting

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/analysis"
	"log"
	"math"
	"sort"
	"time"
)

const (
	InitialBalance = 10.0 // USDT
	Leverage       = 50   // 50x leverage
	FixedSize      = 1.0  // $10 per trade
)

type Trade struct {
	Symbol     string
	EntryTime  time.Time
	ExitTime   time.Time
	Side       string
	EntryPrice float64
	ExitPrice  float64
	Size       float64
	StopLoss   float64
	TakeProfit float64
	PnL        float64
	Reason     string
}

type EquityPoint struct {
	Timestamp time.Time
	Balance   float64
}

type BacktestResults struct {
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	AveragePnL    float64
	MaxDrawdown   float64
	FinalBalance  float64
	SharpeRatio   float64
	Trades        []Trade
	EquityCurve   []EquityPoint
}

type Backtest struct {
	priceRepo      *repositories.PriceRepository
	analysis       *analysis.Analysis
	currentBalance float64
	maxBalance     float64
	trades         []Trade
	equityCurve    []EquityPoint
}

func NewBacktest(priceRepo *repositories.PriceRepository, analysis *analysis.Analysis) *Backtest {
	return &Backtest{
		priceRepo:      priceRepo,
		analysis:       analysis,
		currentBalance: InitialBalance,
		maxBalance:     InitialBalance,
		trades:         make([]Trade, 0),
		equityCurve:    make([]EquityPoint, 0),
	}
}

func (b *Backtest) RunBacktest(startTime, endTime time.Time, symbols []string) (*BacktestResults, error) {
	log.Printf("Running backtest from %s to %s",
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))

	for _, symbol := range symbols {
		log.Printf("Processing %s...", symbol)
		if err := b.runSymbol(symbol, startTime, endTime); err != nil {
			return nil, err
		}
	}

	results := b.calculateResults()
	log.Printf("Processed %d days of data", int(endTime.Sub(startTime).Hours()/24))

	return results, nil
}

func (b *Backtest) runSymbol(symbol string, startTime, endTime time.Time) error {
	// Get all prices for the period
	prices, err := b.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame5m, startTime, endTime)
	if err != nil {
		return err
	}

	if len(prices) < 200 {
		log.Printf("Not enough data for %s, skipping", symbol)
		return nil
	}

	// Sort prices by time to ensure chronological order
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].OpenTime.Before(prices[j].OpenTime)
	})

	var activePosition *Trade

	// Process each candle for the entire period
	for i := 200; i < len(prices); i++ {
		currentPrice := prices[i]

		// Skip if outside our date range
		if currentPrice.OpenTime.Before(startTime) || currentPrice.OpenTime.After(endTime) {
			continue
		}

		if activePosition != nil {
			if b.shouldExitPosition(activePosition, currentPrice) {
				reason := b.getExitReason(activePosition, currentPrice)
				b.closePosition(activePosition, currentPrice, reason)
				activePosition = nil
			}
			continue
		}

		// Analysis window
		analysisWindow := prices[i-200 : i+1]
		result := b.analysis.Analyze(analysisWindow)

		if result.IsValid {
			activePosition = b.openPosition(result, currentPrice)
		}
	}

	return nil
}

func (b *Backtest) shouldExitPosition(trade *Trade, price models.Price) bool {
	if trade.Side == "long" {
		return price.High >= trade.TakeProfit || price.Low <= trade.StopLoss
	}
	return price.Low <= trade.TakeProfit || price.High >= trade.StopLoss
}

func (b *Backtest) getExitReason(trade *Trade, price models.Price) string {
	if trade.Side == "long" {
		if price.High >= trade.TakeProfit {
			return "take_profit"
		}
		if price.Low <= trade.StopLoss {
			return "stop_loss"
		}
	} else {
		if price.Low <= trade.TakeProfit {
			return "take_profit"
		}
		if price.High >= trade.StopLoss {
			return "stop_loss"
		}
	}
	return "unknown"
}

func (b *Backtest) openPosition(result *analysis.AnalysisResult, price models.Price) *Trade {
	size := FixedSize / price.Close // Convert $10 to asset quantity

	return &Trade{
		Symbol:     result.Symbol,
		EntryTime:  price.OpenTime,
		Side:       result.Direction,
		EntryPrice: price.Close,
		Size:       size,
		StopLoss:   result.StopLoss,
		TakeProfit: result.TakeProfit,
	}
}

func (b *Backtest) closePosition(trade *Trade, price models.Price, reason string) {
	trade.ExitTime = price.OpenTime
	trade.ExitPrice = price.Close
	trade.Reason = reason

	// Calculate PnL
	var pnlPercentage float64
	if trade.Side == "long" {
		pnlPercentage = (trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice
	} else {
		pnlPercentage = (trade.EntryPrice - trade.ExitPrice) / trade.EntryPrice
	}

	// Calculate PnL in USDT (Fixed $10 position * leverage * percentage gain/loss)
	trade.PnL = FixedSize * pnlPercentage * float64(Leverage)

	b.updateBalance(trade.PnL)
	b.trades = append(b.trades, *trade)
	b.equityCurve = append(b.equityCurve, EquityPoint{
		Timestamp: price.OpenTime,
		Balance:   b.currentBalance,
	})
}

func (b *Backtest) updateBalance(pnl float64) {
	b.currentBalance += pnl
	if b.currentBalance > b.maxBalance {
		b.maxBalance = b.currentBalance
	}
	if b.currentBalance < 0 {
		b.currentBalance = 0
	}
}

func (b *Backtest) calculateResults() *BacktestResults {
	results := &BacktestResults{
		TotalTrades:  len(b.trades),
		FinalBalance: b.currentBalance,
		Trades:       b.trades,
		EquityCurve:  b.equityCurve,
	}

	var totalPnL float64
	returns := make([]float64, len(b.trades))

	for i, trade := range b.trades {
		if trade.PnL > 0 {
			results.WinningTrades++
		} else {
			results.LosingTrades++
		}
		totalPnL += trade.PnL
		returns[i] = trade.PnL / InitialBalance
	}

	if results.TotalTrades > 0 {
		results.WinRate = float64(results.WinningTrades) / float64(results.TotalTrades)
		results.AveragePnL = totalPnL / float64(results.TotalTrades)
	}

	results.MaxDrawdown = b.calculateMaxDrawdown()
	if len(returns) > 1 {
		results.SharpeRatio = b.calculateSharpeRatio(returns)
	}

	return results
}

func (b *Backtest) calculateMaxDrawdown() float64 {
	if b.maxBalance == 0 {
		return 0
	}

	maxDrawdown := 0.0
	for _, point := range b.equityCurve {
		drawdown := (b.maxBalance - point.Balance) / b.maxBalance
		maxDrawdown = math.Max(maxDrawdown, drawdown)
	}

	return maxDrawdown
}

func (b *Backtest) calculateSharpeRatio(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	avgReturn := average(returns)
	stdDev := standardDeviation(returns, avgReturn)

	if stdDev == 0 {
		return 0
	}

	return (avgReturn * math.Sqrt(252)) / stdDev
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func standardDeviation(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}

	var variance float64
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}

	variance = variance / float64(len(values)-1)
	return math.Sqrt(variance)
}
