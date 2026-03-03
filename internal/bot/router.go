package bot

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"
	"yphx-bot/internal/ai"
	"yphx-bot/internal/bot/commands"
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
	userRepo := repository.NewUsersRepo(r.db)
	vectorRepo := repository.NewVectorsRepo(r.db)
	searchLogRepo := repository.NewSearchLogsRepo(r.db)

	aiClient, err := ai.NewClient(r.aiBaseURL)
	var searchScene *scenes.SearchScene
	var addVectorScene *scenes.AddVectorScene
	if err != nil {
		log.Printf("AI client init failed: %v", err)
	} else {
		searchScene = scenes.NewSearchScene(aiClient, vectorRepo, searchLogRepo, userRepo)
		addVectorScene = scenes.NewAddVectorScene(aiClient, vectorRepo)
	}

	r.bot.Use(middleware.SceneMiddleware(r.redis, r.db, scManager))
	r.bot.Use(middleware.AuthMiddleware(r.redis, r.db, scManager))

	r.bot.Handle("/start", func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		exists, err := userRepo.ExistsByTgUserID(ctx, c.Sender().ID)
		if err != nil {
			return c.Send("Internal error. Please try again later.")
		}
		if !exists {
			register := scenes.NewRegisterScene(userRepo)
			scManager.Set(c.Sender().ID, register)
			return register.Start(c)
		}

		role, err := userRepo.GetRoleByTgUserID(ctx, c.Sender().ID)
		if err != nil {
			return c.Send("Internal error. Please try again later.")
		}

		markup := &tele.ReplyMarkup{
			ResizeKeyboard: true,
			ReplyKeyboard: [][]tele.ReplyButton{
				{{Text: scenes.SearchButtonText}},
			},
		}
		if role == "admin" {
			markup.ReplyKeyboard = [][]tele.ReplyButton{
				{{Text: scenes.SearchButtonText}, {Text: scenes.AddVectorButtonText}},
			}
		}
		return c.Send("Kerakli tugmani tanlang:", markup)
	})
	r.bot.Handle("/help", commands.Help)
	r.bot.Handle(tele.OnText, func(c tele.Context) error {
		text := strings.TrimSpace(c.Text())
		if text == scenes.SearchButtonText {
			if searchScene == nil {
				return c.Send("AI service sozlanmagan.")
			}
			scManager.Set(c.Sender().ID, searchScene)
			return searchScene.Start(c)
		}
		if text == scenes.AddVectorButtonText {
			if addVectorScene == nil {
				return c.Send("AI service sozlanmagan.")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			role, err := userRepo.GetRoleByTgUserID(ctx, c.Sender().ID)
			if err != nil {
				return c.Send("Internal error. Please try again later.")
			}
			if role != "admin" {
				return c.Send("Faqat admin add vector qila oladi.")
			}

			scManager.Set(c.Sender().ID, addVectorScene)
			return addVectorScene.Start(c)
		}
		return c.Send("/start ni bosing va Search tugmasini tanlang.")
	})
	r.bot.Handle(tele.OnPhoto, func(c tele.Context) error {
		return c.Send("/start ni bosing va Search tugmasini tanlang.")
	})
	r.bot.Handle(tele.OnContact, commands.Help)
	r.bot.Handle(tele.OnCallback, func(c tele.Context) error {
		if cb := c.Callback(); cb != nil && strings.HasPrefix(cb.Data, "search_react:") {
			if searchScene == nil {
				return c.Respond()
			}
			return searchScene.HandleReactionCallback(c)
		}
		return c.Respond()
	})
}
