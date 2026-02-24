package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"yphx-bot/internal/config"

	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	cfg config.BotConfig
	db  *sql.DB
	bot *tele.Bot
}

func New(cfg config.BotConfig, db *sql.DB) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is empty")
	}

	pref := tele.Settings{
		Token:  cfg.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	b := &Bot{
		cfg: cfg,
		db:  db,
		bot: bot,
	}

	bot.Handle("/hello", func(c tele.Context) error {
		return c.Send("Hello!")
	})

	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	mode := strings.ToLower(strings.TrimSpace(b.cfg.Mode))
	if mode == "" {
		mode = "polling"
	}

	switch mode {
	case "polling":
		return b.runPolling(ctx)
	case "webhook":
		return fmt.Errorf("webhook mode is not implemented yet (set BOT_MODE=polling)")
	default:
		return fmt.Errorf("unknown BOT_MODE=%q", b.cfg.Mode)
	}
}

func (b *Bot) runPolling(ctx context.Context) error {
	timeout := b.cfg.PollTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	log.Printf("Bot started in polling mode with timeout=%v", timeout)

	// Run the bot in a separate goroutine
	go func() {
		b.bot.Start()
	}()

	// Wait for context cancellation (e.g., SIGINT)
	<-ctx.Done()
	log.Println("Shutting down bot...")

	b.bot.Stop()
	return nil
}
