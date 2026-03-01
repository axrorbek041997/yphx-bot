package bot

import (
	"database/sql"
	"yphx-bot/internal/bot/commands"
	"yphx-bot/internal/bot/middleware"
	"yphx-bot/internal/bot/scenes"

	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"
)

type Router struct {
	bot   *tele.Bot
	redis *redis.Client
	db    *sql.DB
}

func NewRouter(bot *Bot, redis *redis.Client, db *sql.DB) *Router {
	return &Router{bot: bot.bot, redis: redis, db: db}
}

func (r *Router) SetupRoutes() {
	scManager := scenes.NewManager()

	r.bot.Use(middleware.SceneMiddleware(r.redis, r.db, scManager))
	r.bot.Use(middleware.AuthMiddleware(r.redis, r.db, scManager))

	r.bot.Handle(tele.OnText, commands.Help)
	r.bot.Handle(tele.OnContact, commands.Help)
}
