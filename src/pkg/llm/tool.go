package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// ToolFunc 是工具处理函数的统一签名: 输入结构体 T 由模型生成的 JSON 参数解码而来,
// 输出 D 会被序列化为 JSON 回传给模型。T/D 均需为可 JSON 序列化的结构体。
type ToolFunc[T, D any] func(ctx context.Context, input T) (output D, err error)

// Tool 是注册给 Agent 的单个工具: 模型按名称选择调用, 参数按 JSON schema 生成,
// 执行由内部的 eino InvokableTool 完成。
type Tool struct {
	Name        string             // Name 工具名称, 模型据此选择工具, 需全局唯一
	Description string             // Description 工具用途描述, 供模型理解何时调用
	toolInfo    *schema.ToolInfo   // toolInfo 由 InferTool 推导的参数 schema
	invokable   tool.InvokableTool // invokable 承载实际执行逻辑的 eino 工具实例
}

// NewTool 从函数签名自动推导参数 JSON schema, 创建一个可注册给 Agent 的工具。
//
// 输入类型 T 的字段与 json tag 决定 schema; 输出 D 序列化为 JSON 回传模型。
//
// 泛型参数:
//   - T: 工具输入结构体(非指针), 字段 json tag 需完整
//   - D: 工具输出类型, 可 JSON 序列化
//
// 参数:
//   - name: 工具名称, 模型调用的标识
//   - desc: 工具用途描述, 供模型决策是否调用
//   - fn: 工具处理函数
//
// 返回:
//   - t: 已就绪的工具实例
//   - err: schema 推导失败时返回带工具名的错误
//
// 错误场景:
//   - T 无法推导 JSON schema(含不支持的字段类型): 返回推导失败错误
func NewTool[T, D any](name, desc string, fn ToolFunc[T, D]) (t *Tool, err error) {

	var (
		invokable tool.InvokableTool // invokable 承载实际执行逻辑的 eino 工具实例
		toolInfo  *schema.ToolInfo   // toolInfo 推导出的参数 JSON schema
	)

	// 1. 从函数签名推导参数 schema 并创建 eino 工具实例
	if invokable, err = utils.InferTool(name, desc, utils.InvokeFunc[T, D](fn)); err != nil {
		return nil, fmt.Errorf("llm.NewTool InferTool: %w - tool: %s", err, name)
	}

	// 2. 取回推导出的 ToolInfo(InferTool 返回的实例内部持有)
	if toolInfo, err = invokable.Info(context.Background()); err != nil {
		return nil, fmt.Errorf("llm.NewTool tool Info: %w - tool: %s", err, name)
	}

	return &Tool{
		Name:        name,
		Description: desc,
		toolInfo:    toolInfo,
		invokable:   invokable,
	}, nil
}
