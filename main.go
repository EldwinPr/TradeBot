package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"CryptoTradeBot/config"
	"CryptoTradeBot/internal/handlers"
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/repositories"

	"github.com/adshao/go-binance/v2/futures"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Setup database
	db := setupDatabase(cfg.Database)

	// Drop existing prices table if exists
	err = db.Exec("DROP TABLE IF EXISTS prices").Error
	if err != nil {
		log.Fatal("Failed to drop prices table:", err)
	}

	// Create fresh prices table
	err = db.AutoMigrate(&models.Price{})
	if err != nil {
		log.Fatal("Failed to create prices table:", err)
	}

	// Initialize repository
	priceRepo := repositories.NewPriceRepository(db)

	// Initialize Binance client
	futuresClient := futures.NewClient(cfg.Exchange.APIKey, cfg.Exchange.SecretKey)

	// Initialize price handler
	priceHandler := handlers.NewPriceHandler(futuresClient, priceRepo, cfg.Symbols)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start price handling
	if err := priceHandler.Start(ctx); err != nil {
		log.Fatal("Failed to start price handler:", err)
	}

	log.Println("Price recording started...")

	// Handle shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	log.Println("Shutting down...")
	cancel()
	time.Sleep(time.Second * 2) // Give time for cleanup
	log.Println("Shutdown complete")
}

func setupDatabase(dbConfig config.DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate database schemas
	err = db.AutoMigrate(&models.Price{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	return db
}
