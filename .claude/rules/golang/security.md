---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go 安全

> 本文件是 [common/security.md](../common/security.md) 的 Go 专项扩展。

## 密钥管理

```go
apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
    log.Fatal("OPENAI_API_KEY 未配置")
}
```

## 安全扫描

- 使用 **gosec** 进行静态安全分析：
  ```bash
  gosec ./...
  ```

## Context 与超时

始终使用 `context.Context` 控制超时：

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```
