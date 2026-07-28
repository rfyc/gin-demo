package cache

import (
	"fmt"
	"gin-demo/src/core/conf"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
)

type RedisPool struct {
	pool *redis.Pool
	mini *miniredis.Miniredis
}

func NewRedisPool(cfg *conf.RedisCfg) (*RedisPool, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("Redis Addr 为空，如需本地 mock 请使用 NewMockRedisPool")
	}
	pool := &RedisPool{}
	if err := pool.initRealRedis(cfg); err != nil {
		return nil, err
	}
	return pool, nil
}

func NewMockRedisPool(cfg *conf.RedisCfg) (*RedisPool, error) {
	pool := &RedisPool{}
	if err := pool.initMockRedis(cfg); err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *RedisPool) initRealRedis(cfg *conf.RedisCfg) error {
	pool := p.buildPool(cfg.Addr, cfg, false)

	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return fmt.Errorf("Redis PING 失败 [addr=%s]: %w", cfg.Addr, err)
	}

	p.pool = pool
	p.mini = nil
	return nil
}

func (p *RedisPool) initMockRedis(cfg *conf.RedisCfg) error {
	mini, err := startMiniRedis(cfg)
	if err != nil {
		return err
	}
	addr := mini.Addr()
	pool := p.buildPool(addr, cfg, true)

	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		mini.Close()
		return fmt.Errorf("Mock Redis PING 失败 [addr=%s]: %w", addr, err)
	}

	p.pool = pool
	p.mini = mini
	return nil
}

func (p *RedisPool) buildPool(addr string, cfg *conf.RedisCfg, isMock bool) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     cfg.PoolSize / 5,
		MaxActive:   cfg.PoolSize,
		IdleTimeout: time.Duration(cfg.IdleTimeout) * time.Second,
		Wait:        true,
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
			conn, err := redis.Dial("tcp", addr, opts...)
			if err != nil {
				return nil, fmt.Errorf("连接 Redis 失败 [addr=%s, mock=%v]: %w", addr, isMock, err)
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if isMock {
				return nil
			}
			if time.Since(t) < 60*time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func (p *RedisPool) Get() redis.Conn {
	return p.pool.Get()
}

func (p *RedisPool) Close() error {
	err := p.pool.Close()
	if p.mini != nil {
		p.mini.Close()
	}
	return err
}

func startMiniRedis(cfg *conf.RedisCfg) (*miniredis.Miniredis, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("启动 miniredis 失败: %w", err)
	}
	if cfg.Password != "" {
		mr.RequireAuth(cfg.Password)
	}
	return mr, nil
}
