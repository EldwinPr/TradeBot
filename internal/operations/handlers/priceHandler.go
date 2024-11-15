package handlers

import (
	"CryptoTradeBot/internal/operations/priceOperations"
	"CryptoTradeBot/internal/repositories"
	"context"
	"log"
	"os"

	"github.com/adshao/go-binance/v2/futures"
	"gorm.io/gorm"
)

func PriceHandler(db *gorm.DB) {
	priceRepo := repositories.NewPriceRepository(db)

	// Clear price table before starting
	if err := priceRepo.ClearTable(); err != nil {
		log.Printf("Error clearing price table: %v", err)
		return
	}

	futuresClient := futures.NewClient(os.Getenv("BINANCE_API_KEY"), os.Getenv("BINANCE_SECRET_KEY"))
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}

	ctx := context.Background()

	priceRecorder := priceOperations.NewPriceRecorder(futuresClient, priceRepo, symbols)
	priceFetcher := priceOperations.NewPriceFetcher(futuresClient, symbols)

	// fetch historical prices for different timeframes
	timeframes := [5]string{"5m", "15m", "1h", "4h", "1d"}
	days := [5]int{1, 2, 3, 5, 7}
	for i := 0; i < 5; i++ {
		i := i // Create local copy for goroutine
		go func() {
			prices, err := priceFetcher.GetHistoricalPrices(ctx, timeframes[i], days[i])
			if err != nil {
				log.Printf("Error fetching historical prices for %s: %v", timeframes[i], err)
				return
			}

			for _, price := range prices {
				price := price
				if err := priceRepo.Create(&price); err != nil {
					log.Printf("Error creating price record: %v", err)
				}
			}
		}()
	}

	// start recording prices
	go priceRecorder.StartRecording(ctx)
}
