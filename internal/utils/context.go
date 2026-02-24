package utils

import (
	"context"
	"time"
)

func RedisCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}
