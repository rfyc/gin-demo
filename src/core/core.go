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

	// 配置文件路径处理
	var err error
	var configFile string

	// 从命令行参数解析 --conf
	flag.StringVar(&configFile, "conf", "", "config file path")
	flag.Parse()

	// 优先级: --conf > CFG_CONFIG > CFG_PATH > default
	if configFile == "" {
		if envConfig := os.Getenv("CFG_CONFIG"); envConfig != "" {
			configFile = envConfig
		} else if envPath := os.Getenv("CFG_PATH"); envPath != "" {
			configFile = fmt.Sprintf("./config/%s/local/conf.yaml", envPath)
		} else {
			configFile = "./config/local/conf.yaml"
		}
	}

	// 相对路径处理
	configFile = utils.FindConfigFile(configFile)

	// config 初始化
	if Conf, err = conf.Load(configFile); err != nil {
		panic(fmt.Errorf("conf.Load FAIL: %w", err))
	}

	// db 初始化：本地环境且 MySQL 读写 DSN 均为空时使用 mock SQLite，否则连接真实 MySQL
	if Conf.IsLocal() && Conf.MySQL.Reader == "" && Conf.MySQL.Writer == "" {
		// 本地环境 + DSN 为空 → 使用独立的 mock DB 创建函数
		if DB, err = db.NewMockDB(); err != nil {
			panic(fmt.Errorf("db.NewMockDB FAIL: %w", err))
		}
	} else {
		// 非本地环境，或任一 DSN 非空 → 使用真实 MySQL 连接
		if DB, err = db.NewDB(&Conf.MySQL); err != nil {
			panic(fmt.Errorf("db.NewDB FAIL: %w", err))
		}
	}

	// cache 初始化：本地环境且 Redis Addr 为空时使用 miniredis mock，否则连接真实 Redis
	if Conf.IsLocal() && Conf.Redis.Addr == "" {
		// 本地环境 + Addr 为空 → 使用独立的 mock Redis 创建函数
		if Cache, err = cache.NewMockRedisPool(&Conf.Redis); err != nil {
			panic(fmt.Errorf("cache.NewMockRedisPool FAIL: %w", err))
		}
	} else {
		// 非本地环境，或 Addr 非空 → 使用真实 Redis 连接
		if Cache, err = cache.NewRedisPool(&Conf.Redis); err != nil {
			panic(fmt.Errorf("cache.NewRedisPool FAIL: %w", err))
		}
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
