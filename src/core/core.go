// Package core 负责服务全局依赖的初始化：配置、日志、数据库、缓存。
// 通过 init() 在包导入时自动加载，并把全局单例（Conf/Logger/DB/Cache）
// 暴露给各业务模块直接引用。
package core

import (
	"fmt"
	"gin-demo/src/core/cache"
	"gin-demo/src/core/conf"
	"gin-demo/src/core/db"
	"gin-demo/src/pkg/logger"
	"gin-demo/src/utils"
	"os"
	"strings"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	flag "github.com/spf13/pflag"
)

var (
	// Conf 全局配置对象，由 conf.Load 加载，供各模块读取配置项。
	Conf *conf.Config
	// Logger 全局日志对象，封装 zap，支持从 context 中读取 request_id。
	Logger logger.ILogger
	// DB 全局数据库句柄，封装 GORM，区分读写库，支持事务上下文透传。
	DB *db.DB
	// Cache 全局 go-redis 客户端，本地环境自动降级为 miniredis。
	Cache *redis.Client
	// mockRedisSrv 本地 mock 模式下的 miniredis 服务句柄，供 Cleanup 释放；真实模式为 nil。
	mockRedisSrv *miniredis.Miniredis
)

// init 在包被导入时自动执行，按以下顺序完成启动期初始化：
//  1. 解析命令行参数 --conf，确定配置文件路径（失败则回退到环境变量/默认路径）；
//  2. 加载配置，失败直接 panic 终止进程；
//  3. 初始化数据库：本地环境且未配置 MySQL 时降级为 SQLite mock，否则连接 MySQL；
//  4. 初始化缓存：本地环境且未配置 Redis 时降级为 miniredis mock，否则连接真实 Redis；
//  5. 初始化日志。
//
// 注意：init() 无法返回 error，因此初始化失败只能通过 panic 暴露；
// 且 Logger 在最后才初始化，DB/Cache 初始化失败时无法借助 Logger 记录错误细节。
func init() {
	var (
		err        error  // 初始化过程中的错误
		configFile string // 配置文件路径
	)

	// 1. 解析 --conf 命令行参数, 确定配置文件路径
	{
		// 1.1. 解析 --conf 参数, 允许外部显式指定配置文件路径
		flag.StringVar(&configFile, "conf", "", "config file path")
		flag.Parse()

		// 1.2. 未通过命令行指定时, 依次回退: CFG_CONFIG 环境变量 -> CFG_PATH 环境变量 -> 默认路径
		if configFile == "" {
			if envConfig := os.Getenv("CFG_CONFIG"); envConfig != "" {
				configFile = envConfig
			} else if envPath := os.Getenv("CFG_PATH"); envPath != "" {
				configFile = fmt.Sprintf("./config/%s/local/conf.yaml", envPath)
			} else {
				configFile = "./config/local/conf.yaml"
			}
		}

		// 1.3. 将可能为相对路径的配置文件向上查找, 解析为可用的绝对路径
		configFile = utils.FindConfigFile(configFile)
	}

	// 2. 加载并解析配置文件, 失败直接 panic 终止进程
	{
		if Conf, err = conf.Load(configFile); err != nil {
			panic(fmt.Errorf("conf.Load FAIL: %w", err))
		}
		logger.Printf("Env: %s", Conf.Env)
		logger.Printf("Conf: %s", configFile)
	}

	// 3. 初始化数据库: 本地环境且未配置 MySQL DSN 时降级为 SQLite mock, 否则连接真实 MySQL
	{
		// 3.1. 本地 + 无 DSN: 使用 SQLite mock, 便于本地开发调试
		if Conf.IsLocal() && Conf.MySQL.Reader == "" && Conf.MySQL.Writer == "" {
			if DB, err = db.NewMockDB(); err != nil {
				panic(fmt.Errorf("db.NewMockDB FAIL: %w", err))
			}
			if mockPath, pathErr := db.GetMockPath(); pathErr == nil {
				logger.Printf("MySQL: Mock SQLite (%s)", mockPath)
			}
		} else {
			// 3.2. 否则连接真实 MySQL 读写库
			if DB, err = db.NewDB(&Conf.MySQL); err != nil {
				panic(fmt.Errorf("db.NewDB FAIL: %w", err))
			}
			logger.Printf("MySQL: reader=%s writer=%s", redactDSN(Conf.MySQL.Reader), redactDSN(Conf.MySQL.Writer))
		}
	}

	// 4. 初始化缓存: 本地环境且未配置 Redis 地址时降级为 miniredis mock, 否则连接真实 Redis
	{
		// 4.1. 本地 + 无地址: 使用 miniredis mock
		if Conf.IsLocal() && Conf.Redis.Addr == "" {
			if mockRedisSrv, err = cache.MockSrvRun(&Conf.Redis); err != nil {
				panic(fmt.Errorf("cache.MockSrvRun FAIL: %w", err))
			}
			// 用 miniredis 地址构造客户端配置, 指向本地 mock
			Conf.Redis.Addr = mockRedisSrv.Addr()
			if Cache, err = cache.NewRedisCli(&Conf.Redis); err != nil {
				mockRedisSrv.Close()
				panic(fmt.Errorf("cache.NewRedisCli(mock) FAIL: %w", err))
			}
			logger.Printf("Redis: Mock MiniRedis (%s)", Conf.Redis.Addr)
		} else {
			// 4.2. 否则连接真实 Redis
			if Cache, err = cache.NewRedisCli(&Conf.Redis); err != nil {
				panic(fmt.Errorf("cache.NewRedisCli FAIL: %w", err))
			}
			logger.Printf("Redis: addr=%s db=%d poolsize=%d", Conf.Redis.Addr, Conf.Redis.DB, Conf.Redis.PoolSize)
		}
	}

	// 5. 初始化日志
	Logger = logger.NewLogger(Conf.Log)
}

// redactDSN 脱敏 MySQL DSN 中的账号密码, 仅保留连接地址与库名部分(如 tcp(host:port)/db)。
func redactDSN(dsn string) string {

	if i := strings.Index(dsn, "@"); i >= 0 {
		return dsn[i+1:]
	}
	return dsn
}

// Cleanup 释放全局资源，应在进程退出前（通常以 defer 方式）调用：
//   - 关闭数据库连接（含读写库）；
//   - 关闭 Redis 客户端及其 miniredis（mock 模式）。
//
// 若 init 中途 panic，本函数不会被调用，资源由进程回收兜底。
func Cleanup() {

	if DB != nil {
		DB.Close()
	}
	if Cache != nil {
		cache.Close(Cache)
	}
	if mockRedisSrv != nil {
		mockRedisSrv.Close()
	}
}
