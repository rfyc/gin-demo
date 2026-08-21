// hello_test.go 是 Hello 业务函数的单元测试:
// 覆盖正常与边界场景, 验证返回问候语。
package home

import (
	"testing"

	"gin-demo/src/utils"

	"github.com/stretchr/testify/assert"
)

// TestHello 表驱动覆盖正常与边界场景: 无论请求参数为何, 均返回固定问候语。
func TestHello(t *testing.T) {

	tests := []struct {
		name    string
		request HelloRequest
		want    HelloResponse
		wantErr error
	}{
		{"正常: 带姓名", HelloRequest{Name: "alice"}, HelloResponse{Message: "hello"}, nil},
		{"边界: 空姓名", HelloRequest{}, HelloResponse{Message: "hello"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			resp, err := Hello(utils.NewTestContext(), tt.request)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, resp)
		})
	}
}
