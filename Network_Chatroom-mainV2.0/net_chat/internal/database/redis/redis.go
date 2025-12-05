package redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

// Rdb redis客户端
var Rdb *redis.Client

// Rctx 通用上下文
var Rctx = context.Background()

// InitRedis 初始化Redis客户端
func InitRedis(addr, pass string, dbNum int) error {
	//不要使用:=导致Rdb成为局部变量
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       dbNum,
	})
	if err := Rdb.Ping(Rctx).Err(); err != nil {
		return fmt.Errorf("redis 连接失败 %w", err)
	}
	return nil
}
