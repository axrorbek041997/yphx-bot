package bot

import (
	"yphx-bot/internal/bot/commands"

	tele "gopkg.in/telebot.v3"
)

type Router struct {
	bot *tele.Bot
}

func NewRouter(bot *tele.Bot) *Router {
	return &Router{bot: bot}
}

func (r *Router) SetupRoutes() {
	r.bot.Handle("/help", commands.Help)
}
