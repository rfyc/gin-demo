---
description: 全面的 Go 代码审查，涵盖惯用模式、并发安全、错误处理和安全性。调用 go-reviewer agent。
---

# Go 代码审查

此命令调用 **go-reviewer** agent，进行全面的 Go 专项代码审查。

## 命令功能

1. **识别 Go 变更**：通过 `git diff` 找出修改的 `.go` 文件
2. **运行静态分析**：执行 `go vet`、`staticcheck` 和 `golangci-lint`
3. **安全扫描**：检查 SQL 注入、命令注入、竞态条件
4. **并发审查**：分析 goroutine 安全性、channel 用法、mutex 模式
5. **惯用 Go 检查**：验证代码是否符合 Go 规范和最佳实践
6. **生成报告**：按严重程度分类问题

## 使用场景

以下情况使用 `/go-review`：
- 编写或修改 Go 代码后
- 提交 Go 变更前
- 审查含 Go 代码的 Pull Request
- 熟悉新的 Go 代码库时
- 学习惯用 Go 模式时

## 审查分类

### 严重（CRITICAL，必须修复）
- SQL/命令注入漏洞
- 无同步的竞态条件
- Goroutine 泄漏
- 硬编码凭据
- 不安全指针使用
- 关键路径中忽略错误

### 高（HIGH，应当修复）
- 缺少带上下文的错误包装
- 用 panic 代替返回错误
- context 未传递
- 无缓冲 channel 导致死锁
- 接口未实现
- 缺少 mutex 保护

### 中（MEDIUM，考虑修复）
- 非惯用代码模式
- 导出符号缺少 godoc 注释
- 低效字符串拼接
- 切片未预分配
- 未使用表驱动测试

## 自动化检查

```bash
# 静态分析
go vet ./...

# 高级检查（如已安装）
staticcheck ./...
golangci-lint run

# 竞态检测
go build -race ./...

# 安全漏洞
govulncheck ./...
```

## 示例用法

```text
用户：/go-review

Agent：
# Go 代码审查报告

## 审查文件
- internal/handler/user.go（已修改）
- internal/service/auth.go（已修改）

## 静态分析结果
✓ go vet：无问题
✓ staticcheck：无问题

## 发现问题

[严重] 竞态条件
文件：internal/service/auth.go:45
问题：共享 map 无同步访问
```go
var cache = map[string]*Session{}  // 并发访问！

func GetSession(id string) *Session {
    return cache[id]  // 竞态条件
}
```
修复：使用 sync.RWMutex 或 sync.Map
```go
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

| 状态 | 条件 |
|------|------|
| 通过（Approve） | 无 CRITICAL 或 HIGH 问题 |
| 警告（Warning） | 仅有 MEDIUM 问题（谨慎合并） |
| 阻塞（Block） | 发现 CRITICAL 或 HIGH 问题 |

## 与其他命令配合

- 先用 `/go-test` 确保测试通过
- 构建报错时用 `/go-build`
- 提交前用 `/go-review`
- 非 Go 相关问题用 `/code-review`

## 相关资源

- Agent：`agents/go-reviewer.md`
- Skill：`skills/golang-patterns/`、`skills/golang-testing/`
