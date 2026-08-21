package cache

import (
	"fmt"
	"gin-demo/src/core/conf"

	"github.com/alicebob/miniredis/v2"
)

// MockSrvRun 启动 miniredis 服务，若配置了密码则设置访问校验。
//
// 参数:
//   - cfg: Redis 连接配置, 读取 Password 用于设置 mock 服务的访问校验
//
// 返回:
//   - *miniredis.Miniredis: 启动成功的 mock 服务句柄
//   - err: 启动失败时返回带上下文的错误
//
// 错误场景:
//   - miniredis 启动失败: 返回带函数名的错误
func MockSrvRun(cfg *conf.RedisCfg) (mr *miniredis.Miniredis, err error) {

	// 1. 启动 miniredis 服务
	if mr, err = miniredis.Run(); err != nil {
		return nil, fmt.Errorf("MockSrvRun 启动 miniredis 失败: %w", err)
	}

	// 2. 配置了密码时设置访问校验
	if cfg.Password != "" {
		mr.RequireAuth(cfg.Password)
	}
	return mr, nil
}
