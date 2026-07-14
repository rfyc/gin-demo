package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是所有 API 返回的统一格式（参照 handout 项目）
type Response struct {
	Code    int         `json:"code"` // 业务错误码，0 表示成功
	Stat    int         `json:"stat"` // 状态：0 失败，1 成功
	Message string      `json:"msg"`  // 错误/成功消息
	Data    interface{} `json:"data"` // 响应数据
}

// Success 返回成功响应（HTTP 200）
// stat=1, code=0, msg="success"
func Success(c *gin.Context, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Stat:    1,
		Message: "success",
		Data:    data,
	})
}

// Fail 返回失败响应
// stat=0，code 默认 -1，返回空结构体避免 null
func Fail(c *gin.Context, err error, code ...int) {
	errCode := -1
	if len(code) > 0 {
		errCode = code[0]
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	c.JSON(http.StatusOK, Response{
		Code:    errCode,
		Stat:    0,
		Message: message,
		Data:    struct{}{},
	})
}

// AbortWithError 中止请求并返回错误
func AbortWithError(c *gin.Context, err error, code ...int) {
	errCode := -1
	if len(code) > 0 {
		errCode = code[0]
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	c.AbortWithStatusJSON(http.StatusOK, Response{
		Code:    errCode,
		Stat:    0,
		Message: message,
		Data:    struct{}{},
	})
}
