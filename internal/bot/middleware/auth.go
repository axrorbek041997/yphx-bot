package middleware

import (
	"database/sql"
	"log"
	"yphx-bot/internal/bot/scenes"
	"yphx-bot/internal/repository"
	"yphx-bot/internal/utils"

	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"
)

func AuthMiddleware(redis *redis.Client, db *sql.DB, scene *scenes.Manager) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			// allow /start always
			if c.Text() == "/start" {
				return next(c)
			}

			uid := c.Sender().ID

			log.Print("Register middleware")
			ctx, cancel := utils.RedisCtx()
			defer cancel()

			cacheKey := utils.UserCacheKey(uid)

			// Check Redis cache first
			exists, err := redis.Exists(ctx, cacheKey).Result()
			if err != nil {
				log.Printf("Redis error: %v", err)
				return c.Send("Internal error. Please try again later.")
			}

			if exists == 0 {
				userRepo := repository.NewUsersRepo(db)
				userExists, err := userRepo.ExistsByTgUserID(ctx, uid)
				if err != nil {
					log.Printf("DB error: %v", err)
					return c.Send("Internal error. Please try again later.")
				}

				if userExists {
					// Cache the existence for future requests (e.g., 1 hour)
					if err := redis.Set(ctx, cacheKey, "1", 0).Err(); err != nil {
						log.Printf("Failed to set Redis cache: %v", err)
					}

					return next(c)
				}

				register := scenes.NewRegisterScene(userRepo)
				scene.Set(uid, register)
				register.Start(c)
				return nil
			}

			return next(c)
		}
	}
}
