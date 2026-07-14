package core

import (
	"fmt"
	"gin-demo/src/core/cache"
	"gin-demo/src/core/conf"
	"gin-demo/src/core/db"
	"gin-demo/src/pkg/logger"
)

var (
	Conf   *conf.Config
	Logger logger.ILogger
	DB     *db.DB
	Cache  *cache.RedisPool
)

func Init(configFile string) (err error) {

	// config 初始化
	if Conf, err = conf.Load(configFile); err != nil {
		return fmt.Errorf("conf.Load FAIL: %w", err)
	}
	// db 初始化
	if DB, err = db.NewDB(&Conf.MySQL); err != nil {
		return fmt.Errorf("db.NewDB FAIL: %w", err)
	}

	// cache 初始化
	if Cache, err = cache.NewRedisPool(&Conf.Redis); err != nil {
		return fmt.Errorf("cache.NewRedisPool FAIL: %w", err)
	}

	// logger 初始化
	Logger = logger.NewLogger(Conf.Log)

	return
}

func Cleanup() {
	if DB != nil {
		DB.Close()
	}
	if Cache != nil {
		Cache.Close()
	}
}
