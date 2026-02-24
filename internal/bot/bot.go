package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"yphx-bot/internal/config"

	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	cfg   config.BotConfig
	db    *sql.DB
	bot   *tele.Bot
	redis *redis.Client
}

func New(cfg config.BotConfig, db *sql.DB, redis *redis.Client) (*Bot, error) {
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
		cfg:   cfg,
		db:    db,
		bot:   bot,
		redis: redis,
	}

	router := NewRouter(b, redis, db)
	router.SetupRoutes()

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

	done := make(chan struct{})

	go func() {
		defer close(done)
		b.bot.Start() // blocks until Stop() is called (ideally)
	}()

	select {
	case <-ctx.Done():
		log.Printf("Bot polling stopping: %v", ctx.Err())

		// Try to stop telebot
		b.bot.Stop()

		// Don't wait forever
		select {
		case <-done:
			log.Printf("Bot polling stopped")
		case <-time.After(3 * time.Second):
			log.Printf("Bot polling stop timeout; exiting anyway")
		}

		return ctx.Err()

	case <-done:
		log.Printf("Bot polling stopped")
		return nil
	}
}
