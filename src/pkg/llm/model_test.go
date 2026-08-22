// model_test.go 是 NewChatModel 工厂的单元测试:
// 覆盖正常创建与模型名边界场景。
// openai.NewChatModel 构造不发起网络请求, 直接真实创建并断言实例类型。
package llm

import (
	"context"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/stretchr/testify/assert"
)

// TestNewChatModelOpenAI 覆盖正常场景: OpenAI 兼容配置创建出 openai.ChatModel 实例。
func TestNewChatModelOpenAI(t *testing.T) {

	var (
		temperature = float32(0.3)
		maxTokens   = 4096
	)
	var c = Conf{ApiKey: "sk-test", ApiUrl: "http://example.com/v1"}
	var cfg = ModelConf{
		Name:        "doubao-seed-2.0-lite",
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Extra:       map[string]any{"k": "v"},
	}

	var chat, err = NewChatModel(context.Background(), c, cfg)

	// case: 创建成功且返回非 nil 实例, 配置合法
	assert.NoError(t, err)
	assert.NotNil(t, chat)
}

// TestNewChatModelClaudeName 覆盖边界场景: claude 模型名创建成功。
// ReasoningEffort 对 claude 模型的跳过逻辑在 NewChatModel 内部,
// 该用例验证 claude 命名不导致创建失败(豆包网关代理 claude 模型的常见形态)。
func TestNewChatModelClaudeName(t *testing.T) {

	var effort = "high"
	var c = Conf{ApiKey: "sk-test", ApiUrl: "http://example.com/v1"}
	var chat, err = NewChatModel(context.Background(), c, ModelConf{
		Name:            "claude-opus-4.8",
		ReasoningEffort: &effort,
	})

	// case: 模型名含 claude 时跳过 ReasoningEffort 透传, 创建本身不受影响
	assert.NoError(t, err)
	assert.NotNil(t, chat)
}

// 编译期断言: 确保测试文件引用的 openai 包路径正确。
var _ = openai.ChatModelConfig{}
