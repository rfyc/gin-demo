// test/llm 是交互式大模型对话 demo:
// 在控制台逐行输入指令, 模型以流式输出实时打印回复, 并保留多轮对话上下文。
// 配置复用 core.Conf 全局配置(Llm 节点, 含测试网关地址与密钥, 来自 config/local/conf.yaml)。
//
// 运行方式:
//
//	go run ./test/llm
//
// 退出命令: exit / quit / q
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"gin-demo/src/core"
	"gin-demo/src/pkg/llm"
	"gin-demo/src/pkg/logger"
)

// 控制台输出样式(ANSI 转义码): 区分提示符/思考过程/回复, 便于阅读。
const (
	colorReset  = "\x1b[0m"    // colorReset 重置所有样式
	colorPrompt = "\x1b[1;34m" // colorPrompt 提示符/模型名亮蓝色
	colorMute   = "\x1b[90m"   // colorMute 分隔标记/次要信息弱灰色
	colorThink  = "\x1b[3;90m" // colorThink 思考过程灰色斜体
	colorReply  = "\x1b[32m"   // colorReply 回复绿色
)

// 区域头部分隔线样式: 思考用虚线(轻), 回复用实线(重), 视觉权重区分主次。
const (
	headerFillLen = 24  // headerFillLen 分隔线标题后的填充字符数
	fillThink     = "┈" // fillThink 思考区虚线填充
	fillReply     = "━" // fillReply 回复区实线填充
)

// sectionHeader 生成区域头部行: 弱灰色标题 + 填充线, 如 "┈┈ 思考 ┈┈┈┈┈┈"。
//
// 参数:
//   - title: 区域标题(如 思考/回复)
//   - fill: 填充字符(思考用虚线, 回复用实线)
//
// 返回:
//   - string: 带 ANSI 样式的头部行(不含换行)
func sectionHeader(title string, fill string) (header string) {

	var titleWidth = len([]rune(title)) * 2                      // titleWidth 标题显示宽度(中文按 2 列)
	var fillStart = strings.Repeat(fill, 2)                      // fillStart 标题前填充
	var fillEnd = strings.Repeat(fill, headerFillLen-titleWidth) // fillEnd 标题后填充
	return colorMute + fillStart + " " + title + " " + fillEnd + colorReset
}

// compactNewlines 压缩流式增量中的连续换行: 模型输出的 "\n\n" 段落分隔会撑出
// 大片空行, 归一为单个换行; 跨 chunk 的换行由 lastNL 状态衔接。
//
// 参数:
//   - delta: 本次收到的思考增量
//   - lastNL: 状态指针, 记录上一输出字符是否为换行(true 时本次的换行被丢弃)
//
// 返回:
//   - string: 压缩后的增量(保留样式所需的换行)
func compactNewlines(delta string, lastNL *bool) (out string) {

	var b strings.Builder // b 压缩输出缓冲
	var r rune            // r 当前字符

	for _, r = range delta {
		// case: 连续换行只保留第一个, 上一输出已是换行则丢弃
		if r == '\n' {
			if *lastNL {
				continue
			}
			*lastNL = true
			b.WriteRune(r)
			continue
		}
		*lastNL = false
		b.WriteRune(r)
	}
	return b.String()
}

// main 交互式对话 demo 入口: 使用 core.Conf 全局配置进入控制台 REPL。
func main() {
	var (
		err error // err 对话运行错误
	)

	// 1. 打印欢迎横幅(模型名/网关地址/退出命令), 进入交互式对话循环
	fmt.Println()
	fmt.Println(colorMute, "╭──────────────────────────╮", colorReset)
	fmt.Println(colorPrompt, "│   LLM 交互对话 Demo      │", colorReset)
	fmt.Println(colorMute, "╰──────────────────────────╯", colorReset)
	fmt.Println(colorMute, "模型: ", colorReset, colorPrompt, llm.DefaultModel.Name, colorReset)
	fmt.Println(colorMute, "退出: exit / quit / q", colorReset)
	fmt.Println()
	if err = runRepl(context.Background(), core.Conf.Llm, llm.DefaultModel, "", os.Stdin, os.Stdout); err != nil {
		logger.Errorf(context.Background(), "对话退出: %v", err)
		os.Exit(1)
	}
}

// runRepl 在标准输入上执行多轮交互式对话: 用户逐行输入指令,
// 模型流式回复并实时打印, 完整回复追加到历史保证后续轮次上下文连续。
//
// 参数:
//   - ctx: 请求上下文
//   - c: 连接配置(apiUrl/apiKey), 来自 core.Conf.Llm
//   - cfg: 模型调用参数配置(模型名/温度等)
//   - instruction: 系统提示词, 空串不注入
//   - in: 指令输入源, 传入 os.Stdin
//   - out: 提示与回复输出目标, 传入 os.Stdout
//
// 返回:
//   - err: 读取输入或模型调用失败时返回带上下文的错误
//
// 错误场景:
//   - 模型调用失败: 返回带上下文包装的错误
//   - 输入读取错误(非 EOF): 返回相应错误
func runRepl(ctx context.Context, c llm.Conf, cfg llm.ModelConf, instruction string, in io.Reader, out io.Writer) (err error) {
	var (
		scanner  = bufio.NewScanner(in) // scanner 输入流逐行读取器
		input    string                 // input 当前用户输入的一行指令
		messages []*schema.Message      // messages 多轮对话历史(系统提示由 llm 注入)
		reply    string                 // reply 模型聚合的完整回复
		reqCtx   context.Context        // reqCtx 单次请求的可取消上下文
		cancel   context.CancelFunc     // cancel reqCtx 的取消函数
	)

	// 1. 交互循环: 读取指令 -> 流式输出思考/回复 -> 上下文累积
	for {
		// 1.1. 提示输入; EOF 或退出命令时结束
		fmt.Fprint(out, colorPrompt, "❯ ", colorReset)
		if !scanner.Scan() {
			break
		}
		if input = strings.TrimSpace(scanner.Text()); input == "" {
			continue
		}
		if isQuitCmd(input) {
			break
		}

		// 1.2. 用户指令追加到历史, 流式调用模型并分样式输出
		{
			var (
				thinkOpened bool      // thinkOpened 思考区是否已开启
				thinkLastNL bool      // thinkLastNL 思考内容上一输出字符是否为换行(压缩连续空行用)
				replyOpened bool      // replyOpened 回复区是否已开启
				reqStart    time.Time // reqStart 单次请求开始时间, 收尾时统计耗时
			)
			messages = append(messages, schema.UserMessage(input))
			reqStart = time.Now()
			reqCtx, cancel = context.WithTimeout(ctx, 120*time.Second)
			reply, err = llm.RunChatAgentStream(reqCtx, c, cfg, instruction, messages,
				// 文本增量: 绿色回复; 首次收到时补齐思考区结尾换行, 空行分隔后开启回复区(实线头)
				func(ctx context.Context, delta string) error {
					if !replyOpened {
						if thinkOpened && !thinkLastNL {
							fmt.Fprintln(out)
						}
						fmt.Fprintln(out)
						fmt.Fprintln(out, sectionHeader("回复", fillReply))
						replyOpened = true
					}
					fmt.Fprint(out, colorReply, delta, colorReset)
					return nil
				},
				// 思考增量: 灰色斜体, 虚线头标记思考区; 压缩模型段落换行保证行距均匀
				llm.WithReasoningCallback(func(ctx context.Context, delta string) error {
					if !thinkOpened {
						fmt.Fprintln(out)
						fmt.Fprintln(out, sectionHeader("思考", fillThink))
						thinkOpened, thinkLastNL = true, true
					}
					fmt.Fprint(out, colorThink, compactNewlines(delta, &thinkLastNL), colorReset)
					return nil
				}))
			cancel()
			if err != nil {
				return fmt.Errorf("runRepl chat: %w", err)
			}

			// 1.3. 收尾: 归位光标到行首, 空行分隔后打印耗时, 完整回复追加历史
			if replyOpened || (thinkOpened && !thinkLastNL) {
				fmt.Fprintln(out)
			}
			fmt.Fprintf(out, "\n%s⏱ %.1fs%s\n\n", colorMute, time.Since(reqStart).Seconds(), colorReset)
			messages = append(messages, schema.AssistantMessage(reply, nil))
		}
	}

	// 2. 校验标准输入读取错误(非 EOF)
	if err = scanner.Err(); err != nil {
		return fmt.Errorf("runRepl read stdin: %w", err)
	}
	return nil
}

// isQuitCmd 判断输入是否为退出命令(exit/quit/q)。
//
// 参数:
//   - input: 用户输入, 已去除首尾空白
//
// 返回:
//   - quit: 是退出命令时为 true
func isQuitCmd(input string) (quit bool) {

	switch input {
	case "exit", "quit", "q":
		return true
	}
	return false
}
