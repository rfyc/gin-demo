// mock_srv_test.go 是 MockSrvRun 的单元测试：
// 覆盖 miniredis 服务启动、密码认证、命令兼容性与关闭行为，
// 验证 go-redis 客户端与 mock 服务的联通性。
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

// newMockSrvClient 启动 miniredis mock 服务并返回指向它的 go-redis 客户端，
// 测试结束时统一关闭客户端与服务两侧。
func newMockSrvClient(t *testing.T, cfg *conf.RedisCfg) *redis.Client {

	t.Helper()
	mini, err := cache.MockSrvRun(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {

		mini.Close()
	})

	mockCfg := *cfg
	mockCfg.Addr = mini.Addr()
	c, err := cache.NewRedisCli(&mockCfg)
	require.NoError(t, err)
	t.Cleanup(func() {

		cache.Close(c)
	})
	return c
}

// TestMockSrvRun 覆盖基本场景：无密码启动成功，返回的客户端可正常 PING。
func TestMockSrvRun(t *testing.T) {

	// case: 无密码启动 mock 服务, 返回的客户端可正常 PING
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	require.NoError(t, c.Ping(context.Background()).Err())
}

// TestMockSrvRunWithPassword 覆盖密码场景：设置 RequireAuth 后，
// 带正确密码的客户端可正常 PING，不带密码的客户端认证失败。
func TestMockSrvRunWithPassword(t *testing.T) {

	// case: 设置 RequireAuth 后, 带正确密码可 PING, 无密码认证失败
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10, Password: "secret"}
	c := newMockSrvClient(t, cfg)
	require.NoError(t, c.Ping(context.Background()).Err())

	// case: 无密码客户端应认证失败
	mini, err := cache.MockSrvRun(cfg)
	require.NoError(t, err)
	defer mini.Close()
	noPass := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer noPass.Close()
	require.Error(t, noPass.Ping(context.Background()).Err())
}

// TestMockSrvRunSetGet 验证 mock 服务的 SET / GET 行为与真实 Redis 一致：
// 写入的值能读回，未设置的 key 返回 redis.Nil。
func TestMockSrvRunSetGet(t *testing.T) {

	// case: mock 服务 SET/GET 与真实 Redis 一致, 未设置 key 返回 redis.Nil
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:foo"
	val := "bar-123"
	require.NoError(t, c.Set(ctx, key, val, 0).Err())
	got, err := c.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, val, got)

	// case: GET 未设置过的 key 应返回 redis.Nil
	_, err = c.Get(ctx, "cache:miss:key").Result()
	assert.ErrorIs(t, err, redis.Nil)
}

// TestMockSrvRunHash 验证 mock 服务的 HSET / HGET / HGETALL 命令可用，字段读写一致。
func TestMockSrvRunHash(t *testing.T) {

	// case: mock 服务的 HSET/HGET/HGETALL 命令可用且读写一致
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:user:1"
	require.NoError(t, c.HSet(ctx, key, "name", "alice", "age", "18").Err())
	name, err := c.HGet(ctx, key, "name").Result()
	require.NoError(t, err)
	assert.Equal(t, "alice", name)

	all, err := c.HGetAll(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "18", all["age"])
}

// TestMockSrvRunExpire 验证 mock 服务的 EXPIRE/TTL 命令语义：
// 设置过期时间后，TTL 返回大于 0 的剩余时间。
func TestMockSrvRunExpire(t *testing.T) {

	// case: mock 服务 EXPIRE 后 TTL 返回大于 0 的剩余时间
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:expire"
	require.NoError(t, c.Set(ctx, key, "x", 0).Err())
	require.NoError(t, c.Expire(ctx, key, 60*time.Second).Err())
	ttl, err := c.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
}

// TestMockSrvRunSetTTL 验证带 TTL 的 SET：设置 5s 过期后，TTL 应在 (0, 5s] 范围内。
func TestMockSrvRunSetTTL(t *testing.T) {

	// case: 带 TTL 的 SET 后, TTL 应在 (0, 5s] 范围内
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:ttl-key"
	require.NoError(t, c.Set(ctx, key, "v", 5*time.Second).Err())
	ttl, err := c.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 5*time.Second)
}

// TestMockSrvRunClose 覆盖异常场景：关闭 mock 服务后，新客户端无法连接。
func TestMockSrvRunClose(t *testing.T) {

	// case: 关闭 mock 服务后, 新客户端无法连接
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	mini, err := cache.MockSrvRun(cfg)
	require.NoError(t, err)
	addr := mini.Addr()
	mini.Close()

	// case: 服务已停止, 新连接应失败
	client := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second})
	defer client.Close()
	require.Error(t, client.Ping(context.Background()).Err())
}
