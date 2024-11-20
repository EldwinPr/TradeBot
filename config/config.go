package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func Load() (*config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	return &config{
		Exchange: ExchangeConfig{
			APIKey:    os.Getenv("BINANCE_API_KEY"),
			SecretKey: os.Getenv("BINANCE_SECRET_KEY"),
		},
		Database: DatabaseConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     EnvtoInt(os.Getenv("DB_PORT")),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
		},
		Symbols: getSymbols(),
	}, nil
}

// helper env(string) to int
func EnvtoInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// helper to get symbols
func getSymbols() []string {
	symbols := os.Getenv("TRADING_SYMBOLS")
	if symbols == "" {
		return []string{"BTCUSDT", "ETHUSDT"} // Default pairs if none specified
	}
	return strings.Split(symbols, ",")
}
