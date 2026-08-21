package home

import "context"

// Hello 返回问候语, 用于验证服务可用性。
//
// 参数:
//   - ctx: 请求上下文, 透传取消信号
//   - request: 请求参数, 含姓名等
//
// 返回:
//   - response: 问候响应体
//   - err: 恒为 nil
func Hello(ctx context.Context, request HelloRequest) (response HelloResponse, err error) {

	return HelloResponse{Message: "hello"}, nil
}

// HelloRequest 是 Hello 接口的请求参数。
type HelloRequest struct {
	Name string `json:"name"` // Name 姓名
}

// HelloResponse 是 Hello 接口的响应体。
type HelloResponse struct {
	Message string `json:"message"` // Message 问候语内容
}
