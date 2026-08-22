package utils

import (
	"context"
	"fmt"
	"time"
)

// detachedCtx 保留父 ctx 的所有 values，但屏蔽其 Done/Deadline/Err（去掉生命周期）
type detachedCtx struct{ context.Context }

func (detachedCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (detachedCtx) Done() <-chan struct{}       { return nil }
func (detachedCtx) Err() error                  { return nil }

func CopyContext(ctx context.Context) context.Context {
	return detachedCtx{ctx}
}

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

type TimeSince struct {
	Begin time.Time `json:"begin"`
	since []timeSince
}

func (t *TimeSince) Add(title string) {
	t.since = append(t.since, timeSince{Title: title, RunTime: time.Now()})
}

func (t *TimeSince) String() string {

	var (
		out  string    // out 拼接后的耗时明细字符串
		last time.Time // last 上一阶段的完成时间, 用于计算阶段间隔
	)

	last = t.Begin
	for _, since := range t.since {
		out += fmt.Sprintf("[%s:%s]", since.Title, since.RunTime.Sub(last).String())
		last = since.RunTime
	}
	return out
}

type timeSince struct {
	Title   string    `json:"title"`
	RunTime time.Time `json:"run_time"`
}
