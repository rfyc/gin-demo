package middleware

import (
	"fmt"
	"time"

	"gin-demo/src/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Logger 返回一个将请求信息写入日志的 gin 中间件。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		requestID := ""
		if v, ok := c.Get("X-Request-ID"); ok {
			requestID = v.(string)
		}

		msg := fmt.Sprintf("[%s] %s %s [%d] [%v] [%s] [%s] [%s]",
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
