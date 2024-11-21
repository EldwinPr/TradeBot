package price

import (
	"context"
	"log"
	"strconv"
	"time"

	"CryptoTradeBot/internal/models"

	"github.com/adshao/go-binance/v2/futures"
)

type PriceFetcher struct {
	client  *futures.Client
	symbols []string
}

func NewPriceFetcher(client *futures.Client, symbols []string) *PriceFetcher {
	return &PriceFetcher{
		client:  client,
		symbols: symbols,
	}
}

func (f *PriceFetcher) FetchPrices(ctx context.Context, timeframe string, days int) ([]models.Price, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)
	var allPrices []models.Price

	// Calculate time chunks based on timeframe
	chunkDuration := calculateChunkDuration(timeframe)
	currentStart := startTime
	currentEnd := currentStart.Add(chunkDuration)

	// Process until we reach end time
	for currentStart.Before(endTime) {
		// Adjust if the end chunk would go past endTime
		if currentEnd.After(endTime) {
			currentEnd = endTime
		}

		for _, symbol := range f.symbols {
			klines, err := f.client.NewKlinesService().
				Symbol(symbol).
				Interval(timeframe).
				StartTime(currentStart.UnixNano() / int64(time.Millisecond)).
				EndTime(currentEnd.UnixNano() / int64(time.Millisecond)).
				Limit(500).
				Do(ctx)

			if err != nil {
				log.Printf("Error fetching prices for %s: %v", symbol, err)
				continue
			}

			for _, k := range klines {
				openTime := time.Unix(k.OpenTime/1000, 0)
				closeTime := time.Unix(k.CloseTime/1000, 0)

				price := models.Price{
					Symbol:     symbol,
					TimeFrame:  timeframe,
					OpenTime:   openTime,
					CloseTime:  closeTime,
					Open:       parseFloat(k.Open),
					High:       parseFloat(k.High),
					Low:        parseFloat(k.Low),
					Close:      parseFloat(k.Close),
					Volume:     parseFloat(k.Volume),
					TradeCount: k.TradeNum,
				}
				allPrices = append(allPrices, price)
			}

			log.Printf("Fetched %d %s candles for %s from %s to %s",
				len(klines),
				timeframe,
				symbol,
				currentStart.Format("2006-01-02 15:04:05"),
				currentEnd.Format("2006-01-02 15:04:05"))
		}

		// Move window forward
		currentStart = currentEnd
		currentEnd = currentStart.Add(chunkDuration)

		// Add small delay to avoid rate limits
		time.Sleep(100 * time.Millisecond)
	}

	return allPrices, nil
}

func calculateChunkDuration(timeframe string) time.Duration {
	// Calculate how many intervals fit in 500 candles
	intervalsMap := map[string]time.Duration{
		"1m":  time.Minute,
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
		"4h":  4 * time.Hour,
	}

	interval := intervalsMap[timeframe]
	return interval * 500 // 500 is Binance's max limit
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Error parsing float: %v", err)
		return 0
	}
	return f
}
