package db

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"server/config"
)

/*
redis 工具类.

启动期连接策略:
  - 创建 client 不会真的拨号, 仅 Ping 才会建连
  - 失败后按指数退避重试 5 次 (1s, 2s, 4s, 8s, 16s), 最长 ~31s
  - 仍失败则 log.Fatal, 让进程退出由 docker restart 接管
  - 不再用 panic, 避免 docker compose 依赖未就绪场景下出现死循环重启
*/
var Rdb *redis.Client
var Cxt = context.Background()

const (
	redisRetryAttempts = 5
	redisRetryBase     = 1 * time.Second

	// Redis 连接池. 采集场景下 spider 协程 (config.MAXGoroutine 默认 32) 与 Web API 协程同时争用,
	// 历史值 10 形成严重排队 (单条 ZPopMax / ZAdd 都要排队 → RTT 翻倍).
	// 64 是经验值, 配合 MinIdleConns 预热可避免冷启动时的 dial 抖动.
	redisPoolSize     = 64
	redisMinIdleConns = 16
)

// InitRedisConn 初始化redis客户端, 失败重试 5 次仍不行则 log.Fatal.
func InitRedisConn() error {
	Rdb = redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDBNo,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdleConns,
		DialTimeout:  10 * time.Second,
	})

	var lastErr error
	for i := 0; i < redisRetryAttempts; i++ {
		if _, err := Rdb.Ping(Cxt).Result(); err == nil {
			return nil
		} else {
			lastErr = err
			wait := redisRetryBase << i
			log.Printf("WARN redis: Ping %s 失败 (%d/%d): %v, %s 后重试", config.RedisAddr, i+1, redisRetryAttempts, err, wait)
			time.Sleep(wait)
		}
	}
	log.Fatalf("redis: 连接 %s 失败, 最后错误: %v", config.RedisAddr, lastErr)
	return lastErr // unreachable, 让编译器满意
}

// CloseRedis 关闭redis连接
func CloseRedis() error {
	return Rdb.Close()
}
