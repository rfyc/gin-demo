// Package cache 提供 Redis 客户端构造，支持真实 Redis 与本地 miniredis mock 两种模式。
// go-redis 内部自带连接池（redis.Options 的 PoolSize/MinIdleConns 等），
// 因此本包直接返回 *redis.Client，业务代码调用原生命令即可，无需额外封装。
package cache

import (
	"context"
	"fmt"
	"gin-demo/src/core/conf"
	"gin-demo/src/utils"
	"time"

	"github.com/go-redis/redis/v8"
)

// NewRedisCli 创建连接真实 Redis 的 go-redis 客户端。
// cfg.Addr 必须非空，否则返回错误（本地 mock 请使用 MockSrvRun 启动服务后
// 将地址写入 cfg 再调用本函数）。
// 创建时执行一次 PING 校验连通性，失败返回包含地址的错误。
//   - 连接池参数映射自 cfg（PoolSize/IdleTimeout/MinIdleConns 等）；
//   - 校验失败时关闭客户端。
func NewRedisCli(cfg *conf.RedisCfg) (client *redis.Client, err error) {

	// 涉及第三方请求(连接 Redis), 必须记录日志: 耗时, error, 入参出参
	defer func(startTime time.Time) {
		utils.LogErrorInfo(context.Background(), startTime, "NewRedisCli", err, "addr", cfg.Addr)
	}(time.Now())

	var (
		opts = &redis.Options{ // go-redis 连接池配置, 参数映射自 cfg
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.PoolSize / 5,
			IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}
		ctx    context.Context    // PING 连通性校验的带超时上下文
		cancel context.CancelFunc // 上下文取消函数
	)

	// 1. 基于配置创建客户端
	client = redis.NewClient(opts)

	// 2. 执行一次 PING 校验连通性, 失败时关闭客户端并返回带地址的错误
	{
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = client.Ping(ctx).Err(); err != nil {
			_ = client.Close() // 关闭失败不影响返回的 PING 错误
			return nil, fmt.Errorf("Redis PING 失败 [addr=%s]: %w", cfg.Addr, err)
		}
	}

	return client, nil
}

// Close 关闭 go-redis 客户端（真实模式）。
// mock 模式下 miniredis 由启动方负责关闭（见 MockSrvRun）；重复调用可能返回错误。
func Close(client *redis.Client) error {

	var err error
	if client != nil {
		err = client.Close()
	}
	return err
}
