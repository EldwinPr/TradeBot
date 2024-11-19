package handlers

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/analysis"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	InitialBalance = 1000.0 // USDT
	Leverage       = 50     // Fixed leverage
	RiskPerTrade   = 0.02   // 2% per trade
)

type AnalysisHandler struct {
	analysis     *analysis.Analysis
	priceRepo    *repositories.PriceRepository
	positionRepo *repositories.PositionRepository
	balanceRepo  *repositories.BalanceRepository
}

func NewAnalysisHandler(
	analysis *analysis.Analysis,
	priceRepo *repositories.PriceRepository,
	positionRepo *repositories.PositionRepository,
	balanceRepo *repositories.BalanceRepository,
) *AnalysisHandler {
	return &AnalysisHandler{
		analysis:     analysis,
		priceRepo:    priceRepo,
		positionRepo: positionRepo,
		balanceRepo:  balanceRepo,
	}
}

func (h *AnalysisHandler) Start(ctx context.Context, symbols []string) {
	// Start position monitor
	go h.monitorPositions(ctx)

	// Start analysis for each symbol
	var wg sync.WaitGroup
	for _, symbol := range symbols {
		wg.Add(1)
		go h.analyzeSymbol(ctx, symbol, &wg)
	}
	wg.Wait()
}

func (h *AnalysisHandler) analyzeSymbol(ctx context.Context, symbol string, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check for existing position
			positions, err := h.positionRepo.FindOpenPositionsBySymbol(symbol)
			if err != nil {
				log.Printf("Error checking positions for %s: %v", symbol, err)
				continue
			}

			// Skip if position exists
			if len(positions) > 0 {
				continue
			}

			// Get latest prices
			prices, err := h.priceRepo.GetPricesByTimeFrame(
				symbol,
				models.PriceTimeFrame5m,
				time.Now().AddDate(0, 0, -1),
				time.Now(),
			)
			if err != nil {
				log.Printf("Error getting prices for %s: %v", symbol, err)
				continue
			}

			if len(prices) < 10 {
				continue
			}

			// Run analysis
			result := h.analysis.Analyze(prices)

			// Execute trade if valid
			if result.IsValid {
				if err := h.openPosition(result); err != nil {
					log.Printf("Error opening position for %s: %v", symbol, err)
					continue
				}
				log.Printf("Opened position for %s: %s at price %.8f",
					symbol, result.Direction, result.EntryPrice)
			}
		}
	}
}

func (h *AnalysisHandler) openPosition(result *analysis.AnalysisResult) error {
	// Get current balance
	balance, err := h.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %v", err)
	}

	// Use the balance variable to log the current balance
	log.Printf("Current balance: %.2f USDT", balance.Balance)

	// Calculate position size using fixed size
	const FixedSize = 1.0 // $1 per trade
	positionSize := (FixedSize / result.EntryPrice) * float64(Leverage)

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

	return h.positionRepo.Create(position)
}

func (h *AnalysisHandler) monitorPositions(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := h.checkOpenPositions(); err != nil {
				log.Printf("Error checking positions: %v", err)
			}
		}
	}
}

func (h *AnalysisHandler) checkOpenPositions() error {
	positions, err := h.positionRepo.FindOpenPositions()
	if err != nil {
		return fmt.Errorf("failed to get open positions: %v", err)
	}

	for i := range positions {
		if err := h.checkPosition(&positions[i]); err != nil {
			log.Printf("Error checking position %d: %v", positions[i].ID, err)
		}
	}

	return nil
}

func (h *AnalysisHandler) checkPosition(position *models.Position) error {
	latest, err := h.priceRepo.GetLatestPrice(position.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get price: %v", err)
	}

	currentPrice := latest.Close
	shouldClose := false
	pnl := 0.0

	if position.Side == models.PositionSideLong {
		if currentPrice >= position.TakeProfitPrice || currentPrice <= position.StopLossPrice {
			pnl = (currentPrice - position.EntryPrice) * position.Size
			shouldClose = true
		}
	} else {
		if currentPrice <= position.TakeProfitPrice || currentPrice >= position.StopLossPrice {
			pnl = (position.EntryPrice - currentPrice) * position.Size
			shouldClose = true
		}
	}

	if shouldClose {
		return h.closePosition(position, currentPrice, pnl)
	}

	return nil
}

func (h *AnalysisHandler) closePosition(position *models.Position, closePrice, pnl float64) error {
	position.CloseTime = time.Now()
	position.Status = models.PositionStatusClosed
	position.PnL = pnl
	position.UpdatedAt = time.Now()

	if err := h.positionRepo.Update(position); err != nil {
		return fmt.Errorf("failed to update position: %v", err)
	}

	balance, err := h.balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %v", err)
	}

	balance.Balance += pnl
	balance.LastUpdated = time.Now()

	if err := h.balanceRepo.Update(balance); err != nil {
		return fmt.Errorf("failed to update balance: %v", err)
	}

	log.Printf("Position closed: %s %s | Entry: %.8f Exit: %.8f | PnL: %.2f USDT",
		position.Symbol, position.Side, position.EntryPrice, closePrice, pnl)

	return nil
}
