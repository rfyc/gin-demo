// Package conf 负责配置文件加载与解析：
// 读取 YAML 配置文件并映射为 Config 结构，供 core 初始化各模块使用；
// 同时提供 Env 环境判断方法（本地/测试/灰度/线上）。
package conf

import (
	"fmt"
	"gin-demo/src/pkg/llm"
	"gin-demo/src/pkg/logger"
	"gin-demo/src/utils"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// 环境类型常量，对应配置文件中 Env 字段的取值。
const (
	LOCAL  = "LOCAL"  // LOCAL 本地开发环境
	TEST   = "TEST"   // TEST 测试环境
	GRAY   = "GRAY"   // GRAY 灰度环境
	ONLINE = "ONLINE" // ONLINE 线上环境
)

// Config 是应用配置的根结构，字段与配置文件的顶层 key 一一对应。
type Config struct {
	Env    string           `mapstructure:"Env"`    // Env 当前环境（LOCAL/TEST/GRAY/ONLINE）
	Server ServerCfg        `mapstructure:"Server"` // Server HTTP 服务配置
	Log    logger.LogConfig `mapstructure:"Log"`    // Log 日志配置
	MySQL  DBCfg            `mapstructure:"Mysql"`  // MySQL 读写库 DSN 配置
	Redis  RedisCfg         `mapstructure:"Redis"`  // Redis 连接配置
	Llm    llm.Conf         `mapstructure:"Llm"`    // Llm 大模型默认配置
}

// IsLocal 判断当前是否为本地开发环境。
func (c *Config) IsLocal() bool {

	return c.Env == LOCAL
}

// IsTest 判断当前是否为测试环境。
func (c *Config) IsTest() bool {

	return c.Env == TEST
}

// IsGray 判断当前是否为灰度环境。
func (c *Config) IsGray() bool {

	return c.Env == GRAY
}

// IsOnline 判断当前是否为线上环境。
func (c *Config) IsOnline() bool {

	return c.Env == ONLINE
}

// ServerCfg 是 HTTP 服务配置。
type ServerCfg struct {
	Mode         string        `mapstructure:"mode"`         // Mode gin 运行模式（debug/release/test）
	Addr         string        `mapstructure:"addr"`         // Addr 监听地址，如 ":8080"
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`  // ReadTimeout 读超时
	WriteTimeout time.Duration `mapstructure:"writeTimeout"` // WriteTimeout 写超时
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`  // IdleTimeout 空闲连接超时
	Grace        bool          `mapstructure:"grace"`        // Grace 是否优雅关闭
}

// DBCfg 是 MySQL 读写分离的 DSN 配置。
type DBCfg struct {
	Reader string `mapstructure:"reader"` // Reader 读库 DSN，为空时表示未配置
	Writer string `mapstructure:"writer"` // Writer 写库 DSN，为空时表示未配置
}

// RedisCfg 是 Redis 连接池配置。
type RedisCfg struct {
	Addr        string `mapstructure:"addr"`        // Addr Redis 地址，如 "127.0.0.1:6379"，为空时表示未配置
	Password    string `mapstructure:"password"`    // Password 认证密码，为空表示无需认证
	IdleTimeout int    `mapstructure:"idletimeout"` // IdleTimeout 空闲连接超时（秒）
	PoolSize    int    `mapstructure:"poolsize"`    // PoolSize 连接池大小
	DB          int    `mapstructure:"db"`          // DB 选中的逻辑库编号
}

// Load 加载并解析配置文件：
//   - configFile 为空时返回错误（调用方应通过 --conf 参数或环境变量传入）；
//   - 相对路径会尝试基于项目根目录（go.mod 所在目录）转换为绝对路径；
//   - 使用 viper 按文件扩展名推断格式（yaml/yml/json 等）并反序列化到 Config。
//
// 读取或解析失败时返回带上下文的错误。
func Load(configFile string) (cfg *Config, err error) {

	var v *viper.Viper // viper 配置解析器实例

	// 1. 校验配置文件路径非空
	if configFile == "" {
		return nil, fmt.Errorf("CFG_PATH 环境变量未设置且未传入配置文件路径")
	}

	// 2. 相对路径转换为基于项目根目录的绝对路径
	if !filepath.IsAbs(configFile) {
		if root, e := utils.FindProjectRoot(); e == nil {
			configFile = filepath.Join(root, configFile)
		}
	}

	// 3. 通过 viper 读取并解析配置文件
	{
		v = viper.New()
		v.SetConfigFile(configFile)
		v.SetConfigType(strings.TrimPrefix(filepath.Ext(configFile), "."))
		if err = v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		if err = v.Unmarshal(&cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	}

	return cfg, nil
}
