package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 返回跨域资源共享(CORS)处理中间件。
//
// 返回:
//   - gin.HandlerFunc: 设置允许跨域请求的响应头;
//     OPTIONS 预检请求直接以 204 终止, 其余请求放行
func CORS() gin.HandlerFunc {

	return func(c *gin.Context) {

		// 1. 设置允许跨域请求的响应头
		{
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		}

		// 2. OPTIONS 预检请求直接终止, 不进入后续处理
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		// 3. 其余请求放行至后续处理
		c.Next()
	}
}
