// internal/bot/commands/start.go
package commands

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Start(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
	name := strings.TrimSpace(m.From.FirstName + " " + m.From.LastName)
	if name == "" {
		name = "do'st"
	}

	text := fmt.Sprintf(
		"Salom, %s! 👋\n\n"+
			"Bot ishlayapti ✅\n\n"+
			"Buyruqlar:\n"+
			"• /help — yordam\n"+
			"• /ping — tekshirish\n\n"+
			"Istalgan matn yozsang, echo qilib qaytaraman.",
		name,
	)

	msg := tgbotapi.NewMessage(m.Chat.ID, text)
	return sendSafe(api, msg)
}

// sendSafe: message yuborishda common optionsni bitta joyga yig'ish uchun
func sendSafe(api *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) error {
	// msg.ParseMode = "HTML" // kerak bo'lsa yoqasiz
	_, err := api.Send(msg)
	return err
}
