package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"yphx-bot/internal/bot/commands"
	"yphx-bot/internal/bot/scenes"
	"yphx-bot/internal/config"
	"yphx-bot/internal/repository"
)

type Bot struct {
	cfg    config.BotConfig
	db     *sql.DB
	api    *tgbotapi.BotAPI
	router *Router
}

func New(cfg config.BotConfig, db *sql.DB) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is empty")
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("new telegram bot: %w", err)
	}
	api.Debug = cfg.Debug

	b := &Bot{
		cfg: cfg,
		db:  db,
		api: api,
	}

	usersRepo := repository.NewUsersRepo(db)

	sceneMgr := scenes.NewManager()
	regScene := scenes.NewRegisterScene(usersRepo)

	// Router setup
	r := NewRouter(api)
	r.Use(RecoveryMiddleware())
	r.Use(LoggingMiddleware())

	// Register commands
	r.Register("start", commands.Start)
	// r.Register("help", commands.Help)
	// r.Register("ping", commands.Ping)

	// /register command: scene start qiladi
	r.Register("register", func(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
		return commands.Register(api, m, regScene, func(userID int64, name string) {
			sceneMgr.Set(userID, name)
		})
	})

	// Optional: non-command messages handler
	// r.SetNonCommandHandler(commands.Echo)

	// Optional: unknown command handler
	// r.SetUnknownHandler(commands.Unknown)

	b.router = r

	log.Printf("telegram bot authorized as @%s", api.Self.UserName)
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

	u := tgbotapi.NewUpdate(0)
	u.Timeout = int(timeout.Seconds())

	updates := b.api.GetUpdatesChan(u)
	log.Printf("bot polling started (timeout=%s)", timeout)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			log.Println("bot stopped (context cancelled)")
			return nil
		case upd, ok := <-updates:
			if !ok {
				return nil
			}
			if upd.Message != nil {
				if err := b.router.HandleMessage(upd.Message); err != nil {
					log.Printf("router error: %v", err)
				}
			}
		}
	}
}
