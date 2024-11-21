package backtest

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/strategy"
	"fmt"
	"sync"
	"time"
)

type Simulator struct {
	// Core components
	priceRepo       *repositories.PriceRepository
	strategyManager *strategy.StrategyManager
	config          Config

	// State tracking
	balance   float64
	positions map[string]*Trade

	// Price data cache
	priceCache struct {
		data map[string]map[string][]models.Price // symbol -> timeframe -> prices
		mu   sync.RWMutex
	}

	// Results collection
	results *BacktestResults
	mu      sync.RWMutex
}

func NewSimulator(config Config, strategyManager *strategy.StrategyManager, priceRepo *repositories.PriceRepository) *Simulator {
	return &Simulator{
		config:          config,
		strategyManager: strategyManager,
		priceRepo:       priceRepo,
		balance:         config.InitialBalance,
		positions:       make(map[string]*Trade),
		results:         &BacktestResults{},
		priceCache: struct {
			data map[string]map[string][]models.Price
			mu   sync.RWMutex
		}{
			data: make(map[string]map[string][]models.Price),
		},
	}
}

func (s *Simulator) ProcessCandle(symbol string, candle models.Price) error {
	// Check and load required price data for analysis
	if err := s.ensurePriceData(symbol, candle.OpenTime); err != nil {
		return err
	}

	// Get historical price arrays for analysis
	prices5m, err := s.getPriceHistory(symbol, "5m", candle.OpenTime)
	if err != nil {
		return err
	}
	prices15m, err := s.getPriceHistory(symbol, "15m", candle.OpenTime)
	if err != nil {
		return err
	}
	prices1h, err := s.getPriceHistory(symbol, "1h", candle.OpenTime)
	if err != nil {
		return err
	}
	prices4h, err := s.getPriceHistory(symbol, "4h", candle.OpenTime)
	if err != nil {
		return err
	}

	// Check existing position
	pos := s.positions[symbol]
	if pos != nil {
		// Check for TP/SL hits first
		if s.checkTPSL(pos, candle) {
			return nil
		}

		// Convert backtest Trade to models.Position for strategy
		modelPos := &models.Position{
			Symbol:   pos.Symbol,
			Side:     pos.Side,
			Size:     pos.Size,
			OpenTime: pos.EntryTime,
		}

		// Check for potential reversal
		result, err := s.strategyManager.Analyze(
			modelPos,
			prices5m,
			prices15m,
			prices1h,
			prices4h,
		)
		if err != nil {
			return err
		}

		if result.IsValid {
			return s.reversePosition(pos, result, candle)
		}
	} else {
		// No position - check for new entry
		result, err := s.strategyManager.Analyze(
			nil,
			prices5m,
			prices15m,
			prices1h,
			prices4h,
		)
		if err != nil {
			return err
		}

		if result.IsValid {
			return s.openPosition(symbol, result, candle)
		}
	}

	return nil
}

func (s *Simulator) checkTPSL(pos *Trade, candle models.Price) bool {
	if pos.Side == "long" {
		if candle.High >= pos.TakeProfit || candle.Low <= pos.StopLoss {
			s.closePosition(pos, candle)
			return true
		}
	} else {
		if candle.Low <= pos.TakeProfit || candle.High >= pos.StopLoss {
			s.closePosition(pos, candle)
			return true
		}
	}
	return false
}

func (s *Simulator) openPosition(symbol string, result *strategy.StrategyResult, candle models.Price) error {
	size := s.calculatePositionSize(result.EntryPrice)

	pos := &Trade{
		Symbol:     symbol,
		EntryTime:  candle.OpenTime,
		Side:       result.Direction,
		EntryPrice: candle.Close,
		Size:       size,
		StopLoss:   result.StopLoss,
		TakeProfit: result.TakeProfit,
	}

	s.positions[symbol] = pos
	return nil
}

func (s *Simulator) closePosition(pos *Trade, candle models.Price) {
	exitPrice := s.determineExitPrice(pos, candle)
	reason := s.determineExitReason(pos, candle)

	pos.ExitTime = candle.OpenTime
	pos.ExitPrice = exitPrice
	pos.Reason = reason

	// Calculate PnL
	pnlPercent := 0.0
	if pos.Side == "long" {
		pnlPercent = (exitPrice - pos.EntryPrice) / pos.EntryPrice
	} else {
		pnlPercent = (pos.EntryPrice - exitPrice) / pos.EntryPrice
	}

	pos.PnL = pos.Size * pnlPercent * float64(s.config.Leverage)

	// Update balance
	s.updateBalance(pos.PnL)

	// Store trade
	s.mu.Lock()
	s.results.Trades = append(s.results.Trades, *pos)
	s.mu.Unlock()

	// Clear position
	delete(s.positions, pos.Symbol)
}

func (s *Simulator) determineExitPrice(pos *Trade, candle models.Price) float64 {
	if pos.Side == "long" {
		if candle.High >= pos.TakeProfit {
			return pos.TakeProfit // Take profit hit
		}
		if candle.Low <= pos.StopLoss {
			return pos.StopLoss // Stop loss hit
		}
	} else {
		if candle.Low <= pos.TakeProfit {
			return pos.TakeProfit // Take profit hit
		}
		if candle.High >= pos.StopLoss {
			return pos.StopLoss // Stop loss hit
		}
	}
	return candle.Close // Regular close
}

func (s *Simulator) determineExitReason(pos *Trade, candle models.Price) string {
	if pos.Side == "long" {
		if candle.High >= pos.TakeProfit {
			return "take_profit"
		}
		if candle.Low <= pos.StopLoss {
			return "stop_loss"
		}
	} else {
		if candle.Low <= pos.TakeProfit {
			return "take_profit"
		}
		if candle.High >= pos.StopLoss {
			return "stop_loss"
		}
	}
	return "reversal"
}

func (s *Simulator) calculatePositionSize(price float64) float64 {
	riskAmount := s.balance * s.config.RiskPerTrade
	return (riskAmount * float64(s.config.Leverage)) / price
}

func (s *Simulator) updateBalance(pnl float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.balance += pnl
	if s.balance <= 0 {
		s.balance = 0
	}

	// Record equity point
	s.results.EquityCurve = append(s.results.EquityCurve, EquityPoint{
		Timestamp: time.Now(),
		Balance:   s.balance,
	})
}

func (s *Simulator) ensurePriceData(symbol string, Time time.Time) error {
	s.priceCache.mu.Lock()
	defer s.priceCache.mu.Unlock()

	// Initialize map for symbol if doesn't exist
	if _, exists := s.priceCache.data[symbol]; !exists {
		s.priceCache.data[symbol] = make(map[string][]models.Price)
	}

	// Load data for each timeframe if needed
	timeframes := []string{"5m", "15m", "1h", "4h"}
	for _, tf := range timeframes {
		if len(s.priceCache.data[symbol][tf]) == 0 {
			// Load enough historical data
			start := Time.Add(time.Duration(-24) * time.Hour)

			prices, err := s.priceRepo.GetPricesByTimeFrame(symbol, tf, start, Time)
			if err != nil {
				return err
			}
			s.priceCache.data[symbol][tf] = prices
		}
	}
	return nil
}

func (s *Simulator) getPriceHistory(symbol, timeframe string, currentTime time.Time) ([]models.Price, error) {
	s.priceCache.mu.RLock()
	defer s.priceCache.mu.RUnlock()

	if prices, exists := s.priceCache.data[symbol][timeframe]; exists {
		// Filter prices up to currentTime
		var filtered []models.Price
		for _, p := range prices {
			if p.OpenTime.Before(currentTime) || p.OpenTime.Equal(currentTime) {
				filtered = append(filtered, p)
			}
		}
		return filtered, nil
	}
	return nil, fmt.Errorf("no price data available for %s %s", symbol, timeframe)
}

func (s *Simulator) reversePosition(pos *Trade, result *strategy.StrategyResult, candle models.Price) error {
	// Close current position
	s.closePosition(pos, candle)

	// Open new position in opposite direction
	return s.openPosition(pos.Symbol, result, candle)
}
