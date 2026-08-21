package middleware

import (
	"fmt"
	"runtime/debug"

	"gin-demo/src/pkg/logger"
	"gin-demo/src/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recover 返回 panic 恢复中间件, 兜底捕获业务处理过程中的 panic。
//
// 捕获后记录错误日志与堆栈, 并返回 500 统一错误响应, 避免进程崩溃。
//
// 返回:
//   - gin.HandlerFunc: 包裹业务处理链, 捕获并处理 panic
func Recover() gin.HandlerFunc {

	return func(c *gin.Context) {

		// 延迟恢复, 兜底捕获后续处理链中的 panic
		defer func() {

			if err := recover(); err != nil {
				logger.Errorf(c.Request.Context(), "panic recovered: %v\nstack: %s", err, debug.Stack())
				response.AbortWithError(c, fmt.Errorf("服务器内部错误"), 500)
			}
		}()
		c.Next()
	}
}
