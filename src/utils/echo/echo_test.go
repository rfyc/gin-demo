package echo

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout 捕获 f 执行期间写入标准输出的内容并返回。
func captureStdout(f func()) string {
	var old *os.File
	var r *os.File
	var w *os.File
	var err error

	old = os.Stdout
	r, w, err = os.Pipe()
	if err != nil {
		fmt.Println(err)
		return ""
	}
	os.Stdout = w
	f()
	// 关闭写端并恢复标准输出, 防止 f 内部 panic 导致管道泄漏
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	r.Close()
	return string(out)
}

// TestJson 测试 Json 函数的基本打印与 nil 入参场景。
func TestJson(t *testing.T) {
	// 基本场景: 打印结构体的 JSON 内容
	out := captureStdout(func() {
		Json(map[string]int{"a": 1})
	})
	if !strings.Contains(out, `"a": 1`) {
		t.Errorf("Json 输出缺少预期内容, 实际输出: %s", out)
	}

	// 异常场景: nil 入参不得 panic, 且打印 "nil"
	out = captureStdout(func() {
		Json(nil)
	})
	if !strings.Contains(out, "nil") {
		t.Errorf("Json(nil) 输出缺少 nil, 实际输出: %s", out)
	}
}

// TestYml 测试 Yml 函数的基本打印、xxx_ 过滤与 nil 入参场景。
func TestYml(t *testing.T) {
	// 基本场景: 打印 map 的 YAML 内容
	out := captureStdout(func() {
		Yml(map[string]string{"a": "b"})
	})
	if !strings.Contains(out, "a: b") {
		t.Errorf("Yml 输出缺少预期内容, 实际输出: %s", out)
	}

	// 过滤场景: 含 xxx_ 的键对应的行应被过滤
	out = captureStdout(func() {
		Yml(map[string]string{"xxx_pwd": "secret", "name": "ok"})
	})
	if strings.Contains(out, "xxx_pwd") {
		t.Errorf("Yml 未过滤含 xxx_ 的行, 实际输出: %s", out)
	}
	if !strings.Contains(out, "name: ok") {
		t.Errorf("Yml 过滤掉了正常行, 实际输出: %s", out)
	}

	// 异常场景: nil 入参不得 panic, 且打印 "nil"
	out = captureStdout(func() {
		Yml(nil)
	})
	if !strings.Contains(out, "nil") {
		t.Errorf("Yml(nil) 输出缺少 nil, 实际输出: %s", out)
	}
}

// TestDump 测试 Dump 函数的 nil 入参场景。
func TestDump(t *testing.T) {
	// 异常场景: nil 入参不得 panic
	out := captureStdout(func() {
		Dump(nil)
	})
	if !strings.Contains(out, "nil") {
		t.Errorf("Dump(nil) 输出缺少 nil, 实际输出: %s", out)
	}
}

// TestContext 测试 Context 函数的 nil、emptyCtx、valueCtx、自定义结构体场景。
func TestContext(t *testing.T) {
	// 异常场景: nil 入参不得 panic, 且打印提示
	out := captureStdout(func() {
		Context(nil)
	})
	if !strings.Contains(out, "Context is nil") {
		t.Errorf("Context(nil) 输出缺少提示, 实际输出: %s", out)
	}

	// 异常场景: context.Background(底层为 int 的 emptyCtx) 不得 panic
	out = captureStdout(func() {
		Context(context.Background())
	})
	if !strings.Contains(out, "context.emptyCtx") {
		t.Errorf("Context(Background) 输出缺少类型名, 实际输出: %s", out)
	}

	// 基本场景: 带 value 的 context 链不得 panic
	out = captureStdout(func() {
		Context(context.WithValue(context.Background(), "k1", "v1"))
	})
	if !strings.Contains(out, "context.valueCtx") {
		t.Errorf("Context(WithValue) 输出缺少类型名, 实际输出: %s", out)
	}

	// 基本场景: 自定义 context 结构体(含导出 struct 字段)不得 panic
	out = captureStdout(func() {
		Context(&probeCtx{Context: context.Background()})
	})
	if !strings.Contains(out, "Context:") {
		t.Errorf("Context(自定义结构体) 输出缺少嵌入字段, 实际输出: %s", out)
	}
}

// TestContextValues 测试 contextValues 函数的 nil、非结构体、valueCtx 链场景。
func TestContextValues(t *testing.T) {
	// 异常场景: nil 入参不得 panic
	out := captureStdout(func() {
		contextValues(nil)
	})
	if out != "" {
		t.Errorf("contextValues(nil) 应无输出, 实际输出: %s", out)
	}

	// 异常场景: 非结构体实现(context.Background 底层为 int)不得 panic
	out = captureStdout(func() {
		contextValues(context.Background())
	})
	if out != "" {
		t.Errorf("contextValues(Background) 应无输出, 实际输出: %s", out)
	}

	// 基本场景: valueCtx 链递归遍历不得 panic, 且能打印出 key 字段
	// 注: valueCtx 的 key/value 是未导出字段, 未经 unsafe 无法读取真实值, 仅打印类型名
	out = captureStdout(func() {
		contextValues(context.WithValue(context.WithValue(context.Background(), "k1", "v1"), "k2", "v2"))
	})
	if !strings.Contains(out, "key:") {
		t.Errorf("contextValues(valueCtx) 应打印出 key 字段, 实际输出: %s", out)
	}
}

// probeCtx 是测试用的自定义 context 结构体, 含导出 struct 字段以覆盖 IsNil 误用场景。
type probeCtx struct {
	context.Context
	Point struct{ X, Y int }
}
