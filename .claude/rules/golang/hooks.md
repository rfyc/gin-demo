---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go Hooks

> 本文件是 [common/hooks.md](../common/hooks.md) 的 Go 专项扩展。

## PostToolUse Hooks

在 `~/.claude/settings.json` 中配置：

- **gofmt/goimports**：编辑 `.go` 文件后自动格式化
- **go vet**：编辑 `.go` 文件后运行静态分析
- **staticcheck**：对修改的包运行扩展静态检查
