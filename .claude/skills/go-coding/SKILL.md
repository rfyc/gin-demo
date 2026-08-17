---
name: go-coding
description: 惯用 Go 模式、最佳实践与规范，用于构建健壮、高效、可维护的 Go 应用程序。
origin: ECC
---

# Go 开发模式

构建健壮、高效、可维护应用程序的惯用 Go 模式与最佳实践。

> **规范定位**: 本技能承载项目 Go 代码的强制规范（函数格式、注释、日志、测试）与惯用模式；CLAUDE.md 不再重复这些规范, 仅保留语言输出等全局约定。

## 激活时机

- 编写新 Go 代码时
- 编写测试时
- 审查 Go 代码时
- 重构已有 Go 代码时
- 设计 Go 包/模块时

## 强制规范（必须遵循）

### 函数编写

- 第一个参数必须是 `context.Context`, 用于传递取消信号、上下文信息
- 函数要短小精悍, 复杂逻辑拆分为多个子函数
- 所有变量在函数开头 `var ()` 中提前声明, 不使用 `:=`
- 每个变量必须带注释说明用途
- 涉及第三方请求(数据库/外部服务/网络)或者重要逻辑的函数必须记录日志: 耗时, error, 入参出参, 主要参数
- 处理错误必须包含上下文: 产生错误的函数名、关键参数值, 使用 `%w` 包装
- 逻辑复杂时拆分为多个小逻辑块, 一级逻辑块用 `{}` 包裹, 每个块带编号注释(1. 2. 2.1. 2.2.)说明层级与关联

### 注释要求

- **所有修改/新增的代码块、变量、函数、方法，必须添加详细注释**：
- 函数必须写 GoDoc 风格注释，说明用途、参数、返回值、错误场景
- 非公开函数至少说明用途和关键逻辑
- 复杂逻辑块需要行内注释说明意图

### 测试要求

- **所有修改/新增的代码，必须以「函数」为基本单元编写单元测试**：
- 未完成测试的代码，不得视为「完成」
- 测试文件位置:
  - 同包单元测试: 在被测代码所在目录下创建 `*_test.go`（可访问被测包的未导出成员）
  - 跨包/集成验证: 统一放在 `test/` 目录（独立包, 无法访问被测包的未导出成员）
- 测试case要覆盖基本的场景和异常场景
- 测试函数格式严格为:

```golang
func TestXxx(t *testing.T) {
    // 被测函数或代码逻辑块, 批量测试case
}
```

### 函数编写示例（必须照此风格编写）

```go
// GetUserOrders 查询指定用户的订单列表, 并填充每个订单的商品详情.
// 入参: ctx 请求上下文(携带超时与取消信号), userID 用户ID.
// 出参: orders 订单列表(含商品详情), err 查询失败时返回带上下文信息的错误.
func GetUserOrders(ctx context.Context, userID int64) (orders []*Order, err error) {
	
	var (
		db         *gorm.DB  // 数据库连接
		orders     []*Order  // 订单列表
		goods      []*Goods  // 商品列表
		goodsIDs   []int64   // 订单涉及的商品ID列表
		goodsMap   map[int64]*Goods // 商品ID到商品的索引
	)

	// 涉及第三方请求(数据库/外部服务)的函数, 必须记录处理日志: 耗时, error, 入参出参, 主要参数
	defer func(startTime time.Time) {
		if err != nil {
			log.Printf("GetUserOrders FAIL - userID: %v - err: %v - cost: %s", userID, err, time.Since(startTime))
		} else {
			log.Printf("GetUserOrders OK - userID: %v - orderCount: %d - cost: %s", userID, len(orders), time.Since(startTime))
		}
	}(time.Now())

	// 1. 查询订单列表; 直接调用函数同时处理 error, 尽量避免新开行判断错误
	if orders, err = queryOrders(ctx, userID); err != nil {
		return nil, fmt.Errorf("query orders fail: %w - userID: %v", err, userID)
	}

	// 2. 填充商品详情
	{
		// 2.1. 批量查询所有订单涉及的商品, 一次请求避免 N+1 问题
		if goodsIDs = collectGoodsIDs(orders); len(goodsIDs) > 0 {
			if goods, err = queryGoods(ctx, goodsIDs); err != nil {
				return nil, fmt.Errorf("query goods fail: %w - goodsIDs: %v", err, goodsIDs)
			}
		}
		// 2.2. 按商品ID建立索引, 逐单填充商品信息
		if goodsMap = indexGoods(goods); len(goodsMap) > 0 {
			for _, order := range orders {
				order.Goods = goodsMap[order.GoodsID]
			}
		}
	}
	return orders, nil
}
```

## 编码风格

### 不可变性（关键）

**始终**创建新对象，**绝不**修改已有对象：

```
// 伪代码
错误：modify(original, field, value) → 就地修改 original
正确：update(original, field, value) → 返回带有变更的新副本
```

原因：不可变数据可防止隐藏的副作用，使调试更容易，并支持安全的并发。

### 文件组织

多个小文件 > 少数大文件：
- 高内聚、低耦合
- 典型 200-400 行，最多 800 行
- 从大模块中提取工具函数
- 按功能/领域组织，而非按类型

### 错误处理

**始终**全面处理错误：
- 在每个层级显式处理错误
- 面向 UI 的代码提供用户友好的错误信息
- 服务端记录详细的错误上下文
- 绝不静默吞掉错误

### 输入验证

**始终**在系统边界处验证：
- 处理前验证所有用户输入
- 有条件时使用基于 schema 的验证
- 快速失败并给出清晰的错误信息
- 永远不信任外部数据（API 响应、用户输入、文件内容）

### 代码质量清单

标记工作完成前：
- [ ] 代码可读，命名清晰
- [ ] 函数短小（< 50 行）
- [ ] 文件专注（< 800 行）
- [ ] 无深度嵌套（> 4 层）
- [ ] 有适当的错误处理
- [ ] 无硬编码值（使用常量或配置）
- [ ] 无可变操作（使用不可变模式）
- [ ] 每个新增函数有对应单元测试

## 核心原则

### 简洁与清晰

Go 偏向简洁而非技巧。代码应直观易读。

```go
// 好：清晰直接
func GetUser(id string) (user *User, err error) {
    if user, err = db.FindUser(id);err != nil {
        return nil, fmt.Errorf("get user %s: %w", id, err)
    }
    return user, nil
}

// 差：过于技巧性
func GetUser(id string) (*User, error) {
    return func() (*User, error) {
        if u, e := db.FindUser(id); e == nil {
            return u, nil
        } else {
            return nil, e
        }
    }()
}
```

### 让零值有用

设计类型时，其零值无需初始化即可直接使用。

```go
// 好：零值有用
type Counter struct {
    mu    sync.Mutex
    count int // 零值为 0，可直接使用
}

func (c *Counter) Inc() {
    c.mu.Lock()
    c.count++
    c.mu.Unlock()
}

// 好：bytes.Buffer 零值可用
var buf bytes.Buffer
buf.WriteString("hello")

// 差：需要初始化
type BadCounter struct {
    counts map[string]int // nil map 会 panic
}
```

### 接受接口，返回结构体

函数应接受接口参数，返回具体类型。

```go
// 好：接受接口，返回具体类型
func ProcessData(r io.Reader) (res *Result, err error) {
    if data, err = io.ReadAll(r); err != nil {
        return nil, err
    }
    return &Result{Data: data}, nil
}

// 差：返回接口（不必要地隐藏实现细节）
func ProcessData(r io.Reader) (io.Reader, error) {
    // ...
}
```

## 错误处理模式

### 带上下文的错误包装

```go
// 好：包装错误并附带上下文
func LoadConfig(path string) (cfg *Config, err error) {
	var data []byte
    if data, err = os.ReadFile(path); err != nil {
        return nil, fmt.Errorf("load config %s: %w", path, err)
    }
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parse config %s: %w", path, err)
    }
    return cfg, nil
}
```

### 自定义错误类型

```go
// 定义领域特定错误
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("字段 %s 验证失败: %s", e.Field, e.Message)
}

// 常见情况的哨兵错误
var (
    ErrNotFound     = errors.New("资源未找到")
    ErrUnauthorized = errors.New("未授权")
    ErrInvalidInput = errors.New("输入无效")
)
```

### 使用 errors.Is 和 errors.As 检查错误

```go
func HandleError(err error) {
    // 检查特定错误
    if errors.Is(err, sql.ErrNoRows) {
        log.Println("未找到记录")
        return
    }

    // 检查错误类型
    var validationErr *ValidationError
    if errors.As(err, &validationErr) {
        log.Printf("字段 %s 验证错误: %s",
            validationErr.Field, validationErr.Message)
        return
    }

    // 未知错误
    log.Printf("意外错误: %v", err)
}
```

### 绝不忽略错误

```go
// 差：用空标识符忽略错误
result, _ := doSomething()

// 好：处理错误，或明确说明为何可以忽略
result, err := doSomething()
if err != nil {
    return err
}

// 可接受：错误确实无关紧要时（极少见）
_ = writer.Close() // 尽力清理，错误已在其他地方记录
```

## 并发模式

### Worker Pool

```go
func WorkerPool(jobs <-chan Job, results chan<- Result, numWorkers int) {
    var wg sync.WaitGroup

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                results <- process(job)
            }
        }()
    }

    wg.Wait()
    close(results)
}
```

### 使用 Context 取消和超时

```go
func FetchWithTimeout(ctx context.Context, url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch %s: %w", url, err)
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}
```

### 优雅关闭

```go
func GracefulShutdown(server *http.Server) {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    <-quit
    log.Println("正在关闭服务器...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("服务器强制关闭: %v", err)
    }

    log.Println("服务器已退出")
}
```

### 使用 errgroup 协调 Goroutine

```go
import "golang.org/x/sync/errgroup"

func FetchAll(ctx context.Context, urls []string) ([][]byte, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([][]byte, len(urls))

    for i, url := range urls {
        i, url := i, url // 捕获循环变量
        g.Go(func() error {
            data, err := FetchWithTimeout(ctx, url)
            if err != nil {
                return err
            }
            results[i] = data
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

### 避免 Goroutine 泄漏

```go
// 差：context 取消时 goroutine 泄漏
func leakyFetch(ctx context.Context, url string) <-chan []byte {
    ch := make(chan []byte)
    go func() {
        data, _ := fetch(url)
        ch <- data // 无接收方时永久阻塞
    }()
    return ch
}

// 好：正确处理取消
func safeFetch(ctx context.Context, url string) <-chan []byte {
    ch := make(chan []byte, 1) // 带缓冲的 channel
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

## 接口设计

### 小而专注的接口

```go
// 好：单方法接口
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// 按需组合接口
type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

### 在使用处定义接口

```go
// 在消费者包中定义，而非在提供者包中
package service

// UserStore 定义此 service 所需的能力
type UserStore interface {
    GetUser(id string) (*User, error)
    SaveUser(user *User) error
}

type Service struct {
    store UserStore
}

// 具体实现可以在另一个包中
// 它不需要知道此接口的存在
```

### 通过类型断言实现可选行为

```go
type Flusher interface {
    Flush() error
}

func WriteAndFlush(w io.Writer, data []byte) error {
    if _, err := w.Write(data); err != nil {
        return err
    }

    // 如果支持则 flush
    if f, ok := w.(Flusher); ok {
        return f.Flush()
    }
    return nil
}
```

## 包组织

### 标准项目布局

```text
myproject/
├── cmd/
│   └── myapp/
│       └── main.go           # 入口点
├── internal/
│   ├── handler/              # HTTP 处理器
│   ├── service/              # 业务逻辑
│   ├── repository/           # 数据访问
│   └── config/               # 配置
├── pkg/
│   └── client/               # 公开 API 客户端
├── api/
│   └── v1/                   # API 定义（proto、OpenAPI）
├── testdata/                 # 测试数据
├── go.mod
├── go.sum
└── Makefile
```

### 包命名

```go
// 好：简短、小写、无下划线
package http
package json
package user

// 差：冗长、大小写混合或多余后缀
package httpHandler
package json_parser
package userService // 多余的 'Service' 后缀
```

### 避免包级状态

```go
// 差：全局可变状态
var db *sql.DB

func init() {
    db, _ = sql.Open("postgres", os.Getenv("DATABASE_URL"))
}

// 好：依赖注入
type Server struct {
    db *sql.DB
}

func NewServer(db *sql.DB) *Server {
    return &Server{db: db}
}
```

## 结构体设计

### 函数式选项模式

```go
type Server struct {
    addr    string
    timeout time.Duration
    logger  *log.Logger
}

type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) {
        s.timeout = d
    }
}

func WithLogger(l *log.Logger) Option {
    return func(s *Server) {
        s.logger = l
    }
}

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{
        addr:    addr,
        timeout: 30 * time.Second, // 默认值
        logger:  log.Default(),    // 默认值
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// 用法
server := NewServer(":8080",
    WithTimeout(60*time.Second),
    WithLogger(customLogger),
)
```

### 嵌入实现组合

```go
type Logger struct {
    prefix string
}

func (l *Logger) Log(msg string) {
    fmt.Printf("[%s] %s\n", l.prefix, msg)
}

type Server struct {
    *Logger // 嵌入 —— Server 获得 Log 方法
    addr    string
}

func NewServer(addr string) *Server {
    return &Server{
        Logger: &Logger{prefix: "SERVER"},
        addr:   addr,
    }
}

// 用法
s := NewServer(":8080")
s.Log("启动中...") // 调用嵌入的 Logger.Log
```

## 内存与性能

### 已知大小时预分配切片

```go
// 差：切片多次扩容
func processItems(items []Item) []Result {
    var results []Result
    for _, item := range items {
        results = append(results, process(item))
    }
    return results
}

// 好：单次分配
func processItems(items []Item) []Result {
    results := make([]Result, 0, len(items))
    for _, item := range items {
        results = append(results, process(item))
    }
    return results
}
```

### 高频分配使用 sync.Pool

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func ProcessRequest(data []byte) []byte {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    buf.Write(data)
    // 处理...
    return buf.Bytes()
}
```

### 避免循环中字符串拼接

```go
// 差：产生大量字符串分配
func join(parts []string) string {
    var result string
    for _, p := range parts {
        result += p + ","
    }
    return result
}

// 好：使用 strings.Builder 单次分配
func join(parts []string) string {
    var sb strings.Builder
    for i, p := range parts {
        if i > 0 {
            sb.WriteString(",")
        }
        sb.WriteString(p)
    }
    return sb.String()
}

// 最佳：使用标准库
func join(parts []string) string {
    return strings.Join(parts, ",")
}
```

## Go 工具集成

### 常用命令

```bash
# 构建与运行
go build ./...
go run ./cli/myapp

# 测试
go test ./...
go test -race ./...
go test -cover ./...

# 静态分析
go vet ./...
staticcheck ./...
golangci-lint run

# 模块管理
go mod tidy
go mod verify

# 格式化
gofmt -w .
goimports -w .
```

### 推荐 Linter 配置（.golangci.yml）

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unconvert
    - unparam

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    check-shadowing: true

issues:
  exclude-use-default: false
```

## 快速参考：Go 惯用法

| 惯用法 | 说明 |
|--------|------|
| 接受接口，返回结构体 | 函数接受接口参数，返回具体类型 |
| 错误是值 | 将错误视为一等公民，而非异常 |
| 不要通过共享内存通信 | 使用 channel 协调 goroutine |
| 让零值有用 | 类型无需显式初始化即可工作 |
| 少量复制好过多余依赖 | 避免不必要的外部依赖 |
| 清晰优于技巧 | 优先考虑可读性而非聪明 |
| gofmt 是大家的朋友 | 始终用 gofmt/goimports 格式化 |
| 提前返回 | 先处理错误，保持主路径不缩进 |

## 应避免的反模式

```go
// 差：在长函数中使用裸返回
func process() (result int, err error) {
    // ... 50 行 ...
    return // 返回了什么？
}

// 差：用 panic 控制流程
func GetUser(id string) *User {
    user, err := db.Find(id)
    if err != nil {
        panic(err) // 不要这样做
    }
    return user
}

// 差：在结构体中存储 context
type Request struct {
    ctx context.Context // context 应作为第一个参数
    ID  string
}

// 好：context 作为第一个参数
func ProcessRequest(ctx context.Context, id string) error {
    // ...
}

// 差：混用值接收者和指针接收者
type Counter struct{ n int }
func (c Counter) Value() int { return c.n }    // 值接收者
func (c *Counter) Increment() { c.n++ }        // 指针接收者
// 选择一种风格并保持一致
```

**记住**：Go 代码应该以最好的方式"无聊"——可预测、一致、易于理解。有疑问时，保持简单。
