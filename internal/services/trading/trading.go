package trading

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/analysis"
	"context"
	"fmt"
	"log"
	"time"
)

type PaperTrader struct {
	positionRepo *repositories.PositionRepository
	priceRepo    *repositories.PriceRepository
	balanceRepo  *repositories.BalanceRepository
}

const (
	InitialBalance = 1000.0 // USDT
	Leverage       = 50     // Fixed leverage
	RiskPerTrade   = 0.02   // 2% per trade
)

func NewPaperTrader(positionRepo *repositories.PositionRepository,
	priceRepo *repositories.PriceRepository,
	balanceRepo *repositories.BalanceRepository) *PaperTrader {
	return &PaperTrader{
		positionRepo: positionRepo,
		priceRepo:    priceRepo,
		balanceRepo:  balanceRepo,
	}
}

// OpenPosition creates new paper trade position
func (t *PaperTrader) OpenPosition(result *analysis.AnalysisResult) error {
	// Get current balance
	balance, err := t.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %v", err)
	}

	// Calculate position size
	positionSize := (balance.Balance * RiskPerTrade) * float64(Leverage)

	position := &models.Position{
		Symbol:          result.Symbol,
		Side:            result.Direction,
		Size:            positionSize,
		Leverage:        Leverage,
		EntryPrice:      result.EntryPrice,
		StopLossPrice:   result.StopLoss,
		TakeProfitPrice: result.TakeProfit,
		OpenTime:        time.Now(),
		Status:          models.PositionStatusOpen,
		PnL:             0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	return t.positionRepo.Create(position)
}

// MonitorPositions checks open positions for take profit or stop loss
func (t *PaperTrader) MonitorPositions(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.checkOpenPositions(); err != nil {
				log.Printf("Error checking positions: %v", err)
			}
		}
	}
}

func (t *PaperTrader) checkOpenPositions() error {
	positions, err := t.positionRepo.FindOpenPositions()
	if err != nil {
		return fmt.Errorf("failed to get open positions: %v", err)
	}

	for i := range positions {
		if err := t.checkPosition(&positions[i]); err != nil {
			log.Printf("Error checking position %d: %v", positions[i].ID, err)
		}
	}

	return nil
}

// checkPosition expects a pointer to Position
func (t *PaperTrader) checkPosition(position *models.Position) error {
	// Get current price
	latest, err := t.priceRepo.GetLatestPrice(position.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get price: %v", err)
	}

	currentPrice := latest.Close
	shouldClose := false
	pnl := 0.0

	// Check for take profit or stop loss
	if position.Side == models.PositionSideLong {
		if currentPrice >= position.TakeProfitPrice || currentPrice <= position.StopLossPrice {
			pnl = (currentPrice - position.EntryPrice) * position.Size * float64(position.Leverage)
			shouldClose = true
		}
	} else {
		if currentPrice <= position.TakeProfitPrice || currentPrice >= position.StopLossPrice {
			pnl = (position.EntryPrice - currentPrice) * position.Size * float64(position.Leverage)
			shouldClose = true
		}
	}

	if shouldClose {
		return t.closePosition(position, currentPrice, pnl)
	}

	return nil
}

func (t *PaperTrader) closePosition(position *models.Position, closePrice, pnl float64) error {
	// Update position
	position.CloseTime = time.Now()
	position.Status = models.PositionStatusClosed
	position.PnL = pnl
	position.UpdatedAt = time.Now()

	// Save position
	if err := t.positionRepo.Update(position); err != nil {
		return fmt.Errorf("failed to update position: %v", err)
	}

	// Update balance
	balance, err := t.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %v", err)
	}

	balance.Balance += pnl
	balance.LastUpdated = time.Now()

	if err := t.balanceRepo.Update(balance); err != nil {
		return fmt.Errorf("failed to update balance: %v", err)
	}

	log.Printf("Position closed: %s %s | Entry: %.8f Exit: %.8f | PnL: %.2f USDT",
		position.Symbol, position.Side, position.EntryPrice, closePrice, pnl)

	return nil
}
