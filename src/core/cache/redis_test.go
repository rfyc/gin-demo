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
)

// TestNewRedisCliEmptyAddr 覆盖异常场景：Addr 为空时必须报错，
// 引导先用 MockSrvRun 启动 mock 再建连。
func TestNewRedisCliEmptyAddr(t *testing.T) {
	cfg := &conf.RedisCfg{Addr: ""}
	if c, err := cache.NewRedisCli(cfg); err == nil {
		cache.Close(c)
		t.Fatal("NewRedisCli 空 Addr 期望报错，实际成功")
	}
}

// TestNewRedisCliUnreachable 覆盖异常场景：连接不可达地址时应返回 PING 失败错误。
func TestNewRedisCliUnreachable(t *testing.T) {
	cfg := &conf.RedisCfg{Addr: "127.0.0.1:1", PoolSize: 10}
	if c, err := cache.NewRedisCli(cfg); err == nil {
		cache.Close(c)
		t.Fatal("NewRedisCli 不可达地址期望报错，实际成功")
	}
}

// TestCloseNil 覆盖异常场景：Close(nil) 不应 panic，应返回 nil。
func TestCloseNil(t *testing.T) {
	if err := cache.Close(nil); err != nil {
		t.Fatalf("Close(nil) 期望无错误，实际: %v", err)
	}
}

// TestCloseTwice 覆盖异常场景：Close 重复调用不应 panic
// （go-redis 第二次调用返回 "redis: client is closed"，属预期行为）。
func TestCloseTwice(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	mini, err := cache.MockSrvRun(cfg)
	if err != nil {
		t.Fatalf("cache.MockSrvRun FAIL: %v", err)
	}
	defer mini.Close()

	mockCfg := *cfg
	mockCfg.Addr = mini.Addr()
	c, err := cache.NewRedisCli(&mockCfg)
	if err != nil {
		t.Fatalf("cache.NewRedisCli FAIL: %v", err)
	}
	if err := cache.Close(c); err != nil {
		t.Fatalf("First Close FAIL: %v", err)
	}
	_ = cache.Close(c)
}

// TestRedisCliMock 覆盖基本场景：通过 miniredis mock 验证
// PING / SET / GET / HSET / HGETALL / EXPIRE / TTL 命令可用。
func TestRedisCliMock(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	if err := c.Ping(ctx).Err(); err != nil {
		t.Fatalf("mock Redis PING FAIL: %v", err)
	}

	// SET / GET
	if err := c.Set(ctx, "k", "v", 0).Err(); err != nil {
		t.Fatalf("mock Redis SET FAIL: %v", err)
	}
	if got, err := c.Get(ctx, "k").Result(); err != nil || got != "v" {
		t.Fatalf("mock Redis GET want=v got=%s err=%v", got, err)
	}

	// HSET / HGETALL
	if err := c.HSet(ctx, "h", "name", "alice").Err(); err != nil {
		t.Fatalf("mock Redis HSET FAIL: %v", err)
	}
	if all, err := c.HGetAll(ctx, "h").Result(); err != nil || all["name"] != "alice" {
		t.Fatalf("mock Redis HGETALL want=alice got=%v err=%v", all, err)
	}

	// EXPIRE / TTL
	if err := c.Expire(ctx, "k", 60*time.Second).Err(); err != nil {
		t.Fatalf("mock Redis EXPIRE FAIL: %v", err)
	}
	if ttl, err := c.TTL(ctx, "k").Result(); err != nil || ttl <= 0 {
		t.Fatalf("mock Redis TTL 应大于 0，got=%v err=%v", ttl, err)
	}
}

// TestRedisCliMiss 覆盖异常场景：GET 不存在的 key 应返回 redis.Nil。
func TestRedisCliMiss(t *testing.T) {
	cfg := &conf.RedisCfg{IdleTimeout: 60, PoolSize: 10}
	c := newMockSrvClient(t, cfg)
	ctx := context.Background()

	if _, err := c.Get(ctx, "cache:miss:key").Result(); err != redis.Nil {
		t.Fatalf("GET 不存在的 key 应返回 redis.Nil，实际: %v", err)
	}
}
