package cache

import (
	"fmt"
	"gin-demo/src/core/conf"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
)

// RedisPool 封装 redigo 连接池。
// 真实 Redis 模式下 mini 为 nil；mock 模式下 mini 保存 miniredis 实例句柄，Close 时需停止。
type RedisPool struct {
	pool *redis.Pool
	mini *miniredis.Miniredis
}

// NewRedisPool 创建真实 Redis 连接池。
// 要求 cfg.Addr 必须非空，否则返回错误。
// 若需要本地 mock，请使用 NewMockRedisPool。
//
// 参数:
//   - cfg: Redis 配置（Addr 必须非空）
//
// 返回:
//   - *RedisPool: 初始化后的连接池封装
//   - error: Addr 为空或初始化失败时返回错误
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

// NewMockRedisPool 创建基于 miniredis 的本地 mock Redis 连接池。
// 本函数与真实 Redis 连接逻辑完全独立，仅用于本地开发环境且无可用 Redis 时的场景。
// miniredis 是纯 Go 实现的内存 Redis，支持大部分常用命令，无需外部部署 redis-server。
//
// 参数:
//   - cfg: Redis 配置（仅使用 Password、DB、PoolSize、IdleTimeout 字段；Addr 字段被忽略）
//
// 返回:
//   - *RedisPool: 初始化后的 mock 连接池封装
//   - error: 初始化失败时返回错误
func NewMockRedisPool(cfg *conf.RedisCfg) (*RedisPool, error) {
	pool := &RedisPool{}
	if err := pool.initMockRedis(cfg); err != nil {
		return nil, err
	}
	return pool, nil
}

// initRealRedis 初始化真实 Redis 连接池。
func (p *RedisPool) initRealRedis(cfg *conf.RedisCfg) error {
	pool := p.buildPool(cfg.Addr, cfg, false, nil)

	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return fmt.Errorf("Redis PING 失败 [addr=%s]: %w", cfg.Addr, err)
	}

	p.pool = pool
	p.mini = nil
	return nil
}

// initMockRedis 启动 miniredis 并初始化对应的连接池。
func (p *RedisPool) initMockRedis(cfg *conf.RedisCfg) error {
	mini, err := startMiniRedis(cfg)
	if err != nil {
		return err
	}
	addr := mini.Addr()
	pool := p.buildPool(addr, cfg, true, mini)

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

// buildPool 根据地址和配置构建 redigo 连接池。
// isMock=true 时不启用密码校验，省去 TestOnBorrow 的周期性 PING。
func (p *RedisPool) buildPool(addr string, cfg *conf.RedisCfg, isMock bool, mini *miniredis.Miniredis) *redis.Pool {
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
			if !isMock && cfg.Password != "" {
				opts = append(opts, redis.DialPassword(cfg.Password))
			}
			// mock 模式下若设置了密码也启用 AUTH（用于本地测试鉴权逻辑）
			if isMock && cfg.Password != "" {
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

// Get 从连接池取出一个 redis.Conn，使用完必须 Close。
func (p *RedisPool) Get() redis.Conn {
	return p.pool.Get()
}

// Close 关闭底层连接池；mock 模式下同时停止 miniredis 进程内服务。
func (p *RedisPool) Close() error {
	err := p.pool.Close()
	if p.mini != nil {
		p.mini.Close()
	}
	return err
}

// startMiniRedis 在当前进程内启动一个 miniredis 实例。
func startMiniRedis(cfg *conf.RedisCfg) (*miniredis.Miniredis, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("启动 miniredis 失败: %w", err)
	}
	if cfg.Password != "" {
		mr.RequireAuth(cfg.Password)
	}
	_ = cfg.DB
	return mr, nil
}
