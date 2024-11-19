package binance

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/time/rate"
)

type BinanceClient struct {
	client      *futures.Client
	rateLimiter *rate.Limiter
	httpClient  *http.Client
}

func NewBinanceClient(apiKey, secretKey string) *BinanceClient {
	// Create custom HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Create futures client with custom HTTP client
	futuresClient := futures.NewClient(apiKey, secretKey)
	futuresClient.HTTPClient = httpClient

	// Create rate limiter: 10 requests per second with burst of 20
	limiter := rate.NewLimiter(rate.Limit(10), 20)

	return &BinanceClient{
		client:      futuresClient,
		rateLimiter: limiter,
		httpClient:  httpClient,
	}
}

func (c *BinanceClient) GetKlines(ctx context.Context, symbol, interval string, startTime, endTime int64) ([]*futures.Kline, error) {
	var klines []*futures.Kline
	maxRetries := 3
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait for rate limiter
		err := c.rateLimiter.Wait(ctx)
		if err != nil {
			return nil, err
		}

		// Make API call
		klines, err = c.client.NewKlinesService().
			Symbol(symbol).
			Interval(interval).
			StartTime(startTime).
			EndTime(endTime).
			Do(ctx)

		if err == nil {
			return klines, nil
		}

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			return nil, err
		}

		// Calculate backoff duration with exponential increase
		waitTime := time.Duration(math.Pow(2, float64(attempt))) * backoff

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitTime):
			continue
		}
	}

	return klines, nil
}

func (c *BinanceClient) GetHistoricalKlines(ctx context.Context, symbol, interval string, days int) ([]*futures.Kline, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// Convert to milliseconds
	startTimeMs := startTime.UnixNano() / int64(time.Millisecond)
	endTimeMs := endTime.UnixNano() / int64(time.Millisecond)

	// Split request into smaller chunks (1 day each)
	var allKlines []*futures.Kline
	chunkSize := 24 * time.Hour

	for currentStart := startTimeMs; currentStart < endTimeMs; {
		currentEnd := currentStart + chunkSize.Milliseconds()
		if currentEnd > endTimeMs {
			currentEnd = endTimeMs
		}

		klines, err := c.GetKlines(ctx, symbol, interval, currentStart, currentEnd)
		if err != nil {
			return nil, err
		}

		allKlines = append(allKlines, klines...)
		currentStart = currentEnd

		// Small delay between chunks to avoid overwhelming the API
		time.Sleep(100 * time.Millisecond)
	}

	return allKlines, nil
}
