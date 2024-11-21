// backtest/engine.go

package backtest

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/strategy"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type Engine struct {
	// Core components
	priceRepo       *repositories.PriceRepository
	strategyManager *strategy.StrategyManager

	// Backtest state
	currentBalance float64
	maxBalance     float64
	trades         []Trade
	equityCurve    []EquityPoint

	// Config and results
	config  Config
	results *BacktestResults

	// For thread-safe results
	mu sync.RWMutex
}

func NewEngine(priceRepo *repositories.PriceRepository, strategyManager *strategy.StrategyManager, config Config) *Engine {
	return &Engine{
		priceRepo:       priceRepo,
		strategyManager: strategyManager,
		config:          config,
		currentBalance:  config.InitialBalance,
		maxBalance:      config.InitialBalance,
		trades:          make([]Trade, 0),
		equityCurve:     make([]EquityPoint, 0),
		results:         &BacktestResults{},
	}
}

func (e *Engine) RunBacktest(startTime, endTime time.Time, symbols []string) (*BacktestResults, error) {
	fmt.Printf("Running backtest from %s to %s\n",
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))

	for _, symbol := range symbols {
		fmt.Printf("Processing %s...\n", symbol)
		if err := e.runSymbol(symbol, startTime, endTime); err != nil {
			return nil, err
		}
	}

	results := e.calculateResults()
	fmt.Printf("Processed %d days of data\n", int(endTime.Sub(startTime).Hours()/24))

	return results, nil
}

func (e *Engine) runSymbol(symbol string, startTime, endTime time.Time) error {
	// Get historical prices
	prices, err := e.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame5m, startTime, endTime)
	if err != nil {
		return err
	}

	if len(prices) < 200 { // Minimum data requirement
		return fmt.Errorf("insufficient data for %s", symbol)
	}

	// Sort prices by time
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].OpenTime.Before(prices[j].OpenTime)
	})

	var activePosition *Trade

	// Process each candle
	for i := 200; i < len(prices); i++ {
		currentPrice := prices[i]

		// Skip if outside our date range
		if currentPrice.OpenTime.Before(startTime) || currentPrice.OpenTime.After(endTime) {
			continue
		}

		if activePosition != nil {
			if e.shouldExitPosition(activePosition, currentPrice) {
				exitReason := e.getExitReason(activePosition, currentPrice)
				e.closePosition(activePosition, currentPrice, exitReason)
				activePosition = nil
			}
			continue
		}

		// Analysis window (last 200 candles)
		analysisWindow := prices[i-200 : i+1]
		result, err := e.strategyManager.Analyze(nil, analysisWindow, nil, nil, nil) // Adjust parameters as needed

		if err != nil {
			return fmt.Errorf("error analyzing strategy: %w", err)
		}

		if result.IsValid {
			activePosition = e.openPosition(result, currentPrice)
			e.updateEquityCurve(currentPrice.OpenTime)
		}
	}

	return nil
}

func (e *Engine) shouldExitPosition(trade *Trade, price models.Price) bool {
	if trade.Side == "long" {
		return price.High >= trade.TakeProfit || price.Low <= trade.StopLoss
	}
	return price.Low <= trade.TakeProfit || price.High >= trade.StopLoss
}

func (e *Engine) getExitReason(trade *Trade, price models.Price) string {
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

func (e *Engine) openPosition(result *strategy.StrategyResult, price models.Price) *Trade {
	size := e.config.InitialBalance * e.config.RiskPerTrade * float64(e.config.Leverage) / price.Close

	return &Trade{
		Symbol:     price.Symbol,
		EntryTime:  price.OpenTime,
		Side:       result.Direction,
		EntryPrice: price.Close,
		Size:       size,
		StopLoss:   result.StopLoss,
		TakeProfit: result.TakeProfit,
	}
}

func (e *Engine) closePosition(trade *Trade, price models.Price, reason string) {
	trade.ExitTime = price.OpenTime
	trade.ExitPrice = price.Close
	trade.Reason = reason

	// Calculate PnL
	pnlPercent := 0.0
	if trade.Side == "long" {
		pnlPercent = (trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice
	} else {
		pnlPercent = (trade.EntryPrice - trade.ExitPrice) / trade.EntryPrice
	}

	trade.PnL = trade.Size * pnlPercent * float64(e.config.Leverage)

	e.updateBalance(trade.PnL)
	e.mu.Lock()
	e.trades = append(e.trades, *trade)
	e.mu.Unlock()
}

func (e *Engine) updateBalance(pnl float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.currentBalance += pnl
	if e.currentBalance > e.maxBalance {
		e.maxBalance = e.currentBalance
	}
	if e.currentBalance < 0 {
		e.currentBalance = 0
	}
}

func (e *Engine) updateEquityCurve(timestamp time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.equityCurve = append(e.equityCurve, EquityPoint{
		Timestamp: timestamp,
		Balance:   e.currentBalance,
	})
}

func (e *Engine) calculateResults() *BacktestResults {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.trades) == 0 {
		return &BacktestResults{
			FinalBalance: e.currentBalance,
		}
	}

	// Calculate trade metrics
	totalTrades := len(e.trades)
	winningTrades := 0
	losingTrades := 0
	totalPnL := 0.0
	winningPnL := 0.0
	losingPnL := 0.0

	for _, trade := range e.trades {
		if trade.PnL > 0 {
			winningTrades++
			winningPnL += trade.PnL
		} else {
			losingTrades++
			losingPnL += math.Abs(trade.PnL)
		}
		totalPnL += trade.PnL
	}

	// Calculate metrics
	winRate := float64(winningTrades) / float64(totalTrades)
	averagePnL := totalPnL / float64(totalTrades)

	// Calculate drawdown
	maxDrawdown := 0.0
	peakBalance := e.config.InitialBalance

	for _, point := range e.equityCurve {
		if point.Balance > peakBalance {
			peakBalance = point.Balance
		}
		drawdown := (peakBalance - point.Balance) / peakBalance
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	// Calculate Sharpe ratio
	sharpeRatio := e.calculateSharpeRatio()

	return &BacktestResults{
		TotalTrades:   totalTrades,
		WinningTrades: winningTrades,
		LosingTrades:  losingTrades,
		WinRate:       winRate,
		AveragePnL:    averagePnL,
		MaxDrawdown:   maxDrawdown,
		FinalBalance:  e.currentBalance,
		SharpeRatio:   sharpeRatio,
		Trades:        e.trades,
		EquityCurve:   e.equityCurve,
	}
}

func (e *Engine) calculateSharpeRatio() float64 {
	if len(e.equityCurve) < 2 {
		return 0
	}

	// Calculate returns
	returns := make([]float64, len(e.equityCurve)-1)
	for i := 1; i < len(e.equityCurve); i++ {
		returns[i-1] = (e.equityCurve[i].Balance - e.equityCurve[i-1].Balance) /
			e.equityCurve[i-1].Balance
	}

	// Calculate average return
	avgReturn := 0.0
	for _, r := range returns {
		avgReturn += r
	}
	avgReturn /= float64(len(returns))

	// Calculate standard deviation
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-avgReturn, 2)
	}
	variance /= float64(len(returns) - 1) // Use n-1 for sample variance
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	// Annualize (assuming daily returns)
	annualizedReturn := avgReturn * 252 // Trading days in a year
	annualizedStdDev := stdDev * math.Sqrt(252)

	return annualizedReturn / annualizedStdDev
}
