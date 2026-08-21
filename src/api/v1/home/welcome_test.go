package home

import (
	"testing"

	"gin-demo/src/utils"

	"github.com/stretchr/testify/assert"
)

// TestWelcome 表驱动覆盖正常与边界场景: 无论请求参数为何, 均返回固定欢迎语。
func TestWelcome(t *testing.T) {

	tests := []struct {
		name    string
		request *WelcomeRequest
		want    *WelcomeResponse
		wantErr error
	}{
		{"正常: 带ID", &WelcomeRequest{ID: 123}, &WelcomeResponse{Message: "welcome"}, nil},
		{"边界: 零值", &WelcomeRequest{}, &WelcomeResponse{Message: "welcome"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var resp, err = Welcome(utils.NewTestContext(), tt.request)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, resp)
		})
	}
}
