package handlers

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/analysis"
	"context"
	"log"
	"sync"
	"time"
)

type AnalysisHandler struct {
	analysis     *analysis.Analysis
	tradeFunc    func(*analysis.AnalysisResult)
	priceRepo    *repositories.PriceRepository
	positionRepo *repositories.PositionRepository // Add position repository
}

func NewAnalysisHandler(
	analysis *analysis.Analysis,
	priceRepo *repositories.PriceRepository,
	positionRepo *repositories.PositionRepository,
	tradeFunc func(*analysis.AnalysisResult),
) *AnalysisHandler {
	return &AnalysisHandler{
		analysis:     analysis,
		tradeFunc:    tradeFunc,
		priceRepo:    priceRepo,
		positionRepo: positionRepo,
	}
}

func (h *AnalysisHandler) Start(ctx context.Context, symbols []string) {
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
			// Check for existing open position
			openPositions, err := h.positionRepo.FindOpenPositionsBySymbol(symbol)
			if err != nil {
				log.Printf("Error checking positions for %s: %v", symbol, err)
				continue
			}

			// Skip analysis if position exists
			if len(openPositions) > 0 {
				continue
			}

			// Get latest prices
			prices, err := h.priceRepo.GetPricesByTimeFrame(symbol, models.PriceTimeFrame5m, time.Now().AddDate(0, 0, -1), time.Now())
			if err != nil {
				log.Printf("Error getting prices for %s: %v", symbol, err)
				continue
			}

			if len(prices) < 10 {
				continue
			}

			// Run analysis
			result := h.analysis.Analyze(prices)

			// Execute trade if valid and no existing position
			if result.IsValid {
				h.tradeFunc(result)
				log.Printf("Trade signal for %s: %s at price %.8f", symbol, result.Direction, result.EntryPrice)
				// trading.OpenPosition(result)
			}
		}
	}
}
