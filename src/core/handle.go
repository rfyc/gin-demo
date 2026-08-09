package core

import (
	"context"
	"gin-demo/src/pkg/response"
	"reflect"

	"github.com/gin-gonic/gin"
)

// IRequest 业务请求类型的泛型约束。
// 目前为空接口，仅用于限制 HandleFunc 的 input 参数可接受的类型，
// 使泛型在编译期具备明确的类型约束边界。
type IRequest interface{}

// IResponse 业务响应类型的泛型约束。
// 目前为空接口，作用同 IRequest，用于限制 HandleFunc 的 output 参数。
type IResponse interface{}

// HandleFunc 将业务处理函数适配为 gin 的 HTTP 处理函数。
//
// 泛型参数:
//   - input:  请求类型，可为 struct 或 struct 指针（如 UserReq 或 *UserReq）
//   - output: 响应类型，最终序列化为接口响应体
//
// 参数:
//   - fn: 业务处理函数，签名 fn(ctx, request) (output, error)；
//     ctx 为 gin 请求携带的 context，用于传递取消信号、链路信息等
//
// 返回:
//   - gin.HandlerFunc：负责请求绑定、调用 fn，并按结果写回统一响应
//
// 错误场景:
//   - 请求体绑定失败：response.Fail(c, err, -1)
//   - fn 返回 error：response.Fail(c, err, -2)
//   - 成功：response.Success(c, resp)
func HandleFunc[input IRequest, output IResponse](fn func(ctx context.Context, request input) (output, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		var request input
		var bindTarget any
		// 必须在泛型函数内操作 input 类型，传入 any 后反射到的是 interface 层而非具体类型
		if ref := reflect.ValueOf(&request).Elem(); ref.Kind() == reflect.Ptr {
			// input 为指针类型：var request 得到的 nil 指针无法绑定，先反射创建一个非 nil 实例
			ref.Set(reflect.New(ref.Type().Elem()))
			bindTarget = request // request 已是非 nil 指针，直接传给 ShouldBind
		} else {
			bindTarget = &request // struct 取地址传给 ShouldBind，绑定结果回填到 request
		}
		// 解析并校验请求体（Query/JSON/Form 等，由 gin 根据 Content-Type 决定）
		if err := c.ShouldBind(bindTarget); err != nil {
			response.Fail(c, err, -1)
			return
		}
		// 调用业务处理函数，按结果写回统一响应
		if resp, err := fn(c.Request.Context(), request); err != nil {
			response.Fail(c, err, -2)
		} else {
			response.Success(c, resp)
		}
	}
}
