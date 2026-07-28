package core

import (
	"fmt"
	"gin-demo/src/core/cache"
	"gin-demo/src/core/conf"
	"gin-demo/src/core/db"
	"gin-demo/src/pkg/logger"
	"gin-demo/src/utils"
	"os"

	flag "github.com/spf13/pflag"
)

var (
	Conf   *conf.Config
	Logger logger.ILogger
	DB     *db.DB
	Cache  *cache.RedisPool
)

func init() {
	var err error
	var configFile string

	flag.StringVar(&configFile, "conf", "", "config file path")
	flag.Parse()

	if configFile == "" {
		if envConfig := os.Getenv("CFG_CONFIG"); envConfig != "" {
			configFile = envConfig
		} else if envPath := os.Getenv("CFG_PATH"); envPath != "" {
			configFile = fmt.Sprintf("./config/%s/local/conf.yaml", envPath)
		} else {
			configFile = "./config/local/conf.yaml"
		}
	}

	configFile = utils.FindConfigFile(configFile)

	if Conf, err = conf.Load(configFile); err != nil {
		panic(fmt.Errorf("conf.Load FAIL: %w", err))
	}

	if Conf.IsLocal() && Conf.MySQL.Reader == "" && Conf.MySQL.Writer == "" {
		if DB, err = db.NewMockDB(); err != nil {
			panic(fmt.Errorf("db.NewMockDB FAIL: %w", err))
		}
	} else {
		if DB, err = db.NewDB(&Conf.MySQL); err != nil {
			panic(fmt.Errorf("db.NewDB FAIL: %w", err))
		}
	}

	if Conf.IsLocal() && Conf.Redis.Addr == "" {
		if Cache, err = cache.NewMockRedisPool(&Conf.Redis); err != nil {
			panic(fmt.Errorf("cache.NewMockRedisPool FAIL: %w", err))
		}
	} else {
		if Cache, err = cache.NewRedisPool(&Conf.Redis); err != nil {
			panic(fmt.Errorf("cache.NewRedisPool FAIL: %w", err))
		}
	}

	Logger = logger.NewLogger(Conf.Log)
}

func Cleanup() {
	if DB != nil {
		DB.Close()
	}
	if Cache != nil {
		Cache.Close()
	}
}
