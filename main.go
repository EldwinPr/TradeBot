package main

import (
	"CryptoTradeBot/internal/models"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"CryptoTradeBot/internal/operations/handlers"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

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
	symbols := []string{"ONDOSDT", "LINKUSDT", "DOTUSDT", "RENDERUSDT", "POLUSDT", "AAVEUSDT", "NEARUSDT", "FLOKIUSDT", "FETUSDT", "JUPUSDT"}

	handlers.PriceHandler(db, symbols)

	// Keep running until interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
