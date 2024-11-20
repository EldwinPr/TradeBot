package handlers

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/strategy"
	"context"
	"sync"
	"time"
)

type StrategyHandler struct {
	// Repositories
	priceRepo    *repositories.PriceRepository
	positionRepo *repositories.PositionRepository

	// Strategy management
	strategyManager *strategy.StrategyManager
	symbols         map[string]*symbolProcessor

	// Concurrency control
	mu sync.RWMutex
	wg sync.WaitGroup
}

type symbolProcessor struct {
	symbol          string
	lastAnalysis    time.Time
	strategyManager *strategy.StrategyManager
	priceRepo       *repositories.PriceRepository
	positionRepo    *repositories.PositionRepository
}

func NewStrategyHandler(
	priceRepo *repositories.PriceRepository,
	positionRepo *repositories.PositionRepository,
	symbols []string,
) *StrategyHandler {
	handler := &StrategyHandler{
		priceRepo:       priceRepo,
		positionRepo:    positionRepo,
		strategyManager: strategy.NewStrategyManager(),
		symbols:         make(map[string]*symbolProcessor),
	}

	// Initialize symbol processors
	for _, symbol := range symbols {
		handler.symbols[symbol] = &symbolProcessor{
			symbol:          symbol,
			strategyManager: strategy.NewStrategyManager(),
			priceRepo:       priceRepo,
			positionRepo:    positionRepo,
		}
	}

	return handler
}

func (h *StrategyHandler) Start(ctx context.Context) {
	// Create ticker for XX:XX:00
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			// Only process if it's on the minute (:00)
			if t.Second() == 0 {
				h.processAllSymbols(ctx)
			}
		}
	}
}

func (h *StrategyHandler) processAllSymbols(ctx context.Context) {
	h.mu.RLock()
	symbols := make([]*symbolProcessor, 0, len(h.symbols))
	for _, processor := range h.symbols {
		symbols = append(symbols, processor)
	}
	h.mu.RUnlock()

	// Process each symbol concurrently
	var wg sync.WaitGroup
	for _, processor := range symbols {
		wg.Add(1)
		go func(p *symbolProcessor) {
			defer wg.Done()
			if err := p.process(ctx); err != nil {
				// Handle error (log it, etc)
			}
		}(processor)
	}
	wg.Wait()
}

func (p *symbolProcessor) process(ctx context.Context) error {
	// Get current position if any
	position, err := p.positionRepo.FindOpenPositionsBySymbol(p.symbol)
	if err != nil {
		return err
	}

	// Get historical prices for all timeframes
	end := time.Now()
	start := end.Add(-24 * time.Hour) // Adjust based on your needs

	prices5m, err := p.priceRepo.GetPricesByTimeFrame(p.symbol, models.PriceTimeFrame5m, start, end)
	if err != nil {
		return err
	}

	prices15m, err := p.priceRepo.GetPricesByTimeFrame(p.symbol, models.PriceTimeFrame15m, start, end)
	if err != nil {
		return err
	}

	prices1h, err := p.priceRepo.GetPricesByTimeFrame(p.symbol, models.PriceTimeFrame1h, start, end)
	if err != nil {
		return err
	}

	prices4h, err := p.priceRepo.GetPricesByTimeFrame(p.symbol, models.PriceTimeFrame4h, start, end)
	if err != nil {
		return err
	}

	// Analyze using strategy manager
	var currentPosition *models.Position
	if len(position) > 0 {
		currentPosition = &position[0]
	}

	result, err := p.strategyManager.Analyze(
		currentPosition,
		prices5m,
		prices15m,
		prices1h,
		prices4h,
	)
	if err != nil {
		return err
	}

	// Process strategy result
	if result.IsValid {
		p.processValidSignal(ctx, result)
	}

	p.lastAnalysis = time.Now()
	return nil
}

func (p *symbolProcessor) processValidSignal(ctx context.Context, result *strategy.StrategyResult) {
	// Here you would:
	// 1. Send signal to position handler
	// 2. Log the signal
	// 3. Update any monitoring metrics

	// This would typically emit an event or call your position handler
}

// Helper methods for handler management
func (h *StrategyHandler) AddSymbol(symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.symbols[symbol]; !exists {
		h.symbols[symbol] = &symbolProcessor{
			symbol:          symbol,
			strategyManager: strategy.NewStrategyManager(),
			priceRepo:       h.priceRepo,
			positionRepo:    h.positionRepo,
		}
	}
}

func (h *StrategyHandler) RemoveSymbol(symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.symbols, symbol)
}
