package schema

// CTXTraceKey 是 context 与请求头中存储链路追踪 trace_id 的 key。
// trace 中间件写入、logger 中间件读取, 统一引用本常量避免硬编码。
const CTXTraceKey = "trace_id"
