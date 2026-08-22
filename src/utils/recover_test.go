// recover_test.go 是 Recover 的单元测试:
// 验证 goroutine 内 panic 被捕获后不会导致进程崩溃, 且无 panic 时不阻塞。
package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRecoverPanic 覆盖异常场景: 捕获 panic 后 goroutine 正常退出。
func TestRecoverPanic(t *testing.T) {

	// case: goroutine 内 panic, 应被捕获且正常退出不崩溃
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer Recover(context.Background())
		panic("boom")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "Recover 未捕获 panic, goroutine 超时未退出")
	}
}

// TestRecoverNoPanic 覆盖正常场景: 无 panic 时 Recover 不阻塞 goroutine 退出。
func TestRecoverNoPanic(t *testing.T) {

	// case: 无 panic 时, Recover 不应阻塞 goroutine 退出
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer Recover(context.Background())
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "无 panic 时 Recover 不应阻塞 goroutine 退出")
	}
}
