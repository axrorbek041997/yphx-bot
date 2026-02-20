package commands

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func Help(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
	text := "Komandalar:\n/start\n/help\n/ping"
	_, err := api.Send(tgbotapi.NewMessage(m.Chat.ID, text))
	return err
}
