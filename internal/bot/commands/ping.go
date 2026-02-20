package commands

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func Ping(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
	_, err := api.Send(tgbotapi.NewMessage(m.Chat.ID, "pong 🏓"))
	return err
}
