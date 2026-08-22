package schema

// contextKey 是 context 中间值的私有 key 类型, 避免与其他包 key 冲突。
type contextKey string

// CTX_TraceIDKey 是 context 与请求头中存储链路追踪 trace_id 的 key。
// Trace 中间件写入 request context, logger 读取并注入日志字段,
// 统一引用本常量避免各处 key 不一致导致链路追踪断裂。
const CTX_TraceIDKey contextKey = "trace_id"
