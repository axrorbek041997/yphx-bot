package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"yphx-bot/internal/bot"
	"yphx-bot/internal/config"
	"yphx-bot/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	// .env ni yuklash (topilmasa error beradi)
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	// ... qolgan kod
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.ConnectPostgres(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("DB connected ✅")

	bot, err := bot.New(cfg.Bot, db)
	if err != nil {
		log.Fatal(err)
	}

	// Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Bot run
	if err := bot.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
