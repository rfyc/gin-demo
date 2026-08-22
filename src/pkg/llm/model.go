package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// NewChatModel 创建 OpenAI 兼容协议的 ChatModel 实例(覆盖 OpenAI/豆包等兼容网关)。
//
// 参数:
//   - ctx: 请求上下文, 透传给模型实例初始化
//   - c: 连接配置, 提供 apiUrl/apiKey
//   - cfg: 模型调用参数配置(Name/温度等)
//
// 返回:
//   - chat: 支持工具调用的聊天模型实例, 流式与非流式共用
//   - err: 实例初始化失败时返回带模型名与地址的错误
//
// 错误场景:
//   - 实例创建失败: 返回带模型名与网关地址的错误
func NewChatModel(ctx context.Context, c Conf, cfg ModelConf) (chat model.ToolCallingChatModel, err error) {

	var conf = &openai.ChatModelConfig{ // conf OpenAI 兼容网关的实例配置
		Model:   cfg.Name,
		APIKey:  c.ApiKey,
		BaseURL: c.ApiUrl,
	}

	// 1. 采样温度与补全上限: 未设置时保持 nil, 使用模型默认行为
	if cfg.Temperature != nil {
		conf.Temperature = cfg.Temperature
	}
	if cfg.MaxTokens != nil {
		conf.MaxCompletionTokens = cfg.MaxTokens
	}

	// 2. 推理力度: 仅部分模型支持(如 gpt 系列), 模型名不含 claude 时透传
	if cfg.ReasoningEffort != nil && !strings.Contains(strings.ToLower(cfg.Name), "claude") {
		conf.ReasoningEffort = openai.ReasoningEffortLevel(*cfg.ReasoningEffort)
	}

	// 3. 额外请求参数: 整体透传给网关(如 response_format/json 模式)
	if len(cfg.Extra) > 0 {
		conf.ExtraFields = cfg.Extra
	}

	// 4. 创建实例
	if chat, err = openai.NewChatModel(ctx, conf); err != nil {
		return nil, fmt.Errorf("llm.NewChatModel openai.NewChatModel: %w - model: %s - apiUrl: %s", err, conf.Model, conf.BaseURL)
	}
	return chat, nil
}
