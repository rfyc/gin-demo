package router

import (
	"gin-demo/src/api/router/middleware"
	"gin-demo/src/api/v1/home"
	"gin-demo/src/core"
	"gin-demo/src/core/conf"

	"github.com/gin-gonic/gin"
)

// New 创建并返回配置好的 *gin.Engine。
func New(cfg conf.ServerCfg) *gin.Engine {

	gin.SetMode(cfg.Mode)

	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/home/hello", core.HandleFunc(home.Hello))
		v1.GET("/home/welcome", core.HandleFunc(home.Welcome))

	}

	return r
}
