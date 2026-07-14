---
description: 强制执行 Go TDD 工作流。先写表驱动测试，再实现代码。用 go test -cover 验证覆盖率达到 80% 以上。
---

# Go TDD 命令

此命令使用惯用 Go 测试模式强制执行测试驱动开发方法论。

## 命令功能

1. **定义类型/接口**：先搭建函数签名
2. **编写表驱动测试**：创建全面的测试用例（RED）
3. **运行测试**：验证测试因正确原因失败
4. **实现代码**：编写最少量代码使测试通过（GREEN）
5. **重构**：在保持测试通过的前提下改进代码
6. **检查覆盖率**：确保达到 80% 以上

## 使用场景

以下情况使用 `/go-test`：
- 实现新 Go 函数时
- 为已有代码补充测试覆盖时
- 修复 Bug（先写失败测试）
- 构建核心业务逻辑
- 学习 Go 中的 TDD 工作流

## TDD 循环

```
RED     → 编写失败的表驱动测试
GREEN   → 实现最少量代码使测试通过
REFACTOR → 改进代码，测试保持通过
REPEAT  → 下一个测试用例
```

## 示例会话

````
用户：/go-test 我需要一个验证邮箱地址的函数

Agent：
# TDD 会话：邮箱验证器

## 第 1 步：定义接口

```go
// validator/email.go
package validator

// ValidateEmail 检查给定字符串是否是有效的邮箱地址。
// 有效返回 nil，否则返回描述问题的错误。
func ValidateEmail(email string) error {
    panic("尚未实现")
}
```

## 第 2 步：编写表驱动测试（RED）

```go
// validator/email_test.go
package validator

import (
    "testing"
)

func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        // 有效邮箱
        {"简单邮箱", "user@example.com", false},
        {"带子域名", "user@mail.example.com", false},
        {"带加号", "user+tag@example.com", false},
        {"带点号", "first.last@example.com", false},

        // 无效邮箱
        {"空字符串", "", true},
        {"无 @ 符号", "userexample.com", true},
        {"无域名", "user@", true},
        {"无本地部分", "@example.com", true},
        {"双 @", "user@@example.com", true},
        {"含空格", "user @example.com", true},
        {"无顶级域名", "user@example", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if tt.wantErr && err == nil {
                t.Errorf("ValidateEmail(%q) = nil；期望错误", tt.email)
            }
            if !tt.wantErr && err != nil {
                t.Errorf("ValidateEmail(%q) = %v；期望 nil", tt.email, err)
            }
        })
    }
}
```

## 第 3 步：运行测试 —— 验证失败（RED）

```bash
$ go test ./validator/...

--- FAIL: TestValidateEmail (0.00s)
    --- FAIL: TestValidateEmail/简单邮箱 (0.00s)
        panic: 尚未实现

FAIL
```

✓ 测试如预期失败（panic）。

## 第 4 步：实现最少量代码（GREEN）

```go
// validator/email.go
package validator

import (
    "errors"
    "regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

var (
    ErrEmailEmpty   = errors.New("邮箱不能为空")
    ErrEmailInvalid = errors.New("邮箱格式无效")
)

func ValidateEmail(email string) error {
    if email == "" {
        return ErrEmailEmpty
    }
    if !emailRegex.MatchString(email) {
        return ErrEmailInvalid
    }
    return nil
}
```

## 第 5 步：运行测试 —— 验证通过（GREEN）

```bash
$ go test ./validator/...

PASS
ok      project/validator    0.003s
```

✓ 所有测试通过！

## 第 6 步：检查覆盖率

```bash
$ go test -cover ./validator/...

PASS
coverage: 100.0% of statements
ok      project/validator    0.003s
```

✓ 覆盖率：100%

## TDD 完成！
````

## 测试模式

### 表驱动测试
```go
tests := []struct {
    name     string
    input    InputType
    want     OutputType
    wantErr  bool
}{
    {"用例 1", input1, want1, false},
    {"用例 2", input2, want2, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Function(tt.input)
        // 断言
    })
}
```

### 并行测试
```go
for _, tt := range tests {
    tt := tt // 捕获变量
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        // 测试体
    })
}
```

### 测试辅助函数
```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db := createDB()
    t.Cleanup(func() { db.Close() })
    return db
}
```

## 覆盖率命令

```bash
# 基础覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 在浏览器中查看
go tool cover -html=coverage.out

# 按函数查看覆盖率
go tool cover -func=coverage.out

# 带竞态检测
go test -race -cover ./...
```

## 覆盖率目标

| 代码类型 | 目标 |
|----------|------|
| 核心业务逻辑 | 100% |
| 公开 API | 90%+ |
| 通用代码 | 80%+ |
| 生成代码 | 排除 |

## TDD 最佳实践

**应当：**
- 在任何实现之前先写测试
- 每次改动后运行测试
- 使用表驱动测试实现全面覆盖
- 测试行为，而非实现细节
- 包含边界用例（空值、nil、最大值）

**不应当：**
- 在测试之前写实现
- 跳过 RED 阶段
- 直接测试私有函数
- 在测试中使用 `time.Sleep`
- 忽视不稳定的测试

## 相关命令

- `/go-build` —— 修复构建错误
- `/go-review` —— 实现后审查代码
- `/verify` —— 运行完整验证流程

## 相关资源

- Skill：`skills/golang-testing/`
- Skill：`skills/tdd-workflow/`
