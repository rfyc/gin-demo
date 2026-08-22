// agent_test.go 是 Agent 运行层的单元测试:
// 用 fake 模型实现 model.ToolCallingChatModel 接口注入, 覆盖
// 纯对话、工具调用循环、流式回调、强制工具解析、错误场景。
package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

// fakeChatModel 是注入测试的假模型: 按预设脚本依次返回消息。
// 支持流式与非流式两种调用形态。
type fakeChatModel struct {
	generateReplies []*schema.Message   // generateReplies 非流式依次返回的消息脚本
	streamChunks    [][]*schema.Message // streamChunks 每次流式调用返回的 chunk 组
	generateCalls   int                 // generateCalls Generate 被调用的次数
	streamCalls     int                 // streamCalls Stream 被调用的次数
	generateErr     error               // generateErr Generate 固定返回的错误
	streamErr       error               // streamErr Stream 固定返回的错误
}

// Generate 非流式生成: 按脚本顺序返回消息。
func (f *fakeChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {

	f.generateCalls++
	if f.generateErr != nil {
		return nil, f.generateErr
	}
	if f.generateCalls > len(f.generateReplies) {
		return schema.AssistantMessage("done", nil), nil
	}
	return f.generateReplies[f.generateCalls-1], nil
}

// Stream 流式生成: 按脚本顺序返回 chunk 流。
func (f *fakeChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {

	f.streamCalls++
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	var idx = f.streamCalls - 1
	if idx >= len(f.streamChunks) {
		idx = len(f.streamChunks) - 1
	}
	return schema.StreamReaderFromArray(f.streamChunks[idx]), nil
}

// WithTools 实现 ToolCallingChatModel: 返回自身(假模型忽略工具绑定)。
func (f *fakeChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {

	return f, nil
}

// weatherInput 是测试工具的输入结构体。
type weatherInput struct {
	City string `json:"city"`
}

// weatherOutput 是测试工具的输出结构体。
type weatherOutput struct {
	Summary string `json:"summary"`
}

// fakeWeatherHandler 是测试工具处理函数: 记录收到的城市并返回固定结果。
func fakeWeatherHandler(ctx context.Context, input weatherInput) (output weatherOutput, err error) {

	return weatherOutput{Summary: "sunny " + input.City}, nil
}

// TestRunChatAgentPlain 覆盖正常场景: 无工具时模型直答, 返回最终文本。
func TestRunChatAgentPlain(t *testing.T) {

	var fake = &fakeChatModel{
		generateReplies: []*schema.Message{schema.AssistantMessage("你好，我是助手", nil)},
	}
	stubNewChatModel(t, fake)

	var reply, err = RunChatAgent(context.Background(),
		llmConfForTest(), ModelConf{Name: "fake-model"},
		"你是助手",
		[]*schema.Message{schema.UserMessage("你好")})

	// case: 纯对话直答, 返回模型文本
	assert.NoError(t, err)
	assert.Equal(t, "你好，我是助手", reply)
	assert.Equal(t, 1, fake.generateCalls)
}

// TestRunChatAgentToolLoop 覆盖正常场景: 模型先调工具再用结果作答。
func TestRunChatAgentToolLoop(t *testing.T) {

	var tool, err = NewTool("get_weather", "查询天气", fakeWeatherHandler)
	assert.NoError(t, err)

	var fake = &fakeChatModel{
		generateReplies: []*schema.Message{
			// 第一轮: 模型选择调用工具
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID: "call-1",
					Function: schema.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"city":"北京"}`,
					},
				}},
			},
			// 第二轮: 用工具结果作答
			schema.AssistantMessage("北京今天晴", nil),
		},
	}
	stubNewChatModel(t, fake)

	var reply, runErr = RunChatAgent(context.Background(),
		llmConfForTest(), ModelConf{Name: "fake-model"},
		"",
		[]*schema.Message{schema.UserMessage("北京天气")},
		WithTools(tool))

	// case: 工具被真实执行(参数正确解码), 最终返回第二轮文本
	assert.NoError(t, runErr)
	assert.Equal(t, "北京今天晴", reply)
	assert.Equal(t, 2, fake.generateCalls)
}

// TestRunChatAgentModelErr 覆盖异常场景: 模型调用失败返回错误。
func TestRunChatAgentModelErr(t *testing.T) {

	stubNewChatModel(t, &fakeChatModel{generateErr: errors.New("boom")})

	var reply, err = RunChatAgent(context.Background(),
		llmConfForTest(), ModelConf{Name: "fake-model"},
		"",
		[]*schema.Message{schema.UserMessage("hi")})

	// case: 模型错误向上传播, 带模型名上下文
	assert.Error(t, err)
	assert.Empty(t, reply)
}

// TestRunChatAgentStreamPlain 覆盖正常场景: 流式回调逐段收到增量, 聚合完整回复。
func TestRunChatAgentStreamPlain(t *testing.T) {

	var fake = &fakeChatModel{
		streamChunks: [][]*schema.Message{{
			schema.AssistantMessage("你", nil),
			schema.AssistantMessage("好", nil),
			schema.AssistantMessage("呀", nil),
		}},
	}
	stubNewChatModel(t, fake)

	var deltas []string
	var reply, err = RunChatAgentStream(context.Background(),
		llmConfForTest(), ModelConf{Name: "fake-model"},
		"",
		[]*schema.Message{schema.UserMessage("你好")},
		func(ctx context.Context, delta string) error {
			deltas = append(deltas, delta)
			return nil
		})

	// case: 每个 chunk 回调一次, 聚合出完整回复
	assert.NoError(t, err)
	assert.Equal(t, []string{"你", "好", "呀"}, deltas)
	assert.Equal(t, "你好呀", reply)
}

// TestRunChatAgentStreamCallbackErr 覆盖异常场景: 回调返回错误时中止并传播。
func TestRunChatAgentStreamCallbackErr(t *testing.T) {

	var fake = &fakeChatModel{
		streamChunks: [][]*schema.Message{{
			schema.AssistantMessage("你", nil),
			schema.AssistantMessage("好", nil),
		}},
	}
	stubNewChatModel(t, fake)

	var cbErr = errors.New("client closed")
	var reply, err = RunChatAgentStream(context.Background(),
		llmConfForTest(), LlmMessageConfigForTest(),
		"",
		[]*schema.Message{schema.UserMessage("你好")},
		func(ctx context.Context, delta string) error {
			return cbErr
		})

	// case: 回调错误中止流式输出并向上传播
	assert.ErrorIs(t, err, cbErr)
	assert.Empty(t, reply)
}

// TestRunChatAgentStreamNilCallback 覆盖异常场景: 回调为 nil 时直接报错。
func TestRunChatAgentStreamNilCallback(t *testing.T) {

	var reply, err = RunChatAgentStream(context.Background(),
		llmConfForTest(), LlmMessageConfigForTest(), "",
		[]*schema.Message{schema.UserMessage("hi")}, nil)

	// case: nil 回调快速失败, 不发起模型调用
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "callback is nil")
	assert.Empty(t, reply)
}

// TestRunChatAgentStreamReasoning 覆盖正常场景: 思考增量走独立回调, 不聚合进回复。
func TestRunChatAgentStreamReasoning(t *testing.T) {

	var fake = &fakeChatModel{
		streamChunks: [][]*schema.Message{{
			{Role: schema.Assistant, ReasoningContent: "思考一"},
			{Role: schema.Assistant, ReasoningContent: "思考二"},
			schema.AssistantMessage("最终答案", nil),
		}},
	}
	stubNewChatModel(t, fake)

	var (
		reasonings []string // reasonings 思考增量回调收到的内容
		deltas     []string // deltas 文本增量回调收到的内容
	)
	var reply, err = RunChatAgentStream(context.Background(),
		llmConfForTest(), LlmMessageConfigForTest(), "",
		[]*schema.Message{schema.UserMessage("你好")},
		func(ctx context.Context, delta string) error {
			deltas = append(deltas, delta)
			return nil
		},
		WithReasoningCallback(func(ctx context.Context, delta string) error {
			reasonings = append(reasonings, delta)
			return nil
		}))

	// case: 思考增量独立回调, 回复只聚合文本增量
	assert.NoError(t, err)
	assert.Equal(t, []string{"思考一", "思考二"}, reasonings)
	assert.Equal(t, []string{"最终答案"}, deltas)
	assert.Equal(t, "最终答案", reply)
}

// TestRunToolParseAgentOK 覆盖正常场景: 模型按要求调用工具提交结构化结果。
func TestRunToolParseAgentOK(t *testing.T) {

	var gotCity string
	var handler = func(ctx context.Context, input weatherInput) (output weatherOutput, err error) {
		gotCity = input.City
		return weatherOutput{Summary: "ok"}, nil
	}

	var fake = &fakeChatModel{
		generateReplies: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID: "call-1",
					Function: schema.FunctionCall{
						Name:      "submit_result",
						Arguments: `{"city":"上海"}`,
					},
				}},
			},
		},
	}
	stubNewChatModel(t, fake)

	var err = RunToolParseAgent(context.Background(),
		llmConfForTest(), LlmMessageConfigForTest(),
		"解析用户提到的城市",
		"submit_result", "提交解析结果", handler,
		schema.UserMessage("我想查上海的天气"))

	// case: 工具被调用且参数解码正确, agent 正常结束
	assert.NoError(t, err)
	assert.Equal(t, "上海", gotCity)
}

// TestRunToolParseAgentEmptyPrompt 覆盖异常场景: 提示词为空报错。
func TestRunToolParseAgentEmptyPrompt(t *testing.T) {

	var err = RunToolParseAgent(context.Background(),
		llmConfForTest(), LlmMessageConfigForTest(), "",
		"submit_result", "提交解析结果", fakeWeatherHandler,
		schema.UserMessage("hi"))

	// case: 空 prompt 快速失败
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is empty")
}

// LlmMessageConfigForTest 返回测试用模型配置(配合 stubNewChatModel 使用)。
func LlmMessageConfigForTest() ModelConf {

	return ModelConf{Name: "fake-model"}
}

// llmConfForTest 返回测试用连接配置(配合 stubNewChatModel 使用, 不发起真实请求)。
func llmConfForTest() Conf {

	return Conf{ApiKey: "test-key", ApiUrl: "http://fake"}
}

// stubNewChatModel 将包内 NewChatModel 替换为返回 fake 模型。
// 由于 NewChatModel 是包内函数, 测试通过临时变量注入实现打桩。
func stubNewChatModel(t *testing.T, fake model.ToolCallingChatModel) {

	t.Helper()
	newChatModelFn = func(ctx context.Context, c Conf, cfg ModelConf) (model.ToolCallingChatModel, error) {
		return fake, nil
	}
	t.Cleanup(func() {
		newChatModelFn = nil
	})
}

// compile-time check

// TestRunChatAgentEmptyReply 覆盖边界场景: 模型返回空内容时报错而非静默空回复。
func TestRunChatAgentEmptyReply(t *testing.T) {

	stubNewChatModel(t, &fakeChatModel{
		generateReplies: []*schema.Message{schema.AssistantMessage("", nil)},
	})

	var reply, err = RunChatAgent(context.Background(),
		llmConfForTest(), LlmMessageConfigForTest(), "",
		[]*schema.Message{schema.UserMessage("hi")})

	// case: 空回复与流式版一致报 reply empty 错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reply empty")
	assert.Empty(t, reply)
}
