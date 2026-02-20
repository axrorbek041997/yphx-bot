package config

import "time"

type DatabaseConfig struct {
	DSN          string        // postgres://user:pass@host:5432/db?sslmode=disable
	MaxOpenConns int           // pool: max open connections
	MaxIdleConns int           // pool: max idle connections
	ConnMaxIdle  time.Duration // pool: idle time
	ConnMaxLife  time.Duration // pool: max lifetime
	PingTimeout  time.Duration // startup ping timeout
}
