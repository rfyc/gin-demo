package middleware

import (
	"context"

	"gin-demo/src/schema"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Trace 返回链路追踪中间件, 为请求注入 trace_id。
//
// 优先透传请求头中的 trace_id, 缺失时自动生成 UUID;
// 生成的 trace_id 写入 request context、gin 上下文与响应头,
// 供访问日志、业务日志与调用方串联同一请求。
//
// 返回:
//   - gin.HandlerFunc: 注入 trace_id 后放行请求
func Trace() gin.HandlerFunc {

	return func(c *gin.Context) {

		// 1. 从请求头读取 trace_id, 缺失时生成新 UUID
		var (
			requestID = c.GetHeader(string(schema.CTX_TraceIDKey)) // 请求ID: 优先取请求头, 缺失则自动生成 UUID
		)

		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 2. 写入 request context(gin Keys 与业务 ctx 共用同一 key),
		//    并写入响应头, 供访问日志、业务日志与调用方使用
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), schema.CTX_TraceIDKey, requestID))
		c.Set(string(schema.CTX_TraceIDKey), requestID)
		c.Header(string(schema.CTX_TraceIDKey), requestID)

		// 3. 放行请求至后续处理
		c.Next()
	}
}
