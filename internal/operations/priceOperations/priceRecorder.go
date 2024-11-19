package priceOperations

import (
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/time/rate"
)

type PriceRecorder struct {
	client    *futures.Client
	priceRepo *repositories.PriceRepository
	symbols   []string
	limiter   *rate.Limiter
	logger    *log.Logger
	wg        sync.WaitGroup
}

func NewPriceRecorder(client *futures.Client, priceRepo *repositories.PriceRepository, symbols []string) *PriceRecorder {
	return &PriceRecorder{
		client:    client,
		priceRepo: priceRepo,
		symbols:   symbols,
		limiter:   rate.NewLimiter(rate.Limit(10), 20), // 10 requests/sec with burst of 20
		logger:    log.New(log.Writer(), "[PriceRecorder] ", log.LstdFlags),
	}
}

func (r *PriceRecorder) StartRecording(ctx context.Context) {
	timeframes := map[string]time.Duration{
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
		"4h":  4 * time.Hour,
		"1d":  24 * time.Hour,
	}

	// Start recording for each timeframe
	for timeframe, interval := range timeframes {
		r.wg.Add(1)
		go r.recordTimeframe(ctx, timeframe, interval)
	}

	// Wait for all recording goroutines to finish
	go func() {
		r.wg.Wait()
		r.logger.Println("All recording routines have stopped")
	}()
}

func (r *PriceRecorder) recordTimeframe(ctx context.Context, timeframe string, interval time.Duration) {
	defer r.wg.Done()

	// Align ticker to the start of the next interval
	firstTick := time.Now().Add(interval).Truncate(interval)
	time.Sleep(time.Until(firstTick))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.logger.Printf("Starting %s price recording", timeframe)

	for {
		select {
		case <-ctx.Done():
			r.logger.Printf("Stopping %s price recording", timeframe)
			return
		case <-ticker.C:
			if err := r.recordPricesForTimeframe(ctx, timeframe); err != nil {
				r.logger.Printf("Error recording %s prices: %v", timeframe, err)
			}
		}
	}
}

func (r *PriceRecorder) recordPricesForTimeframe(ctx context.Context, timeframe string) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(r.symbols))

	for _, symbol := range r.symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			// Wait for rate limiter
			if err := r.limiter.Wait(ctx); err != nil {
				errors <- fmt.Errorf("rate limiter error for %s: %w", sym, err)
				return
			}

			if err := r.recordSymbolPrice(ctx, sym, timeframe); err != nil {
				errors <- fmt.Errorf("error recording %s-%s: %w", sym, timeframe, err)
			}
		}(symbol)
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errors)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		// Check for any errors
		var errMsgs []string
		for err := range errors {
			errMsgs = append(errMsgs, err.Error())
		}
		if len(errMsgs) > 0 {
			return fmt.Errorf("errors recording prices: %v", errMsgs)
		}
		return nil
	}
}

func (r *PriceRecorder) recordSymbolPrice(ctx context.Context, symbol, timeframe string) error {
	klines, err := r.client.NewKlinesService().
		Symbol(symbol).
		Interval(timeframe).
		Limit(1).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to get kline: %w", err)
	}

	if len(klines) == 0 {
		return fmt.Errorf("no kline data received")
	}

	k := klines[0]
	price := &models.Price{
		Symbol:    symbol,
		TimeFrame: timeframe,
		OpenTime:  time.Unix(k.OpenTime/1000, 0),
		Open:      parseFloat(k.Open),
		High:      parseFloat(k.High),
		Low:       parseFloat(k.Low),
		Close:     parseFloat(k.Close),
		Volume:    parseFloat(k.Volume),
	}

	if err := r.priceRepo.Create(price); err != nil {
		return fmt.Errorf("failed to save price: %w", err)
	}

	r.logger.Printf("Recorded %s price for %s: %v", timeframe, symbol, price.Close)
	return nil
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Error parsing float: %v", err)
		return 0
	}
	return f
}
