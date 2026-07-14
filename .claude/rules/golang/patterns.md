---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go 模式

> 本文件是 [common/patterns.md](../common/patterns.md) 的 Go 专项扩展。

## 函数式选项模式

```go
type Option func(*Server)

func WithPort(port int) Option {
    return func(s *Server) { s.port = port }
}

func NewServer(opts ...Option) *Server {
    s := &Server{port: 8080}
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

## 小接口

在使用处定义接口，而非在实现处定义。

## 依赖注入

使用构造函数注入依赖：

```go
func NewUserService(repo UserRepository, logger Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}
```

## 参考

完整的 Go 模式（包括并发、错误处理、包组织），参见 skill：`golang-patterns`。
