---
name: go-review
description: 全面的 Go 代码审查，涵盖惯用模式、并发安全、错误处理和安全性。在编写/修改 Go 代码、提交变更或审查 PR 时调用。
origin: ECC
---

# Go 代码审查

全面的 Go 专项代码审查技能，覆盖安全漏洞、并发正确性、错误处理、惯用模式与性能优化。

## 激活时机

- 编写或修改 Go 代码后
- 提交 Go 变更前
- 审查含 Go 代码的 Pull Request 时
- 熟悉新的 Go 代码库时
- 学习惯用 Go 模式时

## 审查流程

### 步骤 1：识别变更

执行以下命令，聚焦于已修改的 `.go` 文件：

```bash
git diff -- '*.go'
```

### 步骤 2：运行静态分析

```bash
# 基础分析
go vet ./...

# 高级检查（如已安装）
staticcheck ./...
golangci-lint run

# 竞态检测
go build -race ./...
go test -race ./...

# 安全漏洞
govulncheck ./...
```

### 步骤 3：逐文件深度审查

聚焦于已修改的 Go 文件，按以下优先级逐项检查。

## 审查优先级与分类

### 严重（CRITICAL，必须修复）

#### 安全类
- **SQL 注入**：`database/sql` 查询中使用字符串拼接，未使用参数化查询
- **命令注入**：`os/exec` 中使用未验证的输入拼接命令字符串
- **路径穿越**：用户控制的文件路径未经 `filepath.Clean` + 前缀检查
- **竞态条件**：共享状态（map、slice、变量）未加同步访问
- **unsafe 包**：无正当理由使用 `unsafe` 包
- **硬编码密钥**：源码中含 API 密钥、密码、Token 等敏感凭据
- **不安全 TLS**：`TLSClientConfig` 中设置 `InsecureSkipVerify: true`

#### 错误处理类
- **忽略错误**：使用 `_` 丢弃错误（极个别无关紧要的场景除外）
- **缺少错误包装**：直接 `return err` 而非 `fmt.Errorf("context: %w", err)`
- **可恢复错误使用 panic**：普通错误路径应返回 error 而非 panic
- **缺少 errors.Is/As**：错误判断应使用 `errors.Is(err, target)` 或 `errors.As`，而非 `err == target`

### 高（HIGH，应当修复）

#### 并发类
- **Goroutine 泄漏**：goroutine 无取消机制（应使用 `context.Context` 协调退出）
- **无缓冲 channel 死锁风险**：发送端无对应接收方时永久阻塞
- **缺少 sync.WaitGroup**：启动多个 goroutine 时缺少等待协调
- **Mutex 误用**：未使用 `defer mu.Unlock()`，可能导致死锁

#### 代码质量类
- **函数过长**：单个函数超过 50 行
- **嵌套过深**：条件嵌套超过 4 层
- **非惯用写法**：使用冗长的 `if/else` 而非提前返回（early return）
- **包级可变变量**：使用包级可变全局状态（应通过依赖注入）
- **接口污染**：定义了未被使用或过于宽泛的抽象接口

### 中（MEDIUM，考虑修复）

#### 性能类
- **循环中字符串拼接**：应使用 `strings.Builder` 或 `strings.Join`
- **切片未预分配**：已知大小时应使用 `make([]T, 0, cap)` 预分配
- **N+1 查询**：在循环中执行数据库查询
- **热路径不必要分配**：高频路径中频繁创建临时对象

#### 最佳实践类
- **context 位置**：`ctx context.Context` 应为函数第一个参数
- **未使用表驱动测试**：测试应使用表驱动模式覆盖多个场景
- **错误信息格式**：应小写开头，句末无标点
- **包命名**：应简短、全小写、无下划线
- **循环中 defer**：`defer` 在循环体中会导致资源累积

## 常见问题修复示例

### 示例 1：竞态条件（CRITICAL）

```go
// ❌ 差：共享 map 无同步访问
var cache = map[string]*Session{}

func GetSession(id string) *Session {
    return cache[id] // 竞态条件！
}
```

```go
// ✅ 好：使用 sync.RWMutex 保护
var (
    cache   = map[string]*Session{}
    cacheMu sync.RWMutex
)

func GetSession(id string) *Session {
    cacheMu.RLock()
    defer cacheMu.RUnlock()
    return cache[id]
}
```

### 示例 2：缺少错误上下文（HIGH）

```go
// ❌ 差：无上下文，错误追踪困难
return err
```

```go
// ✅ 好：包装错误并附带上下文
return fmt.Errorf("get user %s: %w", userID, err)
```

### 示例 3：Goroutine 泄漏（HIGH）

```go
// ❌ 差：无取消机制，context 取消时 goroutine 泄漏
func leakyFetch(ctx context.Context, url string) <-chan []byte {
    ch := make(chan []byte)
    go func() {
        data, _ := fetch(url)
        ch <- data // 无接收方时永久阻塞
    }()
    return ch
}
```

```go
// ✅ 好：正确处理取消和缓冲
func safeFetch(ctx context.Context, url string) <-chan []byte {
    ch := make(chan []byte, 1) // 带缓冲避免阻塞
    go func() {
        data, err := fetch(url)
        if err != nil {
            return
        }
        select {
        case ch <- data:
        case <-ctx.Done():
        }
    }()
    return ch
}
```

### 示例 4：非惯用提前返回（HIGH）

```go
// ❌ 差：主路径缩进过深
func Process(ctx context.Context, id string) (*Result, error) {
    if id != "" {
        user, err := GetUser(ctx, id)
        if err == nil {
            if user.Active {
                return ComputeResult(user), nil
            } else {
                return nil, fmt.Errorf("user inactive")
            }
        } else {
            return nil, fmt.Errorf("get user: %w", err)
        }
    } else {
        return nil, fmt.Errorf("empty id")
    }
}
```

```go
// ✅ 好：提前返回，主路径清晰
func Process(ctx context.Context, id string) (*Result, error) {
    if id == "" {
        return nil, fmt.Errorf("empty id")
    }
    user, err := GetUser(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }
    if !user.Active {
        return nil, fmt.Errorf("user inactive")
    }
    return ComputeResult(user), nil
}
```

### 示例 5：切片未预分配（MEDIUM）

```go
// ❌ 差：多次扩容
func processItems(items []Item) []Result {
    var results []Result
    for _, item := range items {
        results = append(results, process(item))
    }
    return results
}
```

```go
// ✅ 好：单次分配
func processItems(items []Item) []Result {
    results := make([]Result, 0, len(items))
    for _, item := range items {
        results = append(results, process(item))
    }
    return results
}
```

## 报告格式

### 审查报告结构

```markdown
# Go 代码审查报告

## 审查文件
- internal/handler/user.go（已修改）
- internal/service/auth.go（已修改）

## 静态分析结果
✓ go vet：无问题
✓ staticcheck：无问题
✓ go test -race：无竞态

## 发现问题

[严重] 竞态条件
文件：internal/service/auth.go:45
问题：共享 map 无同步访问
```go
// 问题代码片段
```
修复：使用 sync.RWMutex 或 sync.Map
```go
// 修复示例
```

[高] 缺少错误上下文
文件：internal/handler/user.go:28
问题：返回错误时无上下文
```go
return err  // 无上下文
```
修复：包装错误
```go
return fmt.Errorf("get user %s: %w", userID, err)
```

## 摘要
- 严重：1
- 高：1
- 中：0

建议：阻塞合并，修复严重问题后方可合并
```

## 审批标准

| 状态 | 条件 | 建议 |
|------|------|------|
| 通过（Approve） | 无 CRITICAL 或 HIGH 问题 | 可直接合并 |
| 警告（Warning） | 仅有 MEDIUM 问题 | 谨慎合并，建议择机修复 |
| 阻塞（Block） | 发现 CRITICAL 或 HIGH 问题 | 阻塞合并，必须修复 |

## 与其他技能配合

- 先用 `golang-testing` 确保测试通过
- 使用 `golang-patterns` 参考惯用 Go 写法
- 构建报错时先排查编译问题
- 非 Go 相关问题使用通用代码审查