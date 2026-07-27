// Package conf 应用配置管理模块，负责配置文件的加载与解析
package conf

import (
	"fmt"
	"gin-demo/src/pkg/logger"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// 运行环境常量定义
const (
	LOCAL  = "LOCAL"  // 本地开发环境
	TEST   = "TEST"   // 测试环境
	GRAY   = "GRAY"   // 灰度环境
	ONLINE = "ONLINE" // 线上生产环境
)

// Config 是整个应用的配置根结构，与配置文件中的节一一对应。
type Config struct {
	Env    string           `mapstructure:"Env"`    // 当前运行环境，取值参见 LOCAL/TEST/GRAY/ONLINE 常量
	Server ServerCfg        `mapstructure:"Server"` // HTTP 服务相关配置
	Log    logger.LogConfig `mapstructure:"Log"`    // 日志相关配置（复用 logger 包的配置结构）
	MySQL  DBCfg            `mapstructure:"Mysql"`  // MySQL 数据库读写分离配置
	Redis  RedisCfg         `mapstructure:"Redis"`  // Redis 缓存配置
}

// IsLocal 判断当前是否为本地开发环境
func (c *Config) IsLocal() bool {
	return c.Env == LOCAL
}

// IsTest 判断当前是否为测试环境
func (c *Config) IsTest() bool {
	return c.Env == TEST
}

// IsGray 判断当前是否为灰度环境
func (c *Config) IsGray() bool {
	return c.Env == GRAY
}

// IsOnline 判断当前是否为线上生产环境
func (c *Config) IsOnline() bool {
	return c.Env == ONLINE
}

// ServerCfg HTTP 服务配置
type ServerCfg struct {
	Mode         string        `mapstructure:"mode"`         // Gin 运行模式：debug/release/test
	Addr         string        `mapstructure:"addr"`         // 服务监听地址，格式如 ":8080"
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`  // 读取请求的超时时间
	WriteTimeout time.Duration `mapstructure:"writeTimeout"` // 写入响应的超时时间
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`  // Keep-Alive 连接的空闲超时时间
	Grace        bool          `mapstructure:"grace"`        // 是否启用优雅停机
}

// DBCfg MySQL 数据库读写分离配置
type DBCfg struct {
	Reader string `mapstructure:"reader"` // 读库 DSN，格式：user:pass@tcp(host:port)/db?params
	Writer string `mapstructure:"writer"` // 写库 DSN，格式同上
}

// RedisCfg Redis 缓存配置
type RedisCfg struct {
	Addr        string `mapstructure:"addr"`        // Redis 地址，格式：host:port
	Password    string `mapstructure:"password"`    // Redis 认证密码
	IdleTimeout int    `mapstructure:"idletimeout"` // 连接空闲超时时间，单位：秒
	PoolSize    int    `mapstructure:"poolsize"`    // 连接池最大连接数
	DB          int    `mapstructure:"db"`          // Redis 数据库编号（0-15）
}

// Load 从指定路径加载配置文件并解析为 Config 结构体。
// 若 configFile 为相对路径，会自动向上查找项目根（go.mod 所在目录）并拼接完整路径。
// 支持 viper 兼容的所有配置格式（yaml/json/toml/ini 等），格式由文件扩展名自动推断。
func Load(configFile string) (cfg *Config, err error) {
	// 参数校验：配置文件路径不能为空
	if configFile == "" {
		return nil, fmt.Errorf("CFG_PATH 环境变量未设置且未传入配置文件路径")
	}
	// 相对路径处理：尝试以项目根目录为基准补全路径
	if !filepath.IsAbs(configFile) {
		if root, e := findProjectRoot(); e == nil {
			configFile = filepath.Join(root, configFile)
		}
	}
	// 创建独立的 viper 实例，避免与全局实例冲突
	v := viper.New()
	v.SetConfigFile(configFile)
	// 根据文件扩展名自动识别配置格式（去除扩展名前的点号）
	v.SetConfigType(strings.TrimPrefix(filepath.Ext(configFile), "."))
	// 读取配置文件内容
	if err = v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	// 将配置内容反序列化为 Config 结构体
	if err = v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return cfg, nil
}

// findProjectRoot 从当前工作目录开始向上递归查找 go.mod 文件所在目录，将其视为项目根目录。
// 若遍历到文件系统根目录仍未找到 go.mod，则返回错误。
func findProjectRoot() (string, error) {
	// 获取当前工作目录
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// 逐层向上查找 go.mod
	for {
		// 检查当前目录下是否存在 go.mod 文件
		if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		// 获取上级目录
		parent := filepath.Dir(dir)
		// 若上级目录等于当前目录，说明已到达文件系统根目录，查找失败
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
