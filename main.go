package main

import (
	"CryptoTradeBot/internal/backtesting"
	"CryptoTradeBot/internal/models"
	"CryptoTradeBot/internal/operations/handlers"
	"CryptoTradeBot/internal/repositories"
	"CryptoTradeBot/internal/services/analysis"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Add command line flags
	mode := flag.String("mode", "live", "Trading mode: 'live' or 'backtest'")
	days := flag.Int("days", 30, "Number of days to backtest")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database setup
	db := setupDatabase()

	// Initialize repositories
	priceRepo := repositories.NewPriceRepository(db)
	positionRepo := repositories.NewPositionRepository(db)
	balanceRepo := repositories.NewBalanceRepository(db)

	// Initialize analysis
	analysis := analysis.NewAnalysis()

	symbols := []string{
		"ONDOUSDT", "LINKUSDT", "DOTUSDT", "BNBUSDT", "SOLUSDT",
		"BTCUSDT", "ETHUSDT", "XRPUSDT", "SUIUSDT", "ADAUSDT",
	}

	switch *mode {
	case "live":
		runLiveTrading(priceRepo, positionRepo, balanceRepo, analysis, symbols)
	case "backtest":
		runBacktest(priceRepo, analysis, symbols, *days)
	default:
		log.Fatal("Invalid mode. Use 'live' or 'backtest'")
	}
}

func setupDatabase() *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = db.AutoMigrate(
		&models.Price{},
		&models.Position{},
		&models.Balance{},
		&models.Transaction{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	db.Logger = db.Logger.LogMode(logger.Error)
	return db
}

func runLiveTrading(priceRepo *repositories.PriceRepository,
	positionRepo *repositories.PositionRepository,
	balanceRepo *repositories.BalanceRepository,
	analysis *analysis.Analysis,
	symbols []string) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize handlers
	priceHandler := handlers.NewPriceHandler(priceRepo)
	analysisHandler := handlers.NewAnalysisHandler(
		analysis,
		priceRepo,
		positionRepo,
		balanceRepo,
	)

	// Initialize balance
	if err := initBalance(balanceRepo); err != nil {
		log.Fatal("Failed to initialize balance:", err)
	}

	log.Println("Starting live trading...")

	// Start price handler
	if err := priceHandler.Start(ctx, symbols); err != nil {
		log.Fatal("Failed to start price handler:", err)
	}

	time.Sleep(time.Second * 10)
	go analysisHandler.Start(ctx, symbols)

	// Handle shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")
	cancel()
	time.Sleep(time.Second * 2)
	log.Println("Shutdown complete")
}

func initBalance(balanceRepo *repositories.BalanceRepository) error {
	balance, err := balanceRepo.FindBySymbol("USDT")
	if err != nil {
		return fmt.Errorf("error checking balance: %v", err)
	}

	if balance == nil {
		newBalance := &models.Balance{
			Symbol:      "USDT",
			Balance:     1000.0, // Starting with 1000 USDT
			LastUpdated: time.Now(),
		}
		if err := balanceRepo.Create(newBalance); err != nil {
			return fmt.Errorf("error creating initial balance: %v", err)
		}
	}
	return nil
}

func runBacktest(priceRepo *repositories.PriceRepository,
	analysis *analysis.Analysis,
	symbols []string,
	days int) {

	log.Printf("Starting backtest for last %d days...", days)

	// Ensure we have enough price data
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// Log the actual data we have
	for _, symbol := range symbols {
		prices, err := priceRepo.GetPricesByTimeFrame(
			symbol,
			models.PriceTimeFrame5m,
			startTime,
			endTime,
		)
		if err != nil {
			log.Printf("Error getting prices for %s: %v", symbol, err)
			continue
		}

		if len(prices) == 0 {
			log.Printf("Warning: No price data found for %s in time range %s to %s",
				symbol,
				startTime.Format("2006-01-02 15:04:05"),
				endTime.Format("2006-01-02 15:04:05"))
			continue
		}

		log.Printf("%s: Got %d candles from %s to %s",
			symbol,
			len(prices),
			prices[0].OpenTime.Format("2006-01-02 15:04:05"),
			prices[len(prices)-1].OpenTime.Format("2006-01-02 15:04:05"),
		)
	}

	bt := backtesting.NewBacktest(priceRepo, analysis)

	endTime = time.Now()
	startTime = endTime.AddDate(0, 0, -30) // 30 days

	results, err := bt.RunBacktest(startTime, endTime, symbols)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nTrade History:")
	for _, trade := range results.Trades {
		fmt.Printf("%s: %s %s Entry: %.8f Exit: %.8f PnL: %.2f\n",
			trade.EntryTime.Format("2006-01-02 15:04"),
			trade.Symbol,
			trade.Side,
			trade.EntryPrice,
			trade.ExitPrice,
			trade.PnL)
	}

	// Print results
	fmt.Println("\nBacktest Results:")
	fmt.Printf("Period: %s to %s\n", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	fmt.Printf("Total Trades: %d\n", results.TotalTrades)
	fmt.Printf("Winning Trades: %d\n", results.WinningTrades)
	fmt.Printf("Losing Trades: %d\n", results.LosingTrades)
	fmt.Printf("Win Rate: %.2f%%\n", results.WinRate*100)
	fmt.Printf("Average PnL: %.2f USDT\n", results.AveragePnL)
	fmt.Printf("Max Drawdown: %.2f%%\n", results.MaxDrawdown*100)
	fmt.Printf("Final Balance: %.2f USDT\n", results.FinalBalance)
	fmt.Printf("Sharpe Ratio: %.2f\n", results.SharpeRatio)

	// Optional: Print detailed trade history to console

}
