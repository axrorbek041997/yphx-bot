package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App   AppConfig
	DB    DatabaseConfig
	Bot   BotConfig
	REDIS RedisConfig
}

type AppConfig struct {
	Env string
}

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Env: getEnv("APP_ENV", "dev"),
		},
		Bot: BotConfig{
			Token: mustEnv("BOT_TOKEN"),
		},
		DB: DatabaseConfig{
			DSN:          mustEnv("DB_DSN"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxIdle:  getEnvDuration("DB_CONN_MAX_IDLE", 5*time.Minute),
			ConnMaxLife:  getEnvDuration("DB_CONN_MAX_LIFETIME", 60*time.Minute),
			PingTimeout:  getEnvDuration("DB_PING_TIMEOUT", 5*time.Second),
		},
		REDIS: RedisConfig{
			Host:        getEnv("REDIS_HOST", "localhost"),
			Port:        getEnvInt("REDIS_PORT", 6379),
			Db:          getEnvInt("REDIS_DB", 0),
			Password:    getEnv("REDIS_PASSWORD", ""),
			PingTimeout: getEnvDuration("REDIS_PING_TIMEOUT", 5*time.Second),
		},
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Bot.Token == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}
	if c.DB.DSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	return os.Getenv(key)
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
