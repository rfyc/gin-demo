package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是统一接口响应体。
type Response struct {
	Code    int         `json:"code"` // Code 业务状态码, 0 表示成功
	Message string      `json:"msg"`  // Message 提示信息
	Data    interface{} `json:"data"` // Data 业务数据
}

// Success 以 200 状态码写出成功响应。
//
// 参数:
//   - c: gin 上下文
//   - data: 业务数据, 为 nil 时写空对象
func Success(c *gin.Context, data interface{}) {

	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Fail 写出失败响应, 默认状态码 -1, 可通过 code 覆盖。
//
// 参数:
//   - c: gin 上下文
//   - err: 错误信息, 非 nil 时写入 message
//   - code: 可选业务状态码, 缺省为 -1
func Fail(c *gin.Context, err error, code ...int) {

	writeFail(c, err, code, false)
}

// AbortWithError 终止后续处理并写出失败响应。
// 与 Fail 的区别: 会调用 Abort 阻止后续中间件继续执行。
//
// 参数:
//   - c: gin 上下文
//   - err: 错误信息, 非 nil 时写入 message
//   - code: 可选业务状态码, 缺省为 -1
func AbortWithError(c *gin.Context, err error, code ...int) {

	writeFail(c, err, code, true)
}

// writeFail 组装失败响应体并写出; abort 为 true 时终止后续处理。
//
// 参数:
//   - c: gin 上下文
//   - err: 错误信息, 非 nil 时写入 message
//   - code: 业务状态码列表, 为空时取默认 -1
//   - abort: 是否调用 Abort 终止后续处理
func writeFail(c *gin.Context, err error, code []int, abort bool) {

	var (
		errCode int      = -1 // 业务状态码, 默认 -1
		message string        // 错误提示信息
		body    Response      // 组装后的响应体
	)

	if len(code) > 0 {
		errCode = code[0]
	}
	if err != nil {
		message = err.Error()
	}

	body = Response{
		Code:    errCode,
		Message: message,
		Data:    struct{}{},
	}
	if abort {
		c.AbortWithStatusJSON(http.StatusOK, body)
	} else {
		c.JSON(http.StatusOK, body)
	}
}
