package middleware

import (
	"fmt"
	"time"

	"gin-demo/src/pkg/logger"
	"gin-demo/src/schema"

	"github.com/gin-gonic/gin"
)

// Logger 返回请求访问日志中间件, 按状态码分级记录请求信息。
//
// 记录内容: 请求ID、方法、路径、状态码、耗时、客户端IP、查询参数、User-Agent;
// 状态码 >= 500 记 Error, >= 400 记 Warn, 其余记 Info。
//
// 返回:
//   - gin.HandlerFunc: 请求处理前后统计耗时并输出访问日志
func Logger() gin.HandlerFunc {

	return func(c *gin.Context) {

		// 1. 记录请求开始时的基本信息, 供后续统计耗时与组装访问日志
		var (
			startTime  = time.Now()             // 请求开始时间, 用于计算处理耗时
			path       = c.Request.URL.Path     // 请求路径
			query      = c.Request.URL.RawQuery // 查询参数
			method     = c.Request.Method       // 请求方法
			ip         = c.ClientIP()           // 客户端IP
			userAgent  = c.Request.UserAgent()  // 客户端 User-Agent
			latency    time.Duration            // 请求处理耗时
			statusCode int                      // 响应状态码
			requestID  string                   // 请求ID(trace 中间件写入)
			msg        string                   // 组装后的访问日志内容
		)

		// 2. 放行请求, 交由后续中间件与业务处理
		c.Next()

		// 3. 请求结束后统计耗时、状态码与请求ID(从 request context 取, 与业务日志同源)
		{
			latency = time.Since(startTime)
			statusCode = c.Writer.Status()
			if v, ok := c.Request.Context().Value(schema.CTX_TraceIDKey).(string); ok {
				requestID = v
			}
		}

		// 4. 按状态码分级输出访问日志
		{
			msg = fmt.Sprintf("[%s] %s %s [%d] [%v] [%s] [%s] [%s]",
				requestID, method, path, statusCode, latency, ip, query, userAgent)
			if statusCode >= 500 {
				logger.Errorf(c.Request.Context(), msg)
			} else if statusCode >= 400 {
				logger.Warnf(c.Request.Context(), msg)
			} else {
				logger.Infof(c.Request.Context(), msg)
			}
		}
	}
}
