// internal/bot/commands/register.go
package commands

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// SceneStarter - register sceneni start qiladigan interface
type SceneStarter interface {
	Start(api *tgbotapi.BotAPI, m *tgbotapi.Message) error
}

func Register(
	api *tgbotapi.BotAPI,
	m *tgbotapi.Message,
	reg SceneStarter,
	setScene func(userID int64, name string),
) error {
	if m.From == nil {
		return nil
	}
	setScene(int64(m.From.ID), "register")
	return reg.Start(api, m)
}
