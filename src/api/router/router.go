package router

import (
	"gin-demo/src/api/router/middleware"
	"gin-demo/src/api/v1/home"
	"gin-demo/src/core"
	"gin-demo/src/core/conf"

	"github.com/gin-gonic/gin"
)

func New(cfg conf.ServerCfg) *gin.Engine {
	gin.SetMode(cfg.Mode)

	r := gin.New()

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
