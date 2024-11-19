package priceOperations

import (
	"CryptoTradeBot/internal/models"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/time/rate"
)

type PriceFetcher struct {
	client     *futures.Client
	symbols    []string
	limiter    *rate.Limiter
	logger     *log.Logger
	retryDelay time.Duration
	maxRetries int
}

// NewPriceFetcher creates a new instance of PriceFetcher with rate limiting
func NewPriceFetcher(client *futures.Client, symbols []string) *PriceFetcher {
	return &PriceFetcher{
		client:     client,
		symbols:    symbols,
		limiter:    rate.NewLimiter(rate.Limit(10), 20), // 10 requests/sec with burst of 20
		logger:     log.New(log.Writer(), "[PriceFetcher] ", log.LstdFlags),
		retryDelay: 100 * time.Millisecond,
		maxRetries: 3,
	}
}

// GetHistoricalPrices retrieves historical price data with improved error handling and rate limiting
func (f *PriceFetcher) GetHistoricalPrices(ctx context.Context, timeframe string, days int) ([]models.Price, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	var allPrices []models.Price
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := make(chan error, len(f.symbols))

	for _, symbol := range f.symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			prices, err := f.fetchSymbolHistory(ctx, sym, timeframe, startTime, endTime)
			if err != nil {
				errors <- fmt.Errorf("error fetching %s data for %s: %w", timeframe, sym, err)
				return
			}

			mu.Lock()
			allPrices = append(allPrices, prices...)
			mu.Unlock()
		}(symbol)
	}

	// Wait for all goroutines or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errors)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		// Check for any errors
		var errMsgs []string
		for err := range errors {
			errMsgs = append(errMsgs, err.Error())
		}
		if len(errMsgs) > 0 {
			return allPrices, fmt.Errorf("errors fetching historical data: %v", errMsgs)
		}
		return allPrices, nil
	}
}

func (f *PriceFetcher) fetchSymbolHistory(ctx context.Context, symbol, timeframe string, startTime, endTime time.Time) ([]models.Price, error) {
	var prices []models.Price
	chunkDuration := 24 * time.Hour // Fetch in 1-day chunks

	currentStart := startTime
	for currentStart.Before(endTime) {
		currentEnd := currentStart.Add(chunkDuration)
		if currentEnd.After(endTime) {
			currentEnd = endTime
		}

		// Wait for rate limiter
		if err := f.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		chunk, err := f.fetchWithRetry(ctx, symbol, timeframe, currentStart, currentEnd)
		if err != nil {
			return nil, err
		}

		prices = append(prices, chunk...)
		currentStart = currentEnd
	}

	return prices, nil
}

func (f *PriceFetcher) fetchWithRetry(ctx context.Context, symbol, timeframe string, start, end time.Time) ([]models.Price, error) {
	var lastErr error

	for attempt := 0; attempt <= f.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * f.retryDelay):
				// Exponential backoff
			}
		}

		klines, err := f.client.NewKlinesService().
			Symbol(symbol).
			Interval(timeframe).
			StartTime(start.UnixNano() / int64(time.Millisecond)).
			EndTime(end.UnixNano() / int64(time.Millisecond)).
			Do(ctx)

		if err == nil {
			return f.convertKlinesToPrices(symbol, timeframe, klines), nil
		}

		lastErr = err
		f.logger.Printf("Attempt %d failed for %s: %v", attempt+1, symbol, err)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", f.maxRetries, lastErr)
}

func (f *PriceFetcher) convertKlinesToPrices(symbol, timeframe string, klines []*futures.Kline) []models.Price {
	prices := make([]models.Price, len(klines))
	for i, k := range klines {
		prices[i] = models.Price{
			Symbol:    symbol,
			TimeFrame: timeframe,
			OpenTime:  time.Unix(k.OpenTime/1000, 0),
			Open:      parseFloat(k.Open),
			High:      parseFloat(k.High),
			Low:       parseFloat(k.Low),
			Close:     parseFloat(k.Close),
			Volume:    parseFloat(k.Volume),
		}
	}
	return prices
}
