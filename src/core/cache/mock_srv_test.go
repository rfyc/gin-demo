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
)

// newMockSrvClient 启动 miniredis mock 服务并返回指向它的 go-redis 客户端，
// 测试结束时统一关闭客户端与服务两侧。
func newMockSrvClient(t *testing.T, cfg *conf.RedisCfg) *redis.Client {
	t.Helper()
	mini, err := cache.MockSrvRun(cfg)
	if err != nil {
		t.Fatalf("cache.MockSrvRun FAIL: %v", err)
	}
	t.Cleanup(func() {
		mini.Close()
	})

	mockCfg := *cfg
	mockCfg.Addr = mini.Addr()
	c, err := cache.NewRedisCli(&mockCfg)
	if err != nil {
		t.Fatalf("cache.NewRedisCli(mock) FAIL: %v", err)
	}
	t.Cleanup(func() {
		cache.Close(c)
	})
	return c
}

// TestMockSrvRun 覆盖基本场景：无密码启动成功，返回的客户端可正常 PING。
func TestMockSrvRun(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	if err := c.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("mock Redis PING FAIL: %v", err)
	}
}

// TestMockSrvRunWithPassword 覆盖密码场景：设置 RequireAuth 后，
// 带正确密码的客户端可正常 PING，不带密码的客户端认证失败。
func TestMockSrvRunWithPassword(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10, Password: "secret"}
	c := newMockSrvClient(t, cfg)
	if err := c.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("带密码客户端 PING FAIL: %v", err)
	}

	// 不带密码的客户端应认证失败
	mini, err := cache.MockSrvRun(cfg)
	if err != nil {
		t.Fatalf("cache.MockSrvRun FAIL: %v", err)
	}
	defer mini.Close()
	noPass := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer noPass.Close()
	if err := noPass.Ping(context.Background()).Err(); err == nil {
		t.Fatal("无密码客户端应认证失败，实际成功")
	}
}

// TestMockSrvRunSetGet 验证 mock 服务的 SET / GET 行为与真实 Redis 一致：
// 写入的值能读回，未设置的 key 返回 redis.Nil。
func TestMockSrvRunSetGet(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:foo"
	val := "bar-123"
	if err := c.Set(ctx, key, val, 0).Err(); err != nil {
		t.Fatalf("mock Redis SET FAIL: %v", err)
	}
	got, err := c.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("mock Redis GET FAIL: %v", err)
	}
	if got != val {
		t.Fatalf("mock Redis GET 值不一致: want=%s got=%s", val, got)
	}

	// 未设置过的 key 应返回 redis.Nil
	if _, err := c.Get(ctx, "cache:miss:key").Result(); err != redis.Nil {
		t.Fatalf("GET 不存在的 key 应返回 redis.Nil，实际: %v", err)
	}
}

// TestMockSrvRunHash 验证 mock 服务的 HSET / HGET / HGETALL 命令可用，字段读写一致。
func TestMockSrvRunHash(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:user:1"
	if err := c.HSet(ctx, key, "name", "alice", "age", "18").Err(); err != nil {
		t.Fatalf("mock Redis HSET FAIL: %v", err)
	}
	name, err := c.HGet(ctx, key, "name").Result()
	if err != nil {
		t.Fatalf("mock Redis HGET FAIL: %v", err)
	}
	if name != "alice" {
		t.Fatalf("HGET name want=alice got=%s", name)
	}

	all, err := c.HGetAll(ctx, key).Result()
	if err != nil {
		t.Fatalf("mock Redis HGETALL FAIL: %v", err)
	}
	if all["age"] != "18" {
		t.Fatalf("HGETALL age want=18 got=%s", all["age"])
	}
}

// TestMockSrvRunExpire 验证 mock 服务的 EXPIRE/TTL 命令语义：
// 设置过期时间后，TTL 返回大于 0 的剩余时间。
func TestMockSrvRunExpire(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:expire"
	if err := c.Set(ctx, key, "x", 0).Err(); err != nil {
		t.Fatalf("mock Redis SET FAIL: %v", err)
	}
	if err := c.Expire(ctx, key, 60*time.Second).Err(); err != nil {
		t.Fatalf("mock Redis EXPIRE FAIL: %v", err)
	}
	ttl, err := c.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("mock Redis TTL FAIL: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("TTL 应大于 0，实际为 %v", ttl)
	}
}

// TestMockSrvRunSetTTL 验证带 TTL 的 SET：设置 5s 过期后，TTL 应在 (0, 5s] 范围内。
func TestMockSrvRunSetTTL(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	key := "cache:mock:ttl-key"
	if err := c.Set(ctx, key, "v", 5*time.Second).Err(); err != nil {
		t.Fatalf("mock Redis SETEX FAIL: %v", err)
	}
	ttl, err := c.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("mock Redis TTL FAIL: %v", err)
	}
	if ttl <= 0 || ttl > 5*time.Second {
		t.Fatalf("TTL 应在 (0, 5s] 内，实际为 %v", ttl)
	}
}

// TestMockSrvRunClose 覆盖异常场景：关闭 mock 服务后，新客户端无法连接。
func TestMockSrvRunClose(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	mini, err := cache.MockSrvRun(cfg)
	if err != nil {
		t.Fatalf("cache.MockSrvRun FAIL: %v", err)
	}
	addr := mini.Addr()
	mini.Close()

	// 服务已停止，新连接应失败
	client := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err == nil {
		t.Fatal("mock 服务关闭后应连接失败，实际成功")
	}
}
