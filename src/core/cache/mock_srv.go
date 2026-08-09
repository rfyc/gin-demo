package cache

import (
	"fmt"
	"gin-demo/src/core/conf"

	"github.com/alicebob/miniredis/v2"
)

// MockSrvRun 启动 miniredis 服务，若配置了密码则设置访问校验。
func MockSrvRun(cfg *conf.RedisCfg) (*miniredis.Miniredis, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("启动 miniredis 失败: %w", err)
	}
	if cfg.Password != "" {
		mr.RequireAuth(cfg.Password)
	}
	return mr, nil
}
