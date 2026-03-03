package bot

import (
	"database/sql"
	"log"
	"yphx-bot/internal/ai"
	"yphx-bot/internal/bot/commands"
	"yphx-bot/internal/bot/handlers"
	"yphx-bot/internal/bot/middleware"
	"yphx-bot/internal/bot/scenes"
	"yphx-bot/internal/repository"

	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"
)

type Router struct {
	bot       *tele.Bot
	redis     *redis.Client
	db        *sql.DB
	aiBaseURL string
}

func NewRouter(bot *Bot, redis *redis.Client, db *sql.DB) *Router {
	return &Router{bot: bot.bot, redis: redis, db: db, aiBaseURL: bot.cfg.AIToolBaseURL}
}

func (r *Router) SetupRoutes() {
	scManager := scenes.NewManager()
	aiClient, err := ai.NewClient(r.aiBaseURL)
	var vectorHandlers *handlers.VectorHandlers
	if err != nil {
		log.Printf("AI client init failed: %v", err)
	} else {
		vectorHandlers = handlers.NewVectorHandlers(aiClient, repository.NewVectorsRepo(r.db))
	}

	r.bot.Use(middleware.SceneMiddleware(r.redis, r.db, scManager))
	r.bot.Use(middleware.AuthMiddleware(r.redis, r.db, scManager))

	r.bot.Handle("/start", commands.Help)
	r.bot.Handle("/help", commands.Help)
	r.bot.Handle(tele.OnText, func(c tele.Context) error {
		if vectorHandlers == nil {
			return c.Send("AI service sozlanmagan.")
		}
		return vectorHandlers.HandleText(c)
	})
	// r.bot.Handle(tele.OnAudio, func(c tele.Context) error {
	// 	if vectorHandlers == nil {
	// 		return c.Send("AI service sozlanmagan.")
	// 	}
	// 	return vectorHandlers.HandleAudio(c)
	// })
	// r.bot.Handle(tele.OnVoice, func(c tele.Context) error {
	// 	if vectorHandlers == nil {
	// 		return c.Send("AI service sozlanmagan.")
	// 	}
	// 	return vectorHandlers.HandleAudio(c)
	// })
	r.bot.Handle(tele.OnContact, commands.Help)
	r.bot.Handle(tele.OnCallback, func(c tele.Context) error {
		return c.Respond()
	})
}
