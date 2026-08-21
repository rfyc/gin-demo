package utils

import "context"

// NewContext 返回一个空的根上下文, 作为业务调用 ctx 的起点。
func NewContext() context.Context {

	return context.Background()
}

// NewSpanContext 返回链路追踪 span 上下文。
// 当前实现为占位, 后续接入 trace 后可基于 ctx 派生 span 上下文。
func NewSpanContext(ctx context.Context) context.Context {

	return context.Background()
}

// NewTestContext 返回测试用根上下文, 供单元测试构造 ctx 参数。
func NewTestContext() context.Context {

	return context.Background()
}
