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
	scene *scenes.Manager
}

func NewRouter(bot *Bot, redis *redis.Client, db *sql.DB) *Router {
	return &Router{bot: bot.bot, redis: redis, db: db, scene: bot.scene}
}

func (r *Router) SetupRoutes() {
	// usersRepo := repository.NewUsersRepo(r.db)
	// registerScene := scenes.NewRegisterScene(usersRepo)

	r.bot.Use(middleware.AuthMiddleware(r.redis, r.db, r.scene))

	r.bot.Handle("/help", commands.Help)
}
