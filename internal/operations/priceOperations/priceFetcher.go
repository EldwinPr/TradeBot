package priceOperations

import (
	"CryptoTradeBot/internal/models"
	"context"
	"log"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

type PriceFetcher struct {
	client  *futures.Client
	symbols []string
}

// NewPriceFetcher creates a new instance of PriceFetcher
func NewPriceFetcher(client *futures.Client, symbols []string) *PriceFetcher {
	return &PriceFetcher{
		client:  client,
		symbols: symbols,
	}
}

// getHistoricalPrices retrieves historical price data for the specified timeframe and number of days
func (f *PriceFetcher) GetHistoricalPrices(ctx context.Context, timeframe string, days int) ([]models.Price, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	var allPrices []models.Price

	for _, symbol := range f.symbols {
		klines, err := f.client.NewKlinesService().
			Symbol(symbol).
			Interval(timeframe).
			StartTime(startTime.UnixNano() / int64(time.Millisecond)).
			EndTime(endTime.UnixNano() / int64(time.Millisecond)).
			Do(ctx)

		if err != nil {
			log.Printf("Error fetching historical data for %s-%s: %v", symbol, timeframe, err)
			continue
		}

		for _, k := range klines {
			price := models.Price{
				Symbol:    symbol,
				TimeFrame: timeframe,
				OpenTime:  time.Unix(k.OpenTime/1000, 0),
				Open:      parseFloat(k.Open),
				High:      parseFloat(k.High),
				Low:       parseFloat(k.Low),
				Close:     parseFloat(k.Close),
				Volume:    parseFloat(k.Volume),
			}
			allPrices = append(allPrices, price)
		}
	}

	return allPrices, nil
}
