// backtest/types.go

package backtest

import (
	"time"
)

// Core trade record
type Trade struct {
	Symbol     string
	EntryTime  time.Time
	ExitTime   time.Time
	Side       string // "long" or "short"
	EntryPrice float64
	ExitPrice  float64
	Size       float64
	StopLoss   float64
	TakeProfit float64
	PnL        float64
	Reason     string // "take_profit", "stop_loss", "reversal"
}

// For tracking equity changes
type EquityPoint struct {
	Timestamp time.Time
	Balance   float64
}

// Final backtest results
type BacktestResults struct {
	// Trade metrics
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	AveragePnL    float64

	// Performance metrics
	MaxDrawdown  float64
	FinalBalance float64
	SharpeRatio  float64

	// Detailed records
	Trades      []Trade
	EquityCurve []EquityPoint
}

// Constants from your trading.go
const (
	InitialBalance = 10.0 // USDT
	Leverage       = 50   // Fixed leverage
	RiskPerTrade   = 0.02 // 2% per trade
)

// Simulation config
type Config struct {
	// Initial conditions
	InitialBalance float64
	Leverage       int
	RiskPerTrade   float64

	// Symbols to test
	Symbols []string

	// Time range
	StartTime time.Time
	EndTime   time.Time

	// Optional settings
	UseCache bool // Whether to cache price data
}

// NewConfig creates default config
func NewConfig() Config {
	return Config{
		InitialBalance: InitialBalance,
		Leverage:       Leverage,
		RiskPerTrade:   RiskPerTrade,
		UseCache:       true,
	}
}
