// context_test.go 是 context 构造函数的单元测试:
// 验证各构造函数返回非 nil 的可用根上下文。
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewContext 覆盖基本场景: 返回非 nil 上下文。
func TestNewContext(t *testing.T) {

	// case: 应返回非 nil 的根上下文
	assert.NotNil(t, NewContext())
}

// TestNewSpanContext 覆盖基本场景: 传入父上下文时返回非 nil 上下文。
func TestNewSpanContext(t *testing.T) {

	// case: 传入父上下文, 应返回非 nil 的 span 上下文
	assert.NotNil(t, NewSpanContext(NewContext()))
}

// TestNewTestContext 覆盖基本场景: 返回非 nil 的测试用上下文。
func TestNewTestContext(t *testing.T) {

	// case: 应返回非 nil 的测试用上下文
	assert.NotNil(t, NewTestContext())
}
