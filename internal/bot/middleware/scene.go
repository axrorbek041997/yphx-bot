package middleware

import (
	"database/sql"
	"yphx-bot/internal/bot/scenes"

	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"
)

func SceneMiddleware(redis *redis.Client, db *sql.DB, scene *scenes.Manager) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			uid := c.Sender().ID

			activeScene, ok := scene.Get(uid)
			if ok {
				done, err := activeScene.Handle(c)
				if done {
					scene.Clear(uid)
				}
				return err
			}

			return next(c)
		}
	}
}
