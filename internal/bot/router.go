package bot

import tele "gopkg.in/telebot.v3"

type Router struct {
	bot *tele.Bot
}

func NewRouter(bot *tele.Bot) *Router {
	return &Router{bot: bot}
}

func (r *Router) SetupRoutes() {
	r.bot.Handle("/hello", func(c tele.Context) error {
		return c.Send("Hello!")
	})
}
