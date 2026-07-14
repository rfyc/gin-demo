---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go 测试

> 本文件是 [common/testing.md](../common/testing.md) 的 Go 专项扩展。

## 测试框架

使用标准 `go test` 配合**表驱动测试**。

## 竞态检测

始终带 `-race` 标志运行：

```bash
go test -race ./...
```

## 覆盖率

```bash
go test -cover ./...
```

## 参考

详细 Go 测试模式和辅助函数，参见 skill：`golang-testing`。
