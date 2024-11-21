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
	"CryptoTradeBot/internal/operations/backtest"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/strategy"

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
	positionRepo := repositories.NewPositionRepository(db)

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

	// Initialize strategy components
	strategyManager := strategy.NewStrategyManager()

	// Setup backtest config
	backtestConfig := backtest.Config{
		InitialBalance: backtest.InitialBalance,
		Leverage:       backtest.Leverage,
		Symbols:        cfg.Symbols, // Your symbols
		StartTime:      time.Date(2024, 11, 17, 0, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2024, 11, 20, 0, 0, 0, 0, time.UTC),
	}

	// Create and run engine
	engine := backtest.NewEngine(priceRepo, positionRepo, strategyManager, backtestConfig)
	results, err := engine.RunBacktest(
		backtestConfig.StartTime,
		backtestConfig.EndTime,
		backtestConfig.Symbols,
	)
	if err != nil {
		log.Fatal("Backtest failed:", err)
	}

	// Print results
	fmt.Println("\n=== Backtest Results ===")
	fmt.Printf("Total Trades: %d\n", results.TotalTrades)
	fmt.Printf("Winning Trades: %d (%.2f%%)\n",
		results.WinningTrades,
		float64(results.WinningTrades)/float64(results.TotalTrades)*100)
	fmt.Printf("Average PnL: $%.2f\n", results.AveragePnL)
	fmt.Printf("Max Drawdown: %.2f%%\n", results.MaxDrawdown*100)
	fmt.Printf("Final Balance: $%.2f\n", results.FinalBalance)
	fmt.Printf("Sharpe Ratio: %.2f\n", results.SharpeRatio)

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

	db.AutoMigrate(&models.Position{})

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
