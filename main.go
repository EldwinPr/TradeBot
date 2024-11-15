package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database connection string
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	// Test database connection
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	fmt.Println("Successfully connected to database!")

	// Initialize Binance Futures client
	client := futures.NewClient(
		os.Getenv("BINANCE_API_KEY"),
		os.Getenv("BINANCE_SECRET_KEY"),
	)

	// Get current BTC perpetual futures price
	prices, err := client.NewListPricesService().Symbol("BTCUSDT").Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get price: %v", err)
	}

	if len(prices) == 0 {
		log.Fatal("No price data available for BTCUSDT")
	}

	price, err := strconv.ParseFloat(prices[0].Price, 64)
	if err != nil {
		log.Fatalf("Error parsing price: %v", err)
	}

	fmt.Printf("Current BTC/USDT Perpetual Futures Price: $%.2f\n", price)
}
