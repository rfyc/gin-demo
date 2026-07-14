package middleware

import (
	"fmt"
	"runtime/debug"

	"artist/application/pkg/logger"
	"artist/application/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery 捕获 panic，记录堆栈，并返回 500 响应。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf(c.Request.Context(), "panic recovered: %v\nstack: %s", err, debug.Stack())
				response.AbortWithError(c, fmt.Errorf("服务器内部错误"), 500)
			}
		}()
		c.Next()
	}
}
