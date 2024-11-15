package repositories

import (
	"CryptoTradeBot/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type PositionRepository struct {
	db *gorm.DB
}

// NewPositionRepository creates a new instance of PositionRepository
func NewPositionRepository(db *gorm.DB) *PositionRepository {
	return &PositionRepository{db: db}
}

// Create adds a new Position record to the database
func (r *PositionRepository) Create(position *models.Position) error {
	if position == nil {
		return errors.New("position cannot be nil")
	}
	return r.db.Create(position).Error
}

// FindByID retrieves a Position record by its ID
func (r *PositionRepository) FindByID(id uint) (*models.Position, error) {
	if id == 0 {
		return nil, errors.New("invalid id")
	}
	var position models.Position
	err := r.db.First(&position, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &position, err
}

// Update modifies an existing Position record
func (r *PositionRepository) Update(position *models.Position) error {
	if position == nil {
		return errors.New("position cannot be nil")
	}
	return r.db.Save(position).Error
}

// Delete removes a Position record from the database
func (r *PositionRepository) Delete(position *models.Position) error {
	if position == nil {
		return errors.New("position cannot be nil")
	}
	return r.db.Delete(position).Error
}

// FindAll retrieves all Position records
func (r *PositionRepository) FindAll() ([]models.Position, error) {
	var positions []models.Position
	err := r.db.Find(&positions).Error
	return positions, err
}

// FindOpenPositions retrieves all open Position records
func (r *PositionRepository) FindOpenPositions() ([]models.Position, error) {
	var positions []models.Position
	err := r.db.Where("status = ?", models.PositionStatusOpen).Find(&positions).Error
	return positions, err
}

// FindClosedPositions retrieves all closed Position records
func (r *PositionRepository) FindPositionsBySymbol(symbol string) ([]models.Position, error) {
	if symbol == "" {
		return nil, errors.New("invalid symbol")
	}
	var positions []models.Position
	err := r.db.Where("symbol = ?", symbol).Find(&positions).Error
	return positions, err
}

// FindOpenPositionsBySymbol retrieves all open Position records for a specific symbol
func (r *PositionRepository) FindOpenPositionsBySymbol(symbol string) ([]models.Position, error) {
	if symbol == "" {
		return nil, errors.New("invalid symbol")
	}
	var positions []models.Position
	err := r.db.Where("symbol = ? AND status = ?", symbol, models.PositionStatusOpen).Find(&positions).Error
	return positions, err
}

// FindClosedPositions retrieves all closed Position records
func (r *PositionRepository) GetPositionsByTimeRange(start, end time.Time) ([]models.Position, error) {
	var positions []models.Position
	err := r.db.Where("open_time BETWEEN ? AND ?", start, end).Find(&positions).Error
	return positions, err
}

// GetTotalPnL calculates the total profit and loss for all closed positions within a time range
func (r *PositionRepository) GetTotalPnL(start, end time.Time) (float64, error) {
	var totalPnL float64
	err := r.db.Model(&models.Position{}).
		Where("close_time BETWEEN ? AND ? AND status = ?", start, end, models.PositionStatusClosed).
		Select("SUM(pnl) as total_pnl").
		Scan(&totalPnL).Error
	return totalPnL, err
}
