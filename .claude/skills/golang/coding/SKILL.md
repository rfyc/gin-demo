---
name: coding
description: 惯用 Go 模式、最佳实践与强制规范（函数格式、注释、日志、测试），用于构建健壮、高效、可维护的 Go 应用。编写/修改/审查/重构 Go 代码、编写测试、设计 Go 包/模块时调用。
---

# Go 开发模式

构建健壮、高效、可维护应用程序的惯用 Go 模式与最佳实践。

> **规范定位**: 本技能承载项目 Go 代码的强制规范（函数格式、注释、日志）与惯用模式；测试强制规范由 testing 承担，本技能仅要求函数对测试友好；CLAUDE.md 不再重复这些规范，仅保留语言输出等全局约定。

## 强制规范（必须遵循）

### 函数编写与注释

#### 函数要求

- 第一个参数必须是 `context.Context`，用于传递取消信号、上下文信息
- 函数要短小精悍，过长时判断是否封装为独立子函数
- 函数体与大逻辑块的开括号 `{` 之后必须空一行，再写块内内容；大逻辑块之间用空行分隔
- 所有变量在函数开头 `var ()` 中提前声明，不使用 `:=`
- 每个变量必须带注释说明用途
- 能在 `var ()` 声明处直接初始化的变量（无依赖、单表达式、非错误处理流程），一律在声明处初始化并注释，不在逻辑块中延迟赋值；仅当依赖前序计算结果（如 `if x, err = f(); err != nil` 中的 `x`、依赖上一步返回值的变量）时，才先声明后赋值
- 涉及第三方请求(数据库/外部服务/网络)或者重要逻辑的函数必须记录日志: 耗时，error，入参出参，主要参数
- 处理错误必须包含上下文: 产生错误的函数名、关键参数值，使用 `%w` 包装
- 逻辑复杂时拆分为多个小逻辑块，每个一级逻辑块带编号注释(1. 2. 2.1. 2.2.)说明层级与关联; **同一函数内有多个一级逻辑块时, 每个逻辑块都必须用 `{}` 包裹; 仅当整个函数只有一个逻辑块时可不使用 `{}`; 逻辑块体内仅 1~2 行时也可不使用 `{}`; `if(){}`/`for(){}`/`if(){}else{}` 这类逻辑块即单条控制流语句的, 自带的 `{}` 即视为已包裹, 不再额外套一层**; 过长时封装为独立子函数，子函数内部同样遵循逻辑块书写格式

#### 注释要求

- **所有修改/新增的代码块、变量、函数、方法，必须添加详细注释**：
  - 函数必须写 GoDoc 风格注释，说明用途、参数、返回值、错误场景
  - 非公开函数至少说明用途和关键逻辑
  - 复杂逻辑块需要行内注释说明意图
  - 注释模板针对实际业务代码强制生效；文档内的教学示例用于展示模式，用途由章节标题说明，可不套用完整模板

#### 函数编写示例（必须照此风格编写）

##### 注释格式模板

```go
// HandleFunc 将业务处理函数适配为 gin 的 HTTP 处理函数。
//
// 泛型参数:
//   - input:  请求类型，可为 struct 或 struct 指针（如 UserReq 或 *UserReq）
//   - output: 响应类型，最终序列化为接口响应体
//
// 参数:
//   - ctx: 请求上下文，透传给 fn，用于传递取消信号、链路信息等
//   - fn: 业务处理函数，签名 fn(ctx, request) (output, error)；
//     ctx 为透传的请求上下文，request 为绑定后的请求体
//
// 返回:
//   - gin.HandlerFunc：负责请求绑定、调用 fn，并按结果写回统一响应
//
// 错误场景:
//   - 请求体绑定失败：response.Fail(c, err, -1)
//   - fn 返回 error：response.Fail(c, err, -2)
//   - 成功：response.Success(c, resp)
func HandleFunc[input IRequest, output IResponse](ctx context.Context, fn func(ctx context.Context, request input) (output, error)) func(c *gin.Context) {
	...
```

- 第一句：「函数名 + 一句话用途」，句号结尾
- 小节标题统一用半角冒号：`泛型参数:`、`参数:`、`返回:`、`错误场景:`
- 列表项用 `- ` 前缀，续行缩进两空格
- 小节按函数实际情况取舍：无泛型时省略「泛型参数」；简单函数可精简为「参数 / 返回」两段

##### 完整示例

```go
// GetUserOrders 查询指定用户的订单列表，并填充每个订单的商品详情。
//
// 参数:
//   - ctx: 请求上下文，携带超时与取消信号
//   - userID: 用户ID
//
// 返回:
//   - orders: 订单列表，含商品详情
//   - err: 查询失败时返回带上下文信息的错误
//
// 错误场景:
//   - 查询订单列表失败：返回带用户ID上下文的错误
//   - 查询商品列表失败：返回带商品ID列表上下文的错误
func GetUserOrders(ctx context.Context, userID int64) (orders []*Order, err error) {

	var (
		goodsIDs []int64          // 订单涉及的商品ID列表
		goods    []*Goods         // 商品列表
		goodsMap map[int64]*Goods // 商品ID到商品的索引
	)

	// 涉及第三方请求(数据库/外部服务)的函数, 必须记录处理日志: 耗时, error, 入参出参, 主要参数
	// 日志统一用项目 src/pkg/logger 包级函数(logger.Infof/logger.Errorf), 自动附加 ctx 中的 request_id
	defer func(startTime time.Time) {
		if err != nil {
			logger.Errorf(ctx, "GetUserOrders FAIL - userID: %v - err: %v - cost: %s", userID, err, time.Since(startTime))
		} else {
			logger.Infof(ctx, "GetUserOrders OK - userID: %v - orderCount: %d - cost: %s", userID, len(orders), time.Since(startTime))
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

### 输入验证

**始终**在系统边界处验证：
- 处理前验证所有用户输入
- 有条件时使用基于 schema 的验证
- 快速失败并给出清晰的错误信息
- 永远不信任外部数据（API 响应、用户输入、文件内容）

### 代码质量清单

标记工作完成前：
- [ ] 代码可读，命名清晰
- [ ] 函数短小
- [ ] 文件专注（< 800 行）
- [ ] 无深度嵌套（> 4 层）
- [ ] 有适当的错误处理
- [ ] 无硬编码值（使用常量或配置）
- [ ] 每个新增函数有对应单元测试

## 核心原则

### 让零值有用

类型的零值应无需初始化即可直接使用：
- 结构体字段含锁（如 `sync.Mutex`）时无需显式初始化
- 内嵌 map/slice 字段保证零值安全，或在使用处显式 `make`
- 零值应表示合法的默认状态

**避免**：设计需要显式初始化才能使用的类型（如字段为 nil map 会 panic）。

### 接受接口，返回结构体

函数签名遵循：
- 参数声明为接口，可接收任意具体类型实现
- 返回值声明为具体类型，隐藏实现细节

**避免**：在 API 边界返回接口（不必要地隐藏实现细节）。

## 错误处理模式

- **包装**：返回错误时用 `fmt.Errorf` 带上下文包装，如 `fmt.Errorf("get user %s: %w", userID, err)`——前缀写明函数名与关键参数值，保留 `errors.Is`/`errors.As` 能力，不用 `%v`
- **哨兵错误**：定义包级 `var ErrXxx = errors.New("xxx")`，调用方用 `errors.Is` 判断
- **自定义错误类型**：错误需携带额外字段时定义类型，调用方用 `errors.As` 提取
- **判断**：一律用 `errors.Is`/`errors.As`，不用 `==` 比较错误
- **绝不忽略**：不静默丢弃错误；确需忽略时显式 `_ =` 并注释原因（极少数）

## 并发模式

- **限流 / Worker Pool**：限制并发数用 `errgroup.Group.SetLimit` 或手写 worker 循环；所有 worker 结束后再 `close(results)`，避免向无接收方的 channel 发送
- **Context 取消与超时**：`context.WithTimeout`/`WithCancel` 创建子 context 后必须 `defer cancel()`；网络/DB 等下游调用一律透传 ctx；请求级 ctx 不存入结构体
- **errgroup**：`errgroup.WithContext` 在任一 goroutine 出错时自动取消其余；goroutine 内 `return err` 由 `g.Wait()` 汇总；注意循环变量捕获（Go 1.22 前需 `i := i`）
- **Goroutine 泄漏**：每个 goroutine 必须有退出路径——channel 带缓冲或用 `select` + `ctx.Done()`；避免无缓冲 channel 只发不收；用 `WaitGroup`/errgroup 等待 goroutine 结束
- **数据竞争**：多个 goroutine 共享 map/变量时用锁或 channel 协调，不用共享内存裸写

## 接口设计

- **小而专注**：接口尽量单方法，按需组合（如 `Reader`/`Writer`/`Closer` 组合成 `ReadWriteCloser`），避免庞大接口
- **在使用处定义**：接口定义在消费者包，而非提供者包；具体实现不感知接口存在
- **可选行为**：探测实现是否支持某能力时用类型断言 `if f, ok := w.(Flusher); ok { ... }`

## 结构体设计

- **函数式选项模式**：构造参数多时，定义 `type Option func(*Server)` 与 `WithTimeout` 等选项函数，`NewServer(addr string, opts ...Option)` 设默认值后逐个应用
- **嵌入实现组合**：结构体嵌入接口/类型获得其方法（如 `type Server struct{ *Logger }`），组合优于继承；嵌入字段名即类型名

## 内存与性能

- **预分配切片**：已知大小时 `make([]T, 0, len)` 预分配，避免 append 反复扩容
- **字符串拼接**：循环拼接用 `strings.Builder` 或 `strings.Join`，不用 `+=`
- **sync.Pool 对象池**：高频分配对象用 `sync.Pool` 复用；Get 后先 `Reset` 再用、`defer Put` 放回；放回后不得再引用其数据（需保留须先复制）

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

// 差：混用值接收者和指针接收者
type Counter struct{ n int }

func (c Counter) Value() int  { return c.n } // 值接收者
func (c *Counter) Increment() { c.n++ }      // 指针接收者
// 选择一种风格并保持一致
```
