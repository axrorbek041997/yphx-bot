package config

import "time"

type BotConfig struct {
	Token string

	Mode  string // "polling" yoki "webhook"
	Debug bool

	// Polling uchun
	PollTimeout time.Duration // masalan 30s

	// Webhook uchun (keyin)
	WebhookURL    string // https://domain.com/bot/webhook
	WebhookSecret string // ixtiyoriy
	ListenAddr    string // ":8080"
}
