---
name: testing
description: 项目单元测试规范：新增/修改的代码一律写单元测试，分功能函数与业务函数两种单元测试形式，均用表驱动覆盖正常/边界/异常；依赖打桩用 bytedance/mockey。编写或修改 Go 测试时调用。
---

# Go 单元测试

> **规范定位**: 本技能承载项目单元测试的强制规范（两种形式、测试文件位置、表驱动形态、覆盖率验收、依赖 mock）。其他测试手段（基准、模糊、快照等）按需使用；竞态检测由 review 的 `go test -race` 承担。

## 单元测试

所有新增/修改的代码，必须以「函数」为基本单元编写单元测试；未完成测试的代码，不得视为「完成」。单元测试分两种形式：

> **测试注释要求**:每个测试函数尽可能添加 GoDoc 注释,说明被测对象与覆盖的场景(正常/边界/异常),便于后续维护者快速理解测试意图。
>
> **函数内部场景注释**:测试函数内部对每个测试场景/断言块尽可能添加 `// case: xxxx` 注释,说明该场景的输入与期望结果;表驱动测试的 case 名(如 "正常: xxx"/"边界: xxx")本身已说明场景,循环内无需重复添加。已有的 "// 基本场景:"/"// 异常场景:" 等写法统一改用 "// case:"。

### 形式一：功能函数单元测试

针对功能函数（工具函数、纯函数），直接调用，验证输入输出与错误行为：

- 位置：改动文件同目录 `xxx_test.go`（同包，可访问被测包的未导出成员）
- 形态：表驱动测试，case 覆盖正常/边界/异常
- 被测对象：如 `fn(input) (output, error)`，无 ctx；测试写法与下方统一示例相同，仅调用时无 ctx 参数

### 形式二：业务函数单元测试

针对业务处理函数（handler 背后的业务逻辑，形如 `fn(ctx, request) (response, error)`），直接调用，验证业务逻辑与错误行为：

- 位置：业务函数同目录 `xxx_test.go`（同包）
- 被测对象：直接调用业务函数（如 `home.Welcome(utils.NewTestContext(), &WelcomeRequest{...})`），**不经过 HTTP 层**；gin 参数绑定与响应逻辑由 review 检查，不单独写接口测试
- 形态：表驱动测试，case 覆盖正常/边界/异常

### 统一示例（两种形式同一写法）

```go
// welcome_test.go(与 welcome.go 同目录)
package home

import (
	"testing"
	"gin-demo/src/utils"
	"github.com/stretchr/testify/assert"
)

func TestWelcome(t *testing.T) {
	
	tests := []struct {
		name    string
		request *WelcomeRequest
		want    *WelcomeResponse
		wantErr error
	}{
		{"正常: 带ID", &WelcomeRequest{ID: 123}, &WelcomeResponse{Message: "welcome"}, nil},
		{"边界: 零值", &WelcomeRequest{}, &WelcomeResponse{Message: "welcome"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := Welcome(utils.NewTestContext(), tt.request)
			// wantErr 为 nil 时等价于断言无错误, 非 nil 时校验特定错误
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, resp)
		})
	}
}
```

## 表驱动规范

- case 字段：`name` + 输入字段 + 期望字段（`want` / `wantErr`，`wantErr` 为 `error` 类型，`nil` 表示期望无错误）
- 用 `t.Run(tt.name, ...)` 包裹，失败时输出 `TestXxx/用例名` 精确定位
- 每个测试至少覆盖三类 case：正常、边界（空/零/上限/越界）、异常（非法输入、错误路径）
- 断言统一用 testify：
  - `assert.Equal(t, want, got)`：深度相等，结构体无需 `reflect.DeepEqual`
  - `assert.ErrorIs(t, err, wantErr)`：`wantErr` 为 `nil` 时等价于断言无错误，非 nil 时校验特定错误
  - 需要失败立即中断时用 `require` 对应函数（如 `require.NoError`）
- 测试 context 统一用 `utils.NewTestContext()`

## Mock 依赖（按需）

依赖打桩统一用 `github.com/bytedance/mockey`，**仅**在本地不方便访问 / 访问慢的函数、接口上使用（第三方 API、外部服务等）：

- 安装：`go get github.com/bytedance/mockey`
- 用法：`mockey.PatchConvey(func(){ mockey.Mock(Fn).Return(v, err).Build(); ... })`，回调结束自动恢复打桩

```go
func TestFetchUser(t *testing.T) {
	
	mockey.PatchConvey(func() {
		
		// 打桩远程调用(本地无法访问/访问慢)
		mockey.Mock(FetchRemoteUser).Return(&User{ID: "123", Name: "Alice"}, nil).Build()

		var svc = NewUserService()
		var user, err = svc.GetUser("123")
		assert.NoError(t, err)
		assert.Equal(t, "Alice", user.Name)
	})
}
```

- 注意：
  - 运行时**全局打桩**，不能与 `t.Parallel()` 混用
  - 目标函数被内联导致打桩失败时，用 `go test -gcflags=all=-l` 关闭内联
  - 打桩必须恢复（`PatchConvey` 自动恢复；或 `patch := mockey.Mock(...).Build(); defer patch.UnPatch()`）

## 覆盖率验收

核心业务逻辑 100%、公开 API 90%+、通用代码 80%+

## 测试命令

**验证顺序**（运行 go 命令前先按 where 定位 go 版本）：`go build ./...` → `go vet ./...` → 按下述命令运行本次修改相关包的测试；任一步失败不得视为完成。

**每次修改后验证只运行本次修改相关包的测试，不执行全部测试**：
- 指定包：`go test ./src/utils/`
- 指定测试函数：`go test -run TestFileName ./src/utils/`
- 修改影响下游（如 `src/core` 被业务包依赖）时，才连带运行相关包
- 全量 `go test ./...` 仅在全量回归或提交前使用

```bash
# 运行全部测试(仅全量回归/提交前使用)
go test ./...

# 运行本次修改相关包的测试
go test ./src/utils/

# 运行指定测试函数
go test -run TestFileName ./src/utils/

# 覆盖率
go test -cover ./...
```

## 补充

- 测试完成后：调用 review 对本次变更进行全面审查，修复其发现的问题
- 修复 Bug 时：先按本技能写失败测试复现，修复后重新运行测试确认无回归（与 review 配合）
- 辅助机制：测试需要 setup/清理/临时文件时用 `t.Helper()` / `t.Cleanup()` / `t.TempDir()`
