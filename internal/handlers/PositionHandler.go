package handlers

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/operations/position"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/strategy"
	"context"
	"fmt"
	"sync"
)

type PositionHandler struct {
	executor     *position.PositionExecutor
	positionRepo *repositories.PositionRepository

	// For tracking and synchronization
	activePositions sync.Map
}

func NewPositionHandler(
	executor *position.PositionExecutor,
	positionRepo *repositories.PositionRepository,
) *PositionHandler {
	return &PositionHandler{
		executor:     executor,
		positionRepo: positionRepo,
	}
}

func (h *PositionHandler) HandlePositionRequest(ctx context.Context, req *strategy.PositionRequest) error {
	// Lock for the specific symbol
	key := fmt.Sprintf("position_%s", req.Symbol)
	if _, loaded := h.activePositions.LoadOrStore(key, true); loaded {
		return fmt.Errorf("position operation in progress for %s", req.Symbol)
	}
	defer h.activePositions.Delete(key)

	switch req.Action {
	case "open":
		position := &models.Position{
			Symbol:          req.Symbol,
			Side:            req.Side,
			EntryPrice:      req.EntryPrice,
			StopLossPrice:   req.StopLoss,
			TakeProfitPrice: req.TakeProfit,
		}
		return h.executor.OpenPosition(ctx, position)

	case "reverse":
		currentPos, err := h.getCurrentPosition(req.Symbol)
		if err != nil {
			return err
		}

		newPos := &models.Position{
			Symbol:          req.Symbol,
			Side:            req.Side,
			EntryPrice:      req.EntryPrice,
			StopLossPrice:   req.StopLoss,
			TakeProfitPrice: req.TakeProfit,
		}
		return h.executor.ReversePosition(ctx, currentPos, newPos)

	default:
		return fmt.Errorf("unknown action: %s", req.Action)
	}
}

func (h *PositionHandler) ClosePosition(ctx context.Context, symbol string, closePrice float64) error {
	position, err := h.getCurrentPosition(symbol)
	if err != nil {
		return err
	}
	return h.executor.ClosePosition(ctx, position, closePrice)
}

func (h *PositionHandler) getCurrentPosition(symbol string) (*models.Position, error) {
	positions, err := h.positionRepo.FindOpenPositionsBySymbol(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing position: %w", err)
	}
	if len(positions) == 0 {
		return nil, fmt.Errorf("no open position found for %s", symbol)
	}
	return &positions[0], nil
}
