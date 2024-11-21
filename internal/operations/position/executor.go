package position

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"context"
	"fmt"
	"time"
)

type PositionExecutor struct {
	positionRepo *repositories.PositionRepository
	balanceRepo  *repositories.BalanceRepository
	leverage     int
	riskPerTrade float64
}

func NewPositionExecutor(
	positionRepo *repositories.PositionRepository,
	balanceRepo *repositories.BalanceRepository,
) *PositionExecutor {
	return &PositionExecutor{
		positionRepo: positionRepo,
		balanceRepo:  balanceRepo,
		leverage:     50,
		riskPerTrade: 0.02,
	}
}

func (e *PositionExecutor) OpenPosition(ctx context.Context, req *models.Position) error {
	balance, err := e.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	req.Size = e.calculatePositionSize(balance.Balance, req.EntryPrice)
	req.Leverage = e.leverage
	req.OpenTime = time.Now()
	req.Status = models.PositionStatusOpen

	return e.positionRepo.Create(req)
}

func (e *PositionExecutor) ReversePosition(ctx context.Context, currentPos *models.Position, newPos *models.Position) error {
	// Calculate PnL for existing position
	pnl := e.calculatePnL(currentPos, newPos.EntryPrice)

	// Update current position
	currentPos.CloseTime = time.Now()
	currentPos.Status = models.PositionStatusClosed
	currentPos.PnL = pnl

	// Get balance
	balance, err := e.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	// Update balance with PnL
	balance.Balance += pnl
	balance.LastUpdated = time.Now()

	// Setup new position
	newPos.Size = e.calculatePositionSize(balance.Balance, newPos.EntryPrice)
	newPos.Leverage = e.leverage
	newPos.OpenTime = time.Now()
	newPos.Status = models.PositionStatusOpen

	// Use repository transaction
	return e.positionRepo.ReversePosition(currentPos, newPos, balance)
}

func (e *PositionExecutor) ClosePosition(ctx context.Context, position *models.Position, closePrice float64) error {
	// Calculate PnL
	pnl := e.calculatePnL(position, closePrice)

	// Update position
	position.CloseTime = time.Now()
	position.Status = models.PositionStatusClosed
	position.PnL = pnl

	// Get balance for update
	balance, err := e.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	balance.Balance += pnl
	balance.LastUpdated = time.Now()

	// Use repository for transaction
	return e.positionRepo.ClosePosition(position, balance)
}

func (e *PositionExecutor) calculatePnL(position *models.Position, closePrice float64) float64 {
	if position.Side == models.PositionSideLong {
		return (closePrice - position.EntryPrice) * position.Size * float64(position.Leverage)
	}
	return (position.EntryPrice - closePrice) * position.Size * float64(position.Leverage)
}

func (e *PositionExecutor) calculatePositionSize(balance, entryPrice float64) float64 {
	riskAmount := balance * e.riskPerTrade
	return (riskAmount * float64(e.leverage)) / entryPrice
}
