package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

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
		Message: message,
		Data:    struct{}{},
	})
}

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
		Message: message,
		Data:    struct{}{},
	})
}
