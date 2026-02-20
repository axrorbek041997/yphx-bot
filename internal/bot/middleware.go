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

func RegisteredOnlyMiddleware(
	users *repository.UsersRepo,
	setScene func(userID int64, scene string),
	startRegister func(api *tgbotapi.BotAPI, m *tgbotapi.Message) error,
	allowCommands ...string,
) Middleware {
	allowed := make(map[string]struct{}, len(allowCommands))
	for _, c := range allowCommands {
		allowed[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
			// Allow-list commandlar
			cmd := strings.ToLower(strings.TrimSpace(m.Command()))
			if _, ok := allowed[cmd]; ok {
				return next(api, m)
			}

			// Safety
			if m.From == nil {
				return next(api, m)
			}

			uid := int64(m.From.ID)

			ok, err := users.ExistsByTgUserID(context.Background(), uid)
			if err != nil {
				log.Printf("auth exists error: %v", err)
				_, _ = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Auth error. Keyinroq urinib ko'ring."))
				return nil
			}

			// ❗️Not registered → register sceneni avtomatik boshlaymiz
			if !ok {
				setScene(uid, "register")
				return startRegister(api, m) // bu "Ismingizni kiriting..." deb yuboradi
			}

			return next(api, m)
		}
	}
}
