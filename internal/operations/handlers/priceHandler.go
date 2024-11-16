package handlers

import (
	"CryptoTradeBot/internal/operations/priceOperations"
	"CryptoTradeBot/internal/repositories"
	"context"
	"log"
	"os"

	"github.com/adshao/go-binance/v2/futures"
)

type PriceHandler struct {
	priceRepo     *repositories.PriceRepository
	futuresClient *futures.Client
	priceRecorder *priceOperations.PriceRecorder
	priceFetcher  *priceOperations.PriceFetcher
}

func NewPriceHandler(priceRepo *repositories.PriceRepository) *PriceHandler {
	futuresClient := futures.NewClient(os.Getenv("BINANCE_API_KEY"), os.Getenv("BINANCE_SECRET_KEY"))

	return &PriceHandler{
		priceRepo:     priceRepo,
		futuresClient: futuresClient,
		// Note: symbols will be passed in Start method
		priceFetcher: priceOperations.NewPriceFetcher(futuresClient, nil),
	}
}

func (h *PriceHandler) Start(ctx context.Context, symbols []string) error {
	// Clear price table before starting
	if err := h.priceRepo.ClearTable(); err != nil {
		return err
	}

	// Initialize PriceRecorder with symbols
	h.priceRecorder = priceOperations.NewPriceRecorder(h.futuresClient, h.priceRepo, symbols)

	// Update PriceFetcher with symbols
	h.priceFetcher = priceOperations.NewPriceFetcher(h.futuresClient, symbols)

	// Fetch initial historical data
	if err := h.fetchHistoricalData(ctx, symbols); err != nil {
		return err
	}

	// Start real-time price recording
	go h.priceRecorder.StartRecording(ctx)

	return nil
}

func (h *PriceHandler) fetchHistoricalData(ctx context.Context, symbols []string) error {
	timeframes := map[string]int{
		"5m":  30, // 30 days
		"15m": 30, // 30 days
		"1h":  30, // 30 days
		"4h":  30, // 30 days
	}

	for timeframe, days := range timeframes {
		log.Printf("Fetching %s historical data for %d days", timeframe, days)

		prices, err := h.priceFetcher.GetHistoricalPrices(ctx, timeframe, days)
		if err != nil {
			return err
		}

		for _, price := range prices {
			if err := h.priceRepo.Create(&price); err != nil {
				log.Printf("Error saving historical price: %v", err)
			}
		}
	}

	return nil
}
