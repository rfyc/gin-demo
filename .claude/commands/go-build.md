---
description: 增量修复 Go 构建错误、go vet 警告和 linter 问题。调用 go-build-resolver agent 进行最小化精准修复。
---

# Go 构建与修复

此命令调用 **go-build-resolver** agent，以最小化改动增量修复 Go 构建错误。

## 命令功能

1. **运行诊断**：执行 `go build`、`go vet`、`staticcheck`
2. **解析错误**：按文件分组，按严重程度排序
3. **增量修复**：每次修复一个错误
4. **验证每次修复**：每次改动后重新构建
5. **输出摘要**：显示已修复内容和剩余问题

## 使用场景

以下情况使用 `/go-build`：
- `go build ./...` 报错失败
- `go vet ./...` 发现问题
- `golangci-lint run` 显示警告
- 模块依赖损坏
- 拉取代码后构建中断

## 执行的诊断命令

```bash
# 主要构建检查
go build ./...

# 静态分析
go vet ./...

# 扩展 lint（如已安装）
staticcheck ./...
golangci-lint run

# 模块问题
go mod verify
go mod tidy -v
```

## 示例会话

```text
用户：/go-build

Agent：
# Go 构建修复

## 初始诊断

```bash
$ go build ./...
internal/service/user.go:25:15: undefined: UserRepository
internal/handler/api.go:42:9: cannot use x (type string) as type int
internal/handler/api.go:58:2: missing return at end of function
```

发现错误：3 个

## 修复 1：未定义标识符

文件：internal/service/user.go:25
错误：undefined: UserRepository
原因：缺少导入

```go
// 添加导入
import "project/internal/repository"

// 修改前
var repo UserRepository
// 修改后
var repo repository.UserRepository
```

```bash
$ go build ./...
# 剩余 2 个错误
```

## 修复 2：类型不匹配

文件：internal/handler/api.go:42
错误：cannot use x (type string) as type int

```go
// 修改前
count := params.Get("count")
// 修改后
countStr := params.Get("count")
count, _ := strconv.Atoi(countStr)
```

```bash
$ go build ./...
# 剩余 1 个错误
```

## 修复 3：缺少 return

文件：internal/handler/api.go:58
错误：missing return at end of function

```go
func GetUser(id string) (*User, error) {
    if id == "" {
        return nil, ErrInvalidID
    }
    user := findUser(id)
    // 添加缺少的 return
    return user, nil
}
```

```bash
$ go build ./...
# 构建成功！
```

## 最终验证

```bash
$ go vet ./...
# 无问题

$ go test ./...
ok      project/internal/service   0.015s
ok      project/internal/handler   0.023s
```

## 摘要

| 指标 | 数量 |
|------|------|
| 修复构建错误 | 3 |
| 修复 vet 警告 | 0 |
| 修改文件 | 2 |
| 剩余问题 | 0 |

构建状态：通过 ✓
```

## 常见错误修复

| 错误 | 典型修复 |
|------|----------|
| `undefined: X` | 添加导入或修正拼写 |
| `cannot use X as Y` | 类型转换或修正赋值 |
| `missing return` | 添加 return 语句 |
| `X does not implement Y` | 添加缺失的方法 |
| `import cycle` | 重构包结构 |
| `declared but not used` | 删除或使用变量 |
| `cannot find package` | `go get` 或 `go mod tidy` |

## 修复策略

1. **先修构建错误** —— 代码必须能编译
2. **再修 vet 警告** —— 修复可疑结构
3. **最后修 lint 警告** —— 风格和最佳实践
4. **每次修一个** —— 验证每次改动
5. **最小化改动** —— 不重构，只修错误

## 停止条件

以下情况 agent 将停止并上报：
- 同一错误经过 3 次尝试仍存在
- 修复引入了更多错误
- 需要架构级变更
- 缺少外部依赖

## 相关命令

- `/go-test` —— 构建成功后运行测试
- `/go-review` —— 审查代码质量
- `/verify` —— 完整验证流程

## 相关资源

- Agent：`agents/go-build-resolver.md`
- Skill：`skills/golang-patterns/`
