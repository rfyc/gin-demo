package core

import (
	"context"
	"gin-demo/src/pkg/response"
	"reflect"

	"github.com/gin-gonic/gin"
)

type IRequest interface{}
type IResponse interface{}

func HandleFunc[input IRequest, output IResponse](fn func(ctx context.Context, request input) (output, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		var request input
		var bindTarget any
		// 必须在泛型函数内操作 input 类型，传入 any 后反射到的是 interface 层而非具体类型
		if ref := reflect.ValueOf(&request).Elem(); ref.Kind() == reflect.Ptr {
			ref.Set(reflect.New(ref.Type().Elem()))
			bindTarget = request // request 已是非 nil 指针，直接传给 ShouldBind
		} else {
			bindTarget = &request // struct 取地址传给 ShouldBind
		}
		if err := c.ShouldBind(bindTarget); err != nil {
			response.Fail(c, err, -1)
			return
		}
		if resp, err := fn(c.Request.Context(), request); err != nil {
			response.Fail(c, err, -2)
		} else {
			response.Success(c, resp)
		}
	}
}
