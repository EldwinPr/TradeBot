package repositories

import (
	"CryptoTradeBot/internal/models"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

type PriceRepository struct {
	db *gorm.DB
}

// NewPriceRepository creates a new instance of PriceRepository
func NewPriceRepository(db *gorm.DB) *PriceRepository {
	return &PriceRepository{db: db}
}

// Create adds a new Price record to the database
func (r *PriceRepository) Create(price *models.Price) error {
	if price == nil {
		return errors.New("price cannot be nil")
	}
	return r.db.Create(price).Error
}

// FindByID retrieves a Price record by its ID
func (r *PriceRepository) FindByID(id uint) (*models.Price, error) {
	if id == 0 {
		return nil, errors.New("invalid id")
	}
	var price models.Price
	err := r.db.First(&price, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &price, err
}

// Update modifies an existing Price record
func (r *PriceRepository) Update(price *models.Price) error {
	if price == nil {
		return errors.New("price cannot be nil")
	}
	return r.db.Save(price).Error
}

// Delete removes a Price record from the database
func (r *PriceRepository) Delete(price *models.Price) error {
	if price == nil {
		return errors.New("price cannot be nil")
	}
	return r.db.Delete(price).Error
}

// FindAll retrieves all Price records
func (r *PriceRepository) FindAll() ([]models.Price, error) {
	var prices []models.Price
	err := r.db.Find(&prices).Error
	return prices, err
}

// GetPriceHistory gets price history for a symbol within a time range
func (r *PriceRepository) GetPriceHistory(symbol string, start, end time.Time) ([]models.Price, error) {
	var prices []models.Price
	err := r.db.Where("symbol = ? AND timestamp BETWEEN ? AND ?",
		symbol, start, end).
		Order("timestamp ASC").
		Find(&prices).Error
	return prices, err
}

// GetPriceRange gets min/max prices for a symbol within a time range
func (r *PriceRepository) GetPriceRange(symbol string, start, end time.Time) (min, max float64, err error) {
	type Result struct {
		Min float64
		Max float64
	}
	var result Result

	err = r.db.Model(&models.Price{}).
		Select("MIN(price) as min, MAX(price) as max").
		Where("symbol = ? AND timestamp BETWEEN ? AND ?",
			symbol, start, end).
		Scan(&result).Error

	return result.Min, result.Max, err
}

// GetOHLCV gets OHLCV data for a specific timeframe
func (r *PriceRepository) GetOHLCV(symbol string, start, end time.Time, interval string) ([]models.Price, error) {
	var prices []models.Price
	err := r.db.Where("symbol = ? AND timestamp BETWEEN ? AND ?",
		symbol, start, end).
		Select("timestamp, open, high, low, close, volume").
		Order("timestamp ASC").
		Find(&prices).Error
	return prices, err
}

// GetAveragePrice gets average price over a time period
func (r *PriceRepository) GetAveragePrice(symbol string, start, end time.Time) (float64, error) {
	var avg struct {
		Avg float64
	}
	err := r.db.Model(&models.Price{}).
		Select("AVG(price) as avg").
		Where("symbol = ? AND timestamp BETWEEN ? AND ?",
			symbol, start, end).
		Scan(&avg).Error
	return avg.Avg, err
}

// GetMultiSymbolPrices gets latest prices for multiple symbols
func (r *PriceRepository) GetMultiSymbolPrices(symbols []string) (map[string]*models.Price, error) {
	var prices []models.Price
	err := r.db.Where("symbol IN ?", symbols).
		Group("symbol").
		Having("timestamp = MAX(timestamp)").
		Find(&prices).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]*models.Price)
	for i := range prices {
		result[prices[i].Symbol] = &prices[i]
	}
	return result, nil
}

// GetPricesByTimeFrame gets price data for a specific symbol and timeframe
func (r *PriceRepository) GetPricesByTimeFrame(symbol string, timeFrame string, start, end time.Time) ([]models.Price, error) {
	if symbol == "" || timeFrame == "" {
		return nil, errors.New("invalid symbol or timeframe")
	}

	var prices []models.Price
	err := r.db.Where("symbol = ? AND time_frame = ? AND open_time BETWEEN ? AND ?",
		symbol, timeFrame, start, end).
		Order("open_time ASC").
		Find(&prices).Error

	// Log the query results
	log.Printf("Got %d prices for %s from %s to %s",
		len(prices),
		symbol,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"))

	return prices, err
}

// GetLatestPriceByTimeFrame gets the most recent price for a symbol and timeframe
func (r *PriceRepository) GetLatestPrice(symbol string) (*models.Price, error) {
	if symbol == "" {
		return nil, errors.New("invalid symbol")
	}

	var price models.Price
	err := r.db.Where("symbol = ?", symbol).
		Order("open_time DESC").
		First(&price).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &price, err
}

func (r *PriceRepository) GetLatestPriceByTimeFrame(symbol, timeFrame string) (*models.Price, error) {
	if symbol == "" || timeFrame == "" {
		return nil, errors.New("invalid symbol or timeframe")
	}

	var price models.Price
	err := r.db.Where("symbol = ? AND time_frame = ?", symbol, timeFrame).
		Order("open_time DESC").
		First(&price).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &price, err
}
