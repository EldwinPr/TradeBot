package price

import (
	"context"
	"log"
	"time"

	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"

	"github.com/adshao/go-binance/v2/futures"
)

type PriceRecorder struct {
	client    *futures.Client
	priceRepo *repositories.PriceRepository
	symbols   []string
}

func NewPriceRecorder(client *futures.Client, priceRepo *repositories.PriceRepository, symbols []string) *PriceRecorder {
	return &PriceRecorder{
		client:    client,
		priceRepo: priceRepo,
		symbols:   symbols,
	}
}

func (r *PriceRecorder) StartRecording(ctx context.Context) {
	// Choose timeframes to record
	timeframes := map[string]time.Duration{
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
		"4h":  4 * time.Hour,
	}

	for timeframe, interval := range timeframes {
		go r.recordTimeframe(ctx, timeframe, interval)
	}
}

func (r *PriceRecorder) recordTimeframe(ctx context.Context, timeframe string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting %s price recording...", timeframe)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping %s price recording...", timeframe)
			return
		case <-ticker.C:
			r.recordPrices(ctx, timeframe)
		}
	}
}

func (r *PriceRecorder) recordPrices(ctx context.Context, timeframe string) {
	for _, symbol := range r.symbols {
		klines, err := r.client.NewKlinesService().
			Symbol(symbol).
			Interval(timeframe).
			Limit(1).
			Do(ctx)

		if err != nil {
			log.Printf("Error getting kline for %s-%s: %v", symbol, timeframe, err)
			continue
		}

		if len(klines) > 0 {
			k := klines[0]
			price := &models.Price{
				Symbol:     symbol,
				TimeFrame:  timeframe,
				OpenTime:   time.Unix(k.OpenTime/1000, 0),
				CloseTime:  time.Unix(k.CloseTime/1000, 0),
				Open:       parseFloat(k.Open),
				High:       parseFloat(k.High),
				Low:        parseFloat(k.Low),
				Close:      parseFloat(k.Close),
				Volume:     parseFloat(k.Volume),
				TradeCount: k.TradeNum, // Including trade count from previous discussion
			}

			if err := r.priceRepo.Create(price); err != nil {
				log.Printf("Error saving price for %s-%s: %v", symbol, timeframe, err)
			} else {
				log.Printf("Recorded %s price for %s: %v", timeframe, symbol, price.Close)
			}
		}
	}
}
