package bot

import (
	"context"
	"log"
	"strings"
	"yphx-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func LoggingMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
			if m.From != nil {
				log.Printf("user=%d username=@%s cmd=/%s", m.From.ID, m.From.UserName, m.Command())
			} else {
				log.Printf("cmd=/%s", m.Command())
			}
			return next(api, m)
		}
	}
}

func RecoveryMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(api *tgbotapi.BotAPI, m *tgbotapi.Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic recovered: %v", r)
					_, _ = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Internal error occurred."))
				}
			}()
			return next(api, m)
		}
	}
}

func RegisteredOnlyMiddleware(users *repository.UsersRepo, allowCommands ...string) Middleware {
	allowed := make(map[string]struct{}, len(allowCommands))
	for _, c := range allowCommands {
		allowed[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
			// non-command message routing bo'lishi mumkin; biz command handlerlar uchun ishlatyapmiz
			cmd := strings.ToLower(strings.TrimSpace(m.Command()))
			if _, ok := allowed[cmd]; ok {
				return next(api, m)
			}

			if m.From == nil {
				return next(api, m)
			}

			ok, err := users.ExistsByTgUserID(context.Background(), int64(m.From.ID))
			if err != nil {
				log.Printf("auth exists error: %v", err)
				_, _ = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Auth error. Keyinroq urinib ko'ring."))
				return nil
			}

			if !ok {
				_, _ = api.Send(tgbotapi.NewMessage(m.Chat.ID,
					"Avval ro'yxatdan o'ting: /register\nYoki /start"))
				return nil
			}

			return next(api, m)
		}
	}
}
