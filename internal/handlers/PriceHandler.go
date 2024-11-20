package handlers

import (
	"context"
	"log"

	"CryptoTradeBot/internal/operations/price"
	"CryptoTradeBot/internal/repositories"

	"github.com/adshao/go-binance/v2/futures"
)

type PriceHandler struct {
	priceRepo     *repositories.PriceRepository
	futuresClient *futures.Client
	priceRecorder *price.PriceRecorder
	priceFetcher  *price.PriceFetcher
	symbols       []string
}

func NewPriceHandler(client *futures.Client, priceRepo *repositories.PriceRepository, symbols []string) *PriceHandler {
	return &PriceHandler{
		futuresClient: client,
		priceRepo:     priceRepo,
		symbols:       symbols,
		priceFetcher:  price.NewPriceFetcher(client, symbols),
	}
}

func (h *PriceHandler) Start(ctx context.Context) error {

	// Initialize PriceRecorder
	h.priceRecorder = price.NewPriceRecorder(h.futuresClient, h.priceRepo, h.symbols)

	// Fetch initial historical data
	if err := h.fetchHistoricalData(ctx); err != nil {
		return err
	}

	// Start real-time price recording
	go h.priceRecorder.StartRecording(ctx)

	return nil
}

func (h *PriceHandler) fetchHistoricalData(ctx context.Context) error {
	timeframes := map[string]int{
		"5m":  31,
		"15m": 31,
		"1h":  31,
		"4h":  31,
	}

	for timeframe, days := range timeframes {
		log.Printf("Fetching %s historical data for %d days", timeframe, days)

		prices, err := h.priceFetcher.FetchPrices(ctx, timeframe, days) // Use FetchPrices instead
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
