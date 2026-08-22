package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"gin-demo/src/utils"
)

// ChatOption 是 Agent 运行的可选配置, 函数式选项模式。
type ChatOption func(*chatOptions)

// chatOptions 是 Agent 运行的可配置项集合。
type chatOptions struct {
	tools         []*Tool                 // tools 本轮对话可用的工具集
	maxIterations int                     // maxIterations 模型生成循环上限, 0 用默认值
	reasoningCb   StreamReasoningCallback // reasoningCb 思考过程增量回调, nil 表示不透出
}

// WithTools 为对话追加可用工具, 可多次调用累积。
//
// 参数:
//   - tools: 工具实例列表
//
// 返回:
//   - ChatOption: 选项函数
func WithTools(tools ...*Tool) ChatOption {

	return func(o *chatOptions) {
		o.tools = append(o.tools, tools...)
	}
}

// WithMaxIterations 设置模型生成循环上限(含工具调用轮次), 防止无限循环。
//
// 参数:
//   - n: 循环上限, <=0 时使用默认值 20
//
// 返回:
//   - ChatOption: 选项函数
func WithMaxIterations(n int) ChatOption {

	return func(o *chatOptions) {
		o.maxIterations = n
	}
}

// StreamCallback 是流式输出的回调函数签名。
// 每收到一段文本增量(delta)调用一次; 回调返回 error 时中止流式输出。
type StreamCallback func(ctx context.Context, delta string) error

// StreamReasoningCallback 是流式思考过程增量(delta)的回调函数签名。
// 每收到一段思考增量调用一次; 回调返回 error 时中止流式输出。
// 与 StreamCallback 的区别: 思考过程不聚合进最终回复, 仅用于展示。
type StreamReasoningCallback func(ctx context.Context, delta string) error

// WithReasoningCallback 设置思考过程增量回调, 透出模型的思考过程(仅流式生效)。
//
// 参数:
//   - cb: 思考增量回调, 每次收到一段思考增量时调用
//
// 返回:
//   - ChatOption: 选项函数
func WithReasoningCallback(cb StreamReasoningCallback) ChatOption {

	return func(o *chatOptions) {
		o.reasoningCb = cb
	}
}

// newChatModelFn 是模型工厂的可替换注入点, 非 nil 时优先于 NewChatModel 使用。
// 仅测试注入 fake 模型时赋值, 生产路径恒为 nil。
var newChatModelFn func(ctx context.Context, c Conf, cfg ModelConf) (model.ToolCallingChatModel, error)

// RunChatAgent 执行一轮带可选工具调用的多轮对话(非流式), 返回最终完整回复。
//
// 无工具时模型直答; 带工具时按 ReAct 循环执行: 模型选择工具 -> 执行 -> 结果回传模型,
// 直到模型给出最终文本回复。
//
// 参数:
//   - ctx: 请求上下文, 透传取消信号与 trace_id
//   - c: 连接配置, 提供 apiUrl/apiKey
//   - cfg: 模型配置
//   - instruction: 系统提示词, 为空时不注入 system 消息
//   - messages: 对话历史(含本轮用户消息)
//   - opts: 可选配置(工具/循环上限)
//
// 返回:
//   - reply: 模型最终回复文本
//   - err: 模型或工具执行失败、回复为空时返回带上下文的错误
//
// 错误场景:
//   - 模型实例创建失败: 返回工厂错误
//   - Agent 运行出错: 返回运行错误
//   - 无任何输出: 返回输出为空错误
func RunChatAgent(ctx context.Context, c Conf, cfg ModelConf, instruction string, messages []*schema.Message, opts ...ChatOption) (reply string, err error) {

	var (
		startTime = time.Now()    // startTime 请求开始时间, 用于统计耗时
		agent     adk.Agent       // agent 就绪的对话 Agent 实例
		finalMsg  *schema.Message // finalMsg 模型最终输出消息
	)

	// 1. 出口日志(第三方请求函数必须记录: 耗时/error/入参出参), err 有无自动分级
	defer func(bTime time.Time) {
		utils.LogErrorInfo(ctx, bTime, "RunChatAgent", err,
			"model", cfg.Name,
			"instructionLen", len(instruction),
			"messageCount", len(messages),
			"toolCount", countTools(opts),
			"replyLen", len(reply))
	}(startTime)

	// 2. 构建 agent 并阻塞运行至结束
	{
		if agent, err = buildChatModelAgent(ctx, c, cfg, instruction, opts...); err != nil {
			return "", err
		}
		if finalMsg, err = runAgentBlocking(ctx, agent, messages); err != nil {
			return "", fmt.Errorf("llm.RunChatAgent run: %w - model: %s", err, cfg.Name)
		}
	}

	// 3. 校验回复非空后返回
	if finalMsg.Content == "" {
		return "", fmt.Errorf("llm.RunChatAgent reply empty - model: %s", cfg.Name)
	}
	return finalMsg.Content, nil
}

// RunChatAgentStream 执行流式对话: 文本增量逐段回调, 返回最终完整回复。
// 工具调用轮次不透出(工具结果由模型内部消化), 仅透出模型生成的文本增量。
//
// 参数:
//   - ctx: 请求上下文, 透传取消信号与 trace_id
//   - c: 连接配置, 提供 apiUrl/apiKey
//   - cfg: 模型配置
//   - instruction: 系统提示词, 为空时不注入 system 消息
//   - messages: 对话历史(含本轮用户消息)
//   - callback: 文本增量回调, 返回 error 时中止
//   - opts: 可选配置(工具/循环上限)
//
// 返回:
//   - reply: 流结束后聚合的完整回复文本
//   - err: 模型或工具执行失败、回调失败时返回带上下文的错误
//
// 错误场景:
//   - 模型实例创建失败: 返回工厂错误
//   - 流读取或回调失败: 返回相应错误
func RunChatAgentStream(ctx context.Context, c Conf, cfg ModelConf, instruction string, messages []*schema.Message, callback StreamCallback, opts ...ChatOption) (reply string, err error) {

	var (
		startTime   = time.Now()            // startTime 请求开始时间, 用于统计耗时
		agent       adk.Agent               // agent 就绪的对话 Agent 实例
		runCtx      context.Context         // runCtx 派生的可取消上下文, 提前返回时止损后台循环
		cancelRun   context.CancelFunc      // cancelRun runCtx 的取消函数
		replyBuf    []byte                  // replyBuf 聚合的完整回复缓冲
		event       *adk.AgentEvent         // event 当前消费的流事件
		ok          bool                    // ok 迭代器是否还有事件
		reasoningCb StreamReasoningCallback // reasoningCb 思考增量回调, 从 opts 提取, nil 表示不透出
	)

	// 1. 出口日志(第三方请求函数必须记录: 耗时/error/入参出参), err 有无自动分级
	defer func(bTime time.Time) {
		utils.LogErrorInfo(ctx, bTime, "RunChatAgentStream", err,
			"model", cfg.Name,
			"messageCount", len(messages),
			"toolCount", countTools(opts),
			"replyLen", len(reply))
	}(startTime)

	// 2. 参数校验: 回调为空直接报错, 不发起模型调用
	if callback == nil {
		return "", fmt.Errorf("llm.RunChatAgentStream callback is nil - model: %s", cfg.Name)
	}

	// 3. 构建 agent; 派生可取消 ctx: 提前返回(错误/回调失败)时 cancel 后台
	//    flowAgent 循环, 避免其继续调用模型产生费用; 正常结束路径在循环耗尽后 cancel 无副作用
	{
		if agent, err = buildChatModelAgent(ctx, c, cfg, instruction, opts...); err != nil {
			return "", err
		}
		runCtx, cancelRun = context.WithCancel(ctx)
		defer cancelRun()
	}

	// 4. 消费事件流: 仅透出模型生成的文本增量(assistant 流), 工具结果不透出
	{
		var (
			runner   = adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true}) // runner 流式 agent 运行器
			iterator = runner.Run(runCtx, messages)                                              // iterator 事件迭代器
			mv       *adk.MessageVariant                                                         // mv 当前事件的消息载体
			o        chatOptions                                                                 // o 应用选项后的配置集合
		)
		for _, opt := range opts {
			opt(&o)
		}
		reasoningCb = o.reasoningCb
		for {
			if event, ok = iterator.Next(); !ok {
				break
			}
			if event.Err != nil {
				cancelRun()
				return "", fmt.Errorf("llm.RunChatAgentStream run: %w - model: %s", event.Err, cfg.Name)
			}
			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}
			mv = event.Output.MessageOutput
			// case: 流式形态, 逐 chunk 回调; 回调失败时 cancel 止损
			if mv.IsStreaming && mv.MessageStream != nil {
				if replyBuf, err = consumeMessageStream(runCtx, mv.MessageStream, callback, reasoningCb, replyBuf); err != nil {
					cancelRun()
					return "", fmt.Errorf("llm.RunChatAgentStream consume: %w - model: %s", err, cfg.Name)
				}
				continue
			}
			// case: 非流式形态的完整消息(如工具调用后最终一轮), 直接聚合
			if mv.Message != nil && mv.Message.Role == schema.Assistant && mv.Message.Content != "" {
				replyBuf = append(replyBuf, mv.Message.Content...)
			}
		}
	}

	// 5. 校验聚合回复非空后返回
	reply = string(replyBuf)
	if reply == "" {
		return "", fmt.Errorf("llm.RunChatAgentStream reply empty - model: %s", cfg.Name)
	}
	return reply, nil
}

// RunToolParseAgent 执行强制工具提交的结构化解析: 在提示词中强制模型调用指定工具,
// 工具入参即为结构化输出, 由 toolHandler 接收并处理。
//
// 与 RunChatAgent 的区别: 模型不产出自由文本, 必须以 tool call 的 JSON 参数
// 形式提交结果; 适合"把自然语言解析为结构体"的场景。
//
// 泛型参数:
//   - T: 工具输入结构体(即期望的结构化解析结果类型)
//   - D: 工具输出类型(处理结果回传模型, 通常为简单确认信息)
//
// 参数:
//   - ctx: 请求上下文, 透传取消信号与 trace_id
//   - c: 连接配置, 提供 apiUrl/apiKey
//   - cfg: 模型配置
//   - prompt: 任务提示词(描述如何解析)
//   - toolName: 工具名称(模型提交结果的标识)
//   - toolDesc: 工具用途描述
//   - toolHandler: 工具处理函数, 入参为模型提交的结构化结果
//   - messages: 对话历史(含待解析的用户输入)
//
// 返回:
//   - err: 模型未按要求调用工具、运行失败、提示词为空时返回带上下文的错误
//
// 错误场景:
//   - prompt 为空: 返回提示词缺失错误
//   - 运行失败或无输出: 返回运行错误
func RunToolParseAgent[T, D any](ctx context.Context, c Conf, cfg ModelConf, prompt string, toolName string, toolDesc string, toolHandler ToolFunc[T, D], messages ...*schema.Message) (err error) {

	var (
		startTime  = time.Now() // startTime 请求开始时间, 用于统计耗时
		fullPrompt string       // fullPrompt 追加强制工具约束后的完整提示词
		parseTool  *Tool        // parseTool 强制提交结果的结构化解析工具
		agent      adk.Agent    // agent 就绪的解析 Agent 实例
	)

	// 1. 出口日志(第三方请求函数必须记录: 耗时/error/入参出参), err 有无自动分级
	defer func(bTime time.Time) {
		utils.LogErrorInfo(ctx, bTime, "RunToolParseAgent", err,
			"model", cfg.Name,
			"toolName", toolName,
			"promptLen", len(prompt))
	}(startTime)

	// 2. 校验提示词, 并在尾部强制注入"必须调用工具提交结果"的约束
	if prompt == "" {
		return fmt.Errorf("llm.RunToolParseAgent prompt is empty - toolName: %s - model: %s", toolName, cfg.Name)
	}
	fullPrompt = fmt.Sprintf("%s\n\n---\n\n# 输出方式（强制）\n\n完成上述任务后，**必须调用工具 `%s`** 提交结果。\n禁止以纯文字段落形式输出结果——调用工具是本任务唯一有效的完成方式，不得跳过。\n\n", prompt, toolName)

	// 3. 创建强制提交工具并构建 agent, 工具设为 ReturnDirectly(调用后直接结束循环)
	{
		if parseTool, err = NewTool(toolName, toolDesc, toolHandler); err != nil {
			return fmt.Errorf("llm.RunToolParseAgent NewTool: %w - toolName: %s", err, toolName)
		}
		if agent, err = buildAgentWithTools(ctx, c, cfg, fullPrompt, []*Tool{parseTool}, map[string]bool{toolName: true}, 0); err != nil {
			return fmt.Errorf("llm.RunToolParseAgent build: %w - toolName: %s", err, toolName)
		}
	}

	// 4. 运行至结束: 模型调用工具 -> toolHandler 执行 -> ReturnDirectly 结束
	if _, err = runAgentBlocking(ctx, agent, messages); err != nil {
		return fmt.Errorf("llm.RunToolParseAgent run: %w - toolName: %s - model: %s", err, toolName, cfg.Name)
	}
	return nil
}

// buildChatModelAgent 按默认配置构建对话 Agent(非流式/流式共用)。
//
// 参数:
//   - ctx: 请求上下文
//   - c: 连接配置, 提供 apiUrl/apiKey
//   - cfg: 模型配置
//   - instruction: 系统提示词
//   - opts: 可选配置
//
// 返回:
//   - agent: 就绪的 Agent 实例
//   - err: 模型创建或 Agent 构建失败时返回
func buildChatModelAgent(ctx context.Context, c Conf, cfg ModelConf, instruction string, opts ...ChatOption) (agent adk.Agent, err error) {

	var (
		o        chatOptions     // o 应用选项后的配置集合
		directly map[string]bool // directly 无强制直接返回工具(全部走完整循环)
	)

	for _, opt := range opts {
		opt(&o)
	}
	return buildAgentWithTools(ctx, c, cfg, instruction, o.tools, directly, o.maxIterations)
}

// buildAgentWithTools 构建带工具集的 ChatModelAgent。
//
// 参数:
//   - ctx: 请求上下文
//   - c: 连接配置, 提供 apiUrl/apiKey
//   - cfg: 模型配置
//   - instruction: 系统提示词
//   - tools: 工具集
//   - returnDirectly: 工具名到"调用后直接返回"的映射, nil 表示全部走完整循环
//   - maxIterations: 循环上限, <=0 时用默认值
//
// 返回:
//   - agent: 就绪的 Agent 实例
//   - err: 模型创建或 Agent 构建失败时返回
func buildAgentWithTools(ctx context.Context, c Conf, cfg ModelConf, instruction string, tools []*Tool, returnDirectly map[string]bool, maxIterations int) (agent adk.Agent, err error) {

	var (
		chat     model.ToolCallingChatModel   // chat 底层聊天模型实例(模型工厂创建)
		cfgAgent = &adk.ChatModelAgentConfig{ // cfgAgent eino agent 配置
			Name:        "llm-chat-agent",
			Description: "gin-demo llm chat agent",
			Instruction: instruction,
		}
	)

	// 1. 创建模型实例(测试通过 newChatModelFn 注入 fake)
	{
		if newChatModelFn != nil {
			if chat, err = newChatModelFn(ctx, c, cfg); err != nil {
				return nil, fmt.Errorf("llm NewChatModel: %w", err)
			}
		} else if chat, err = NewChatModel(ctx, c, cfg); err != nil {
			return nil, fmt.Errorf("llm NewChatModel: %w", err)
		}
		cfgAgent.Model = chat
	}

	// 2. 组装 agent 配置: 工具集 + 直接返回映射 + 循环上限
	{
		for _, t := range tools {
			cfgAgent.ToolsConfig.ToolsNodeConfig.Tools = append(cfgAgent.ToolsConfig.ToolsNodeConfig.Tools, t.invokable)
		}
		cfgAgent.ToolsConfig.ReturnDirectly = returnDirectly
		if maxIterations > 0 {
			cfgAgent.MaxIterations = maxIterations
		}
	}

	// 3. 构建 agent 实例
	if agent, err = adk.NewChatModelAgent(ctx, cfgAgent); err != nil {
		return nil, fmt.Errorf("llm adk.NewChatModelAgent: %w", err)
	}
	return agent, nil
}

// runAgentBlocking 运行 Agent 并阻塞至结束, 返回最终输出消息。
//
// 参数:
//   - ctx: 请求上下文
//   - agent: 就绪的 Agent 实例
//   - messages: 输入消息
//
// 返回:
//   - finalMsg: 最终输出消息
//   - err: 运行出错或无输出时返回
func runAgentBlocking(ctx context.Context, agent adk.Agent, messages []*schema.Message) (finalMsg *schema.Message, err error) {

	var (
		runner   = adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}) // runner 非流式 agent 运行器
		iterator = runner.Run(ctx, messages)                          // iterator 事件迭代器
		event    *adk.AgentEvent                                      // event 当前消费的事件
		ok       bool                                                 // ok 迭代器是否还有事件
	)
	for {
		if event, ok = iterator.Next(); !ok {
			break
		}
		if event.Err != nil {
			return nil, fmt.Errorf("agent run: %w", event.Err)
		}
		// case: 取最后一个 Message 事件作为最终输出(工具 ToolMessage 必在其前)
		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.Message != nil {
			finalMsg = event.Output.MessageOutput.Message
		}
	}
	if finalMsg == nil {
		return nil, fmt.Errorf("agent output empty")
	}
	return finalMsg, nil
}

// consumeMessageStream 消费流式消息: 文本增量回调并聚合完整回复, 思考增量走独立回调。
//
// 参数:
//   - ctx: 请求上下文
//   - stream: 流式消息读取器
//   - callback: 文本增量回调
//   - reasoningCallback: 思考增量回调, nil 时忽略思考内容
//   - replyBuf: 已聚合的回复缓冲
//
// 返回:
//   - replyBuf: 追加本流内容后的回复缓冲
//   - err: 读取或回调失败时返回
func consumeMessageStream(ctx context.Context, stream *schema.StreamReader[*schema.Message], callback StreamCallback, reasoningCallback StreamReasoningCallback, replyBuf []byte) (out []byte, err error) {

	var msg *schema.Message // msg 当前收到的流式消息 chunk

	defer stream.Close()
	for {
		if msg, err = stream.Recv(); err != nil {
			// case: 流正常结束(EOF), 返回已聚合内容
			if errors.Is(err, io.EOF) {
				return replyBuf, nil
			}
			return nil, err
		}
		if msg == nil {
			continue
		}
		// case: 思考增量 chunk, 走独立回调(不聚合进回复, 仅用于展示)
		if msg.ReasoningContent != "" && reasoningCallback != nil {
			if err = reasoningCallback(ctx, msg.ReasoningContent); err != nil {
				return nil, fmt.Errorf("stream reasoning callback: %w", err)
			}
		}
		// case: 文本增量 chunk, 回调并聚合
		if msg.Content != "" {
			if err = callback(ctx, msg.Content); err != nil {
				return nil, fmt.Errorf("stream callback: %w", err)
			}
			replyBuf = append(replyBuf, msg.Content...)
		}
	}
}

// countTools 统计选项中的工具数量, 用于日志。
//
// 参数:
//   - opts: 可选配置列表
//
// 返回:
//   - int: 工具总数
func countTools(opts []ChatOption) (count int) {

	var o chatOptions // o 应用选项后的配置集合

	for _, opt := range opts {
		opt(&o)
	}
	return len(o.tools)
}
