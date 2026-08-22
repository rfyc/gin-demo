// Package llm 提供通用的大模型对话能力:
// 模型配置、OpenAI 兼容 ChatModel 工厂、带工具调用的对话 Agent(流式/非流式)
// 以及强制工具提交的结构化解析, 业务代码通过本包统一接入大模型。
package llm

import (
	"github.com/alibabacloud-go/tea/tea"
)

// Conf 是大模型连接配置, 与配置文件 Llm 节点对应, 提供 apiUrl/apiKey。
type Conf struct {
	ApiKey string `mapstructure:"apiKey"` // ApiKey 认证密钥
	ApiUrl string `mapstructure:"apiUrl"` // ApiUrl OpenAI 兼容网关地址
}

// ModelConf 是单个大模型的调用参数配置。
// 值类型语义: 持有指针字段(Temperature/MaxTokens/ReasoningEffort),
// 跨请求共享前应通过 Clone 拷贝, 避免并发修改指针指向的值。
// 厂商固定为 OpenAI 兼容协议, 连接信息(apiUrl/apiKey)由 Conf 提供。
type ModelConf struct {
	Name            string         // Name 模型真实名称(接口调用使用的 model 参数)
	Temperature     *float32       // Temperature 采样温度, nil 表示用模型默认值
	MaxTokens       *int           // MaxTokens 单次补全的最大 token 数, nil 表示不限制
	ReasoningEffort *string        // ReasoningEffort 推理力度(high/medium/low), 仅部分模型支持, nil 表示不启用
	Extra           map[string]any // Extra 额外请求参数, 透传给厂商接口(如 response_format)
}

// DefaultModel 默认模型调用配置, 供调用方作为兜底使用, 可覆盖。
var DefaultModel = ModelConf{
	Name:        "doubao-seed-2.0-lite",
	Temperature: tea.Float32(0.3),
	MaxTokens:   tea.Int(65535),
	Extra: map[string]any{
		"reasoning": map[string]string{
			"mode":   "enabled",
			"effort": "low", //low,medium,high,max
		},
	},
}

// Clone 返回配置的深拷贝, 拷贝指针字段与 map, 避免副本与原配置共享可变状态。
//
// 返回:
//   - ModelConf: 与原配置等值的独立副本
func (a *ModelConf) Clone() (cfg ModelConf) {

	var (
		temperature         float32 // temperature 拷贝出的采样温度值
		maxCompletionTokens int     // maxCompletionTokens 拷贝出的补全上限值
		reasoningEffort     string  // reasoningEffort 拷贝出的推理力度值
	)

	// 1. 拷贝标量字段
	cfg = ModelConf{
		Name: a.Name,
	}

	// 2. 深拷贝指针字段: 逐一取值后取地址, 不与原配置共享底层
	if a.Temperature != nil {
		temperature = *a.Temperature
		cfg.Temperature = &temperature
	}
	if a.MaxTokens != nil {
		maxCompletionTokens = *a.MaxTokens
		cfg.MaxTokens = &maxCompletionTokens
	}
	if a.ReasoningEffort != nil {
		reasoningEffort = *a.ReasoningEffort
		cfg.ReasoningEffort = &reasoningEffort
	}

	// 3. 深拷贝 Extra map
	if len(a.Extra) > 0 {
		cfg.Extra = make(map[string]any, len(a.Extra))
		for k, v := range a.Extra {
			cfg.Extra[k] = v
		}
	}
	return
}
