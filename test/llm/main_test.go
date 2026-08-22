package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"gin-demo/src/pkg/llm"
)

// TestIsQuitCmd 表驱动验证退出命令识别(exit/quit/q 为退出, 其余非退出)。
func TestIsQuitCmd(t *testing.T) {
	var tests = []struct {
		name  string // name 用例名称
		input string // input 待判断的指令
		want  bool   // want 期望的退出判定
	}{
		{"exit", "exit", true},
		{"quit", "quit", true},
		{"q", "q", true},
		{"普通指令", "你好", false},
		{"大写不识别", "EXIT", false},
		{"空串", "", false},
		{"带前导空格不识别", " exit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isQuitCmd(tt.input))
		})
	}
}

// TestRunReplQuitNoModel 验证空行跳过与退出命令结束对话, 不触发真实模型调用。
func TestRunReplQuitNoModel(t *testing.T) {
	var (
		input = "\n   \nexit\n" // input 空行/空白行/退出命令
		out   = &bytes.Buffer{} // out 提示符输出缓冲
	)

	err := runRepl(context.Background(), llm.Conf{}, llm.ModelConf{}, "", strings.NewReader(input), out)

	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(out.String(), "❯"))
}

// TestRunReplEOF 验证输入流直接 EOF 时正常结束, 不触发模型调用。
func TestRunReplEOF(t *testing.T) {
	var (
		out = &bytes.Buffer{} // out 提示符输出缓冲
	)

	err := runRepl(context.Background(), llm.Conf{}, llm.ModelConf{}, "", strings.NewReader(""), out)

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "❯")
}
