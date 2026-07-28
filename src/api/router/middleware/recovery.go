package middleware

import (
	"fmt"
	"runtime/debug"

	"gin-demo/src/pkg/logger"
	"gin-demo/src/pkg/response"

	"github.com/gin-gonic/gin"
)

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
