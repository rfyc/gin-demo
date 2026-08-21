// redis_test.go 是 NewRedisCli / Close 的单元测试：
// 覆盖真实连接配置校验、不可达地址报错、Close 行为，
// 并通过 miniredis mock 验证客户端命令的可用性。
package cache_test

import (
	"context"
	"gin-demo/src/core/cache"
	"gin-demo/src/core/conf"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRedisCliEmptyAddr 覆盖异常场景：Addr 为空时必须报错，
// 引导先用 MockSrvRun 启动 mock 再建连。
func TestNewRedisCliEmptyAddr(t *testing.T) {

	// case: Addr 为空, 应返回错误
	cfg := &conf.RedisCfg{Addr: ""}
	c, err := cache.NewRedisCli(cfg)
	require.Error(t, err)
	if c != nil {
		cache.Close(c)
	}
}

// TestNewRedisCliUnreachable 覆盖异常场景：连接不可达地址时应返回 PING 失败错误。
func TestNewRedisCliUnreachable(t *testing.T) {

	// case: 连接不可达地址, 应返回 PING 失败错误
	cfg := &conf.RedisCfg{Addr: "127.0.0.1:1", PoolSize: 10}
	c, err := cache.NewRedisCli(cfg)
	require.Error(t, err)
	if c != nil {
		cache.Close(c)
	}
}

// TestCloseNil 覆盖异常场景：Close(nil) 不应 panic，应返回 nil。
func TestCloseNil(t *testing.T) {

	// case: Close(nil) 不应 panic, 应返回 nil
	require.NoError(t, cache.Close(nil))
}

// TestCloseTwice 覆盖异常场景：Close 重复调用不应 panic
// （go-redis 第二次调用返回 "redis: client is closed"，属预期行为）。
func TestCloseTwice(t *testing.T) {

	// case: Close 重复调用不应 panic
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	mini, err := cache.MockSrvRun(cfg)
	require.NoError(t, err)
	defer mini.Close()

	mockCfg := *cfg
	mockCfg.Addr = mini.Addr()
	c, err := cache.NewRedisCli(&mockCfg)
	require.NoError(t, err)
	require.NoError(t, cache.Close(c))
	_ = cache.Close(c)
}

// TestRedisCliMock 覆盖基本场景：通过 miniredis mock 验证
// PING / SET / GET / HSET / HGETALL / EXPIRE / TTL 命令可用。
func TestRedisCliMock(t *testing.T) {

	// case: mock 服务下 PING/SET/GET/HSET/EXPIRE 等命令可用
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	require.NoError(t, c.Ping(ctx).Err())

	// case: SET/GET 读写一致
	require.NoError(t, c.Set(ctx, "k", "v", 0).Err())
	got, err := c.Get(ctx, "k").Result()
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	// case: HSET/HGETALL 读写一致
	require.NoError(t, c.HSet(ctx, "h", "name", "alice").Err())
	all, err := c.HGetAll(ctx, "h").Result()
	require.NoError(t, err)
	assert.Equal(t, "alice", all["name"])

	// case: EXPIRE/TTL 语义可用
	require.NoError(t, c.Expire(ctx, "k", 60*time.Second).Err())
	ttl, err := c.TTL(ctx, "k").Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
}

// TestRedisCliMiss 覆盖异常场景：GET 不存在的 key 应返回 redis.Nil。
func TestRedisCliMiss(t *testing.T) {

	// case: GET 不存在的 key, 应返回 redis.Nil
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	_, err := c.Get(ctx, "cache:miss:key").Result()
	assert.ErrorIs(t, err, redis.Nil)
}
