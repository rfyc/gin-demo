package cache

import (
	"fmt"
	"gin-demo/src/core/conf"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
)

// RedisPool 封装 redigo 连接池。
// 当 conf.RedisCfg.Addr 为空时，自动启动进程内 mini Redis（纯 Go 内存实现）作为本地 mock，
// 无需真实部署 redis-server，业务侧 SET/GET/HSET/ZADD 等常用命令行为与真实 Redis 一致。
type RedisPool struct {
	pool *redis.Pool
	// mini 是 mock 模式下的内存 Redis 实例，真实 Redis 场景下为 nil。
	// 存在的意义是 Close 时停止它（释放端口/goroutine）。
	mini *miniredis.Miniredis
}

// Init 根据 cfg 创建连接池；Addr 为空时启用 miniredis mock。
//
// 参数:
//   - cfg: Redis 配置，包括 Addr、Password、DB、PoolSize 等
//
// 返回:
//   - error: 初始化失败时返回带上下文的错误
func (p *RedisPool) Init(cfg *conf.RedisCfg) (err error) {
	var pool *redis.Pool
	var mini *miniredis.Miniredis
	if pool, mini, err = initRedis(cfg); err != nil {
		return err
	}
	p.pool = pool
	p.mini = mini
	return
}

// NewRedisPool 创建 RedisPool，并根据 cfg.Addr 决定是否启动 miniredis mock。
//
// 参数:
//   - cfg: Redis 配置
//
// 返回:
//   - *RedisPool: 初始化后的连接池封装
//   - error: 初始化失败时返回错误
func NewRedisPool(cfg *conf.RedisCfg) (*RedisPool, error) {
	var pool = &RedisPool{}
	if err := pool.Init(cfg); err != nil {
		return nil, err
	}
	return pool, nil
}

// Get 从连接池取出一个 redis.Conn。
// 封装为显式方法，方便业务代码在无 Pool 字段访问权限时取连接。
//
// 返回:
//   - redis.Conn: redigo 原生连接，使用完必须 Close
func (p *RedisPool) Get() redis.Conn {
	return p.pool.Get()
}

// Close 关闭底层连接池；如果当前处于 mock 模式，同时停止 miniredis 进程内服务。
//
// 返回:
//   - error: 关闭连接池失败时返回错误
func (p *RedisPool) Close() error {
	err := p.pool.Close()
	// mock 模式下停掉 miniredis（忽略其 Close 错误，进程退出即可回收）
	if p.mini != nil {
		p.mini.Close()
	}
	return err
}

// initRedis 根据配置创建 redigo 连接池。
// cfg.Addr 为空时：在本进程内启动一个 miniredis 内存 Redis，返回其监听地址对应的连接池
// 以及 miniredis 实例句柄（供后续 Close 使用）。
// cfg.Addr 非空时：直接走真实 Redis Dial，miniredis 句柄返回 nil。
//
// 参数:
//   - cfg: Redis 配置
//
// 返回:
//   - *redis.Pool: redigo 连接池
//   - *miniredis.Miniredis: mock 模式下的服务实例句柄；真实 Redis 时为 nil
//   - error: 初始化失败时返回带上下文的错误
func initRedis(cfg *conf.RedisCfg) (*redis.Pool, *miniredis.Miniredis, error) {
	var (
		addr   string
		mini   *miniredis.Miniredis
		isMock bool
		err    error
	)

	if cfg.Addr == "" {
		// Addr 为空 → 启用进程内 miniredis mock
		if mini, err = startMiniRedis(cfg); err != nil {
			return nil, nil, err
		}
		addr = mini.Addr()
		isMock = true
	} else {
		// Addr 非空 → 真实 Redis
		addr = cfg.Addr
	}

	pool := &redis.Pool{
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
			// mock 模式不需要密码；真实 Redis 有密码时启用
			if !isMock && cfg.Password != "" {
				opts = append(opts, redis.DialPassword(cfg.Password))
			}
			conn, err := redis.Dial("tcp", addr, opts...)
			if err != nil {
				return nil, fmt.Errorf("连接 Redis 失败 [addr=%s, mock=%v]: %w", addr, isMock, err)
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			// mock 模式下省去周期性 PING，减少无意义的系统调用
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

	// 启动时探测一次，确认连接可用
	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return nil, nil, fmt.Errorf("Redis PING 失败 [addr=%s, mock=%v]: %w", addr, isMock, err)
	}

	return pool, mini, nil
}

// startMiniRedis 在当前进程内启动一个 miniredis 实例。
// 根据 cfg.DB / cfg.Password 做可选的初始化（miniredis 默认可选 AUTH）。
// 该函数仅在 cfg.Addr 为空（mock 模式）时被调用。
//
// 参数:
//   - cfg: Redis 配置（主要用 DB / Password 两个字段）
//
// 返回:
//   - *miniredis.Miniredis: 已启动的内存 Redis 服务句柄
//   - error: 启动失败时返回错误
func startMiniRedis(cfg *conf.RedisCfg) (*miniredis.Miniredis, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("启动 miniredis 失败: %w", err)
	}
	// miniredis 原生不强制 AUTH；有密码要求时按需设置（便于本地测试鉴权逻辑）
	if cfg.Password != "" {
		mr.RequireAuth(cfg.Password)
	}
	// miniredis 默认 DB 号默认可选 SELECT，此处不做额外 DB 切换（行为与真实 Redis 一致）
	_ = cfg.DB
	return mr, nil
}
