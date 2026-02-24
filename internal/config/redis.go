package config

import "time"

type RedisConfig struct {
	Host        string        // localhost
	Port        int           // 6379
	Db          int           // 0-15
	Password    string        // optional
	PingTimeout time.Duration // startup ping timeout
}
