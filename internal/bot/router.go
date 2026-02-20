package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(*tgbotapi.BotAPI, *tgbotapi.Message) error
type Middleware func(HandlerFunc) HandlerFunc

type SceneHandler func(api *tgbotapi.BotAPI, m *tgbotapi.Message) (done bool, err error)

type Router struct {
	api *tgbotapi.BotAPI

	commands   map[string]HandlerFunc
	middleware []Middleware

	nonCommand func(*tgbotapi.BotAPI, *tgbotapi.Message) error
	unknownCmd func(*tgbotapi.BotAPI, *tgbotapi.Message) error

	// ✅ scenes
	getScene      func(userID int64) (string, bool)
	clearScene    func(userID int64)
	sceneHandlers map[string]SceneHandler
}

func NewRouter(api *tgbotapi.BotAPI) *Router {
	return &Router{
		api:           api,
		commands:      make(map[string]HandlerFunc),
		sceneHandlers: make(map[string]SceneHandler),
	}
}

func (r *Router) Use(mw Middleware) {
	r.middleware = append(r.middleware, mw)
}

func (r *Router) Register(cmd string, handler HandlerFunc) {
	r.commands[strings.ToLower(strings.TrimSpace(cmd))] = handler
}

func (r *Router) SetNonCommandHandler(h func(*tgbotapi.BotAPI, *tgbotapi.Message) error) {
	r.nonCommand = h
}

func (r *Router) SetUnknownHandler(h func(*tgbotapi.BotAPI, *tgbotapi.Message) error) {
	r.unknownCmd = h
}

// ✅ Scenes ulash
func (r *Router) EnableScenes(
	getScene func(userID int64) (string, bool),
	clearScene func(userID int64),
	handlers map[string]SceneHandler,
) {
	r.getScene = getScene
	r.clearScene = clearScene
	r.sceneHandlers = handlers
}

func (r *Router) HandleMessage(m *tgbotapi.Message) error {
	if m == nil || m.Text == "" {
		return nil
	}

	// ✅ 1) Agar user scene ichida bo'lsa — avval scene handler ishlasin
	if r.getScene != nil && m.From != nil {
		if scene, ok := r.getScene(int64(m.From.ID)); ok {
			if h, ok := r.sceneHandlers[scene]; ok {
				done, err := h(r.api, m)
				if done && r.clearScene != nil {
					r.clearScene(int64(m.From.ID))
				}
				return err
			}
		}
	}

	// 2) Command bo'lmasa
	if !m.IsCommand() {
		if r.nonCommand != nil {
			return r.nonCommand(r.api, m)
		}
		return nil
	}

	// 3) Command routing
	cmd := strings.ToLower(m.Command())
	handler, ok := r.commands[cmd]
	if !ok {
		if r.unknownCmd != nil {
			return r.unknownCmd(r.api, m)
		}
		_, err := r.api.Send(tgbotapi.NewMessage(m.Chat.ID, "Unknown command. /help ni bosing."))
		return err
	}

	// ✅ middleware apply (outside-in)
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	return handler(r.api, m)
}
