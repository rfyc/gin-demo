---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go 编码风格

> 本文件是 [common/coding-style.md](../common/coding-style.md) 的 Go 专项扩展。

## 格式化

- **gofmt** 和 **goimports** 是强制要求 —— 不存在风格争议

## 设计原则

- 接受接口，返回结构体
- 保持接口小巧（1-3 个方法）

## 错误处理

始终包装错误并附带上下文：

```go
if err != nil {
    return fmt.Errorf("创建用户失败: %w", err)
}
```

## 参考

完整 Go 惯用模式，参见 skill：`golang-patterns`。
