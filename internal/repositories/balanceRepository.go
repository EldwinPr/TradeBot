package repositories

import (
	"CryptoTradeBot/internal/models"
	"errors"

	"gorm.io/gorm"
)

type BalanceRepository struct {
	db *gorm.DB
}

// NewBalanceRepository creates a new instance of BalanceRepository
func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// Create adds a new Balance record to the database
func (r *BalanceRepository) Create(balance *models.Balance) error {
	if balance == nil {
		return errors.New("balance cannot be nil")
	}
	return r.db.Create(balance).Error
}

// FindByID retrieves a Balance record by its ID
func (r *BalanceRepository) FindByID(id uint) (*models.Balance, error) {
	if id == 0 {
		return nil, errors.New("invalid ID")
	}
	var balance models.Balance
	err := r.db.First(&balance, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &balance, err
}

// Update modifies an existing Balance record
func (r *BalanceRepository) Update(balance *models.Balance) error {
	if balance == nil {
		return errors.New("balance cannot be nil")
	}
	return r.db.Save(balance).Error
}

// Delete removes a Balance record from the database
func (r *BalanceRepository) Delete(balance *models.Balance) error {
	if balance == nil {
		return errors.New("balance cannot be nil")
	}
	return r.db.Delete(balance).Error
}

// FindAll retrieves all Balance records
func (r *BalanceRepository) FindAll() ([]models.Balance, error) {
	var balances []models.Balance
	err := r.db.Find(&balances).Error
	return balances, err
}

// FindBySymbol retrieves balance for a specific symbol
func (r *BalanceRepository) FindBySymbol(symbol string) (*models.Balance, error) {
	if symbol == "" {
		return nil, errors.New("invalid symbol")
	}
	var balance models.Balance
	err := r.db.Where("symbol = ?", symbol).First(&balance).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &balance, err
}

// FindByUserID retrieves all balances for a specific user
func (r *BalanceRepository) FindByUserID(userID uint) ([]models.Balance, error) {
	var balances []models.Balance
	err := r.db.Where("user_id = ?", userID).Find(&balances).Error
	return balances, err
}

// UpdateAmount updates the balance amount for a specific record
func (r *BalanceRepository) UpdateAmount(id uint, amount float64) error {
	if id == 0 {
		return errors.New("invalid ID")
	}
	return r.db.Model(&models.Balance{}).Where("id = ?", id).
		Update("amount", amount).Error
}
