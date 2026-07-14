package cache

import (
	"fmt"
	"gin-demo/src/core/conf"
	"time"

	"github.com/gomodule/redigo/redis"
)

// RedisPool 封装 redigo 连接池。
type RedisPool struct {
	pool *redis.Pool
}

func (p *RedisPool) Init(cfg *conf.RedisCfg) (err error) {
	if p.pool, err = initRedis(cfg); err != nil {
		return err
	}
	return
}

func NewRedisPool(cfg *conf.RedisCfg) (*RedisPool, error) {
	var pool = &RedisPool{}
	if err := pool.Init(cfg); err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *RedisPool) Close() error {
	return p.pool.Close()
}

// NewRedisPool 根据配置创建 redigo 连接池。
func initRedis(cfg *conf.RedisCfg) (*redis.Pool, error) {
	pool := &redis.Pool{
		MaxIdle:     cfg.PoolSize / 5, // 最大空闲连接数
		MaxActive:   cfg.PoolSize,     // 最大活跃连接数
		IdleTimeout: time.Duration(cfg.IdleTimeout) * time.Second,
		Wait:        true, // 超过 MaxActive 时阻塞等待
		Dial: func() (redis.Conn, error) {
			opts := []redis.DialOption{
				redis.DialDatabase(cfg.DB),
				redis.DialConnectTimeout(5 * time.Second),
				redis.DialReadTimeout(3 * time.Second),
				redis.DialWriteTimeout(3 * time.Second),
			}
			if cfg.Password != "" {
				opts = append(opts, redis.DialPassword(cfg.Password))
			}
			conn, err := redis.Dial("tcp", cfg.Addr, opts...)
			if err != nil {
				return nil, fmt.Errorf("连接 Redis 失败 [addr=%s]: %w", cfg.Addr, err)
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < 60*time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	// 启动时探测一次
	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return nil, fmt.Errorf("Redis PING 失败: %w", err)
	}

	return pool, nil
}
