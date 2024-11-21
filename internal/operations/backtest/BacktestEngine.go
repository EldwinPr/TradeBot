// backtest/engine.go

package backtest

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/strategy"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

type Engine struct {
	// Core components
	priceRepo       *repositories.PriceRepository
	positionRepo    *repositories.PositionRepository // Add this
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

const FixedSize = 1

func NewEngine(priceRepo *repositories.PriceRepository, positionRepo *repositories.PositionRepository, strategyManager *strategy.StrategyManager, config Config) *Engine {
	return &Engine{
		priceRepo:       priceRepo,
		positionRepo:    positionRepo,
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
	log.Printf("Processing %s from %s to %s", symbol,
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))

	// process 5m data
	extendedStart := startTime.Add(-2 * time.Hour)
	prices5m, err := e.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame5m, extendedStart, endTime)
	if err != nil {
		return err
	}
	log.Printf("Loaded %d 5m candles", len(prices5m))

	// Sort prices by time
	sort.Slice(prices5m, func(i, j int) bool {
		return prices5m[i].OpenTime.Before(prices5m[j].OpenTime)
	})

	// process 15m data
	extendedStart = startTime.Add(-4 * time.Hour)
	prices15m, err := e.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame15m, extendedStart, endTime)
	if err != nil {
		return err
	}
	log.Printf("Loaded %d 15m candles", len(prices15m))

	// Fix: Sort prices15m (was using prices5m in index)
	sort.Slice(prices15m, func(i, j int) bool {
		return prices15m[i].OpenTime.Before(prices15m[j].OpenTime)
	})

	// process 1h data
	extendedStart = startTime.Add(-24 * time.Hour)
	prices1h, err := e.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame1h, extendedStart, endTime)
	if err != nil {
		return err
	}
	log.Printf("Loaded %d 1h candles", len(prices1h))

	sort.Slice(prices1h, func(i, j int) bool {
		return prices1h[i].OpenTime.Before(prices1h[j].OpenTime)
	})

	// process 4h data
	extendedStart = startTime.Add(-96 * time.Hour)
	prices4h, err := e.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame4h, extendedStart, endTime)
	if err != nil {
		return err
	}
	log.Printf("Loaded %d 4h candles", len(prices4h))

	sort.Slice(prices4h, func(i, j int) bool {
		return prices4h[i].OpenTime.Before(prices4h[j].OpenTime)
	})

	log.Printf("Starting candle processing for %s", symbol)

	validSetups := 0
	for i := 200; i < len(prices5m); i++ {
		currentPrice := prices5m[i]

		if currentPrice.OpenTime.Before(startTime) || currentPrice.OpenTime.After(endTime) {
			continue
		}

		positions, err := e.positionRepo.FindOpenPositionsBySymbol(currentPrice.Symbol)
		if err != nil {
			return fmt.Errorf("error checking positions: %w", err)
		}
		if len(positions) > 0 {
			log.Printf("Skip analysis - existing position for %s", currentPrice.Symbol)
			continue
		}

		result, err := e.strategyManager.Analyze(nil, prices5m[i-200:i+1], prices15m, prices1h, prices4h)
		if err != nil {
			return fmt.Errorf("error analyzing strategy: %w", err)
		}

		if result.IsValid {
			validSetups++
			log.Printf("Valid setup found for %s. Direction: %s, Price: %.4f",
				currentPrice.Symbol, result.Direction, currentPrice.Close)

			newModelPosition := &models.Position{
				Symbol:          currentPrice.Symbol,
				Side:            result.Direction,
				Size:            FixedSize,
				Leverage:        Leverage,
				EntryPrice:      currentPrice.Close,
				StopLossPrice:   result.StopLoss,
				TakeProfitPrice: result.TakeProfit,
				OpenTime:        currentPrice.OpenTime,
				Status:          models.PositionStatusOpen,
			}

			if err := e.positionRepo.Create(newModelPosition); err != nil {
				return fmt.Errorf("error creating position: %w", err)
			}
			log.Printf("Position opened for %s", currentPrice.Symbol)
			e.updateEquityCurve(currentPrice.OpenTime)
		}
	}
	log.Printf("Found %d valid setups", validSetups)
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
	return &Trade{
		Symbol:     price.Symbol,
		EntryTime:  price.OpenTime,
		Side:       result.Direction,
		EntryPrice: price.Close,
		Size:       FixedSize, // Use fixed size directly
		StopLoss:   result.StopLoss,
		TakeProfit: result.TakeProfit,
	}
}

func (e *Engine) closePosition(trade *Trade, price models.Price, reason string) {
	trade.ExitTime = price.OpenTime
	trade.ExitPrice = price.Close
	trade.Reason = reason

	pnlPercent := 0.0
	if trade.Side == "long" {
		pnlPercent = (price.Close - trade.EntryPrice) / trade.EntryPrice
	} else {
		pnlPercent = (trade.EntryPrice - price.Close) / trade.EntryPrice
	}

	trade.PnL = FixedSize * pnlPercent * float64(e.config.Leverage)

	// Update position in database
	position, err := e.positionRepo.FindOpenPositionsBySymbol(trade.Symbol)
	if err == nil && len(position) > 0 {
		position[0].Status = models.PositionStatusClosed
		position[0].CloseTime = price.OpenTime
		position[0].PnL = trade.PnL
		e.positionRepo.Update(&position[0])
	}

	e.mu.Lock()
	e.currentBalance += trade.PnL
	if e.currentBalance > e.maxBalance {
		e.maxBalance = e.currentBalance
	}
	e.trades = append(e.trades, *trade)
	e.updateEquityCurve(price.OpenTime)
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

// Calculate PnL for a trade
func (e *Engine) calculatePnL(trade *Trade, currentPrice models.Price) float64 {
	if trade.Side == "long" {
		return (currentPrice.Close - trade.EntryPrice) * trade.Size * float64(e.config.Leverage)
	}
	return (trade.EntryPrice - currentPrice.Close) * trade.Size * float64(e.config.Leverage)
}
