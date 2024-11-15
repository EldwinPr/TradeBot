package repositories

import (
	"CryptoTradeBot/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository creates a new instance of TransactionRepository
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create adds a new Transaction record to the database
func (r *TransactionRepository) Create(transaction *models.Transaction) error {
	if transaction == nil {
		return errors.New("transaction cannot be nil")
	}
	return r.db.Create(transaction).Error
}

// FindByID retrieves a Transaction record by its ID
func (r *TransactionRepository) FindByID(id uint) (*models.Transaction, error) {
	if id == 0 {
		return nil, errors.New("invalid id")
	}
	var transaction models.Transaction
	err := r.db.First(&transaction, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &transaction, err
}

// Update modifies an existing Transaction record
func (r *TransactionRepository) Update(transaction *models.Transaction) error {
	if transaction == nil {
		return errors.New("transaction cannot be nil")
	}
	return r.db.Save(transaction).Error
}

// Delete removes a Transaction record from the database
func (r *TransactionRepository) Delete(transaction *models.Transaction) error {
	if transaction == nil {
		return errors.New("transaction cannot be nil")
	}
	return r.db.Delete(transaction).Error
}

// FindAll retrieves all Transaction records
func (r *TransactionRepository) FindAll() ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.Find(&transactions).Error
	return transactions, err
}

// FindBySymbol retrieves all Transaction records by symbol
func (r *TransactionRepository) FindBySymbol(symbol string) ([]models.Transaction, error) {
	if symbol == "" {
		return nil, errors.New("invalid symbol")
	}
	var transactions []models.Transaction
	err := r.db.Where("symbol = ?", symbol).Find(&transactions).Error
	return transactions, err
}

// GetTransactionsByTimeRange retrieves all Transaction records within a time range
func (r *TransactionRepository) GetTransactionsByTimeRange(start, end time.Time) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.Where("created_at BETWEEN ? AND ?", start, end).
		Order("created_at ASC").
		Find(&transactions).Error
	return transactions, err
}

// GetTotalVolume retrieves the total volume of transactions within a time range
func (r *TransactionRepository) GetTotalVolume(start, end time.Time) (float64, error) {
	var totalVolume float64
	err := r.db.Model(&models.Transaction{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Select("SUM(amount) as total_volume").
		Scan(&totalVolume).Error
	return totalVolume, err
}
