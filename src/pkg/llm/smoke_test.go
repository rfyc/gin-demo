// smoke_test.go 是真实网关的冒烟测试: 验证非流式对话与流式对话的完整链路。
// 依赖测试环境网关可达, 默认跳过, 设环境变量 LLM_SMOKE=1 时启用。
package llm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

// smokeCfg 从环境变量读取网关配置(LLM_SMOKE_API_KEY/LLM_SMOKE_URL 必填,
// LLM_SMOKE_MODEL 默认 doubao-seed-2.0-lite), 密钥不入源码。
func smokeCfg() (c Conf, cfg ModelConf, ok bool) {

	c = Conf{}
	cfg = ModelConf{Name: "doubao-seed-2.0-lite"}
	if key := os.Getenv("LLM_SMOKE_API_KEY"); key != "" {
		c.ApiKey = key
	}
	if url := os.Getenv("LLM_SMOKE_URL"); url != "" {
		c.ApiUrl = url
	}
	if model := os.Getenv("LLM_SMOKE_MODEL"); model != "" {
		cfg.Name = model
	}
	return c, cfg, c.ApiKey != ""
}

// smokeEnabled 判断是否启用冒烟测试。
func smokeEnabled() bool {

	return os.Getenv("LLM_SMOKE") == "1"
}

// TestSmokeChat 非流式真实网关冒烟: 简单问答返回非空回复。
func TestSmokeChat(t *testing.T) {

	var c, cfg, ok = smokeCfg()
	if !smokeEnabled() || !ok {
		t.Skip("设 LLM_SMOKE=1 与 LLM_SMOKE_API_KEY 启用真实网关冒烟")
	}

	var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var reply, err = RunChatAgent(ctx, c, cfg, "",
		[]*schema.Message{schema.UserMessage("用一句话介绍你自己")})

	assert.NoError(t, err)
	assert.NotEmpty(t, reply)
	t.Logf("非流式回复: %s", reply)
}

// TestSmokeChatStream 流式真实网关冒烟: 增量逐段回调且聚合完整。
func TestSmokeChatStream(t *testing.T) {

	var c, cfg, ok = smokeCfg()
	if !smokeEnabled() || !ok {
		t.Skip("设 LLM_SMOKE=1 与 LLM_SMOKE_API_KEY 启用真实网关冒烟")
	}

	var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var deltas []string
	var reply, err = RunChatAgentStream(ctx, c, cfg, "",
		[]*schema.Message{schema.UserMessage("用一句话介绍北京")},
		func(ctx context.Context, delta string) error {
			deltas = append(deltas, delta)
			return nil
		})

	assert.NoError(t, err)
	assert.NotEmpty(t, reply)
	assert.NotEmpty(t, deltas)
	t.Logf("流式聚合回复: %s - 增量段数: %d", reply, len(deltas))
}
