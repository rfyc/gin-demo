package router

import (
	"gin-demo/src/api/router/middleware"
	"gin-demo/src/api/v1/home"
	"gin-demo/src/core"
	"gin-demo/src/core/conf"

	"github.com/gin-gonic/gin"
)

// New 构建 gin 引擎, 注册全局中间件与业务路由。
//
// 参数:
//   - cfg: HTTP 服务配置, 用于设置 gin 运行模式
//
// 返回:
//   - *gin.Engine: 已注册中间件与路由的 gin 引擎
func New(cfg conf.ServerCfg) *gin.Engine {
	var (
		r  = gin.New()      // gin 引擎实例
		v1 *gin.RouterGroup // /api/v1 业务路由分组
	)

	gin.SetMode(cfg.Mode)

	// 1. 注册全局中间件: 异常恢复、访问日志、跨域、链路追踪
	{
		r.Use(middleware.Recover())
		r.Use(middleware.Logger())
		r.Use(middleware.CORS())
		r.Use(middleware.Trace())
	}

	// 2. 注册 /api/v1 业务路由
	{
		v1 = r.Group("/api/v1")
		v1.GET("/home/hello", core.HandleFunc(home.Hello))
		v1.GET("/home/welcome", core.HandleFunc(home.Welcome))
	}

	return r
}
