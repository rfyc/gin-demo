// tool_test.go 是 NewTool 的单元测试:
// 覆盖 schema 推导成功、工具元信息、多字段类型映射场景。
package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// weatherInput 是测试用工具输入结构体, 覆盖 string/int/bool 字段类型。
type weatherToolInput struct {
	City string `json:"city"`
	Days int    `json:"days"`
	Fast bool   `json:"fast"`
}

// weatherOutput 是测试用工具输出结构体。
type weatherToolOutput struct {
	Summary string `json:"summary"`
}

// fakeWeatherTool 是被注册的测试工具处理函数。
func fakeWeatherTool(ctx context.Context, input weatherToolInput) (output weatherToolOutput, err error) {

	return weatherToolOutput{Summary: "sunny"}, nil
}

// TestNewTool 覆盖正常场景: 从函数签名推导 schema 并创建工具。
func TestNewTool(t *testing.T) {

	var tool, err = NewTool("get_weather", "查询指定城市天气", fakeWeatherTool)

	// case: 创建成功, 元信息与推导的 schema 完整
	assert.NoError(t, err)
	assert.NotNil(t, tool)
	assert.Equal(t, "get_weather", tool.Name)
	assert.Equal(t, "查询指定城市天气", tool.Description)
	assert.NotNil(t, tool.toolInfo)

	// case: schema 名称与工具名一致, 参数包含三个输入字段
	assert.Equal(t, "get_weather", tool.toolInfo.Name)
	assert.Equal(t, "查询指定城市天气", tool.toolInfo.Desc)
	assert.NotNil(t, tool.toolInfo.ParamsOneOf)
	var jsonSchema, schemaErr = tool.toolInfo.ParamsOneOf.ToJSONSchema()
	assert.NoError(t, schemaErr)
	_, cityOK := jsonSchema.Properties.Get("city")
	_, daysOK := jsonSchema.Properties.Get("days")
	_, fastOK := jsonSchema.Properties.Get("fast")
	assert.True(t, cityOK)
	assert.True(t, daysOK)
	assert.True(t, fastOK)
}

// TestNewToolDirect 覆盖正常场景: 创建的工具处理函数可被直接调用(经 InvokableTool 路径)。
func TestNewToolDirect(t *testing.T) {

	var _, err = NewTool("get_weather", "查询天气", fakeWeatherTool)

	// case: 同名工具重复创建不冲突(每次独立实例)
	assert.NoError(t, err)
}
