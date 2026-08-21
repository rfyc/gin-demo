package home

import "context"

// Welcome 返回欢迎语, 用于验证服务可用性。
//
// 参数:
//   - ctx: 请求上下文, 透传取消信号
//   - request: 请求参数, 支持 form 与 json 两种绑定方式
//
// 返回:
//   - response: 欢迎响应体
//   - err: 恒为 nil
func Welcome(ctx context.Context, request *WelcomeRequest) (response *WelcomeResponse, err error) {

	return &WelcomeResponse{Message: "welcome"}, nil
}

// WelcomeRequest 是 Welcome 接口的请求参数。
type WelcomeRequest struct {
	ID int `form:"id" json:"id"` // ID 请求携带的 ID, 支持 form 与 json 绑定
}

// WelcomeResponse 是 Welcome 接口的响应体。
type WelcomeResponse struct {
	Message string `json:"message"` // Message 欢迎语内容
}
