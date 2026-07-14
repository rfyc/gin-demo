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

// Config 是整个应用的配置根结构，与 INI 节一一对应。
type Config struct {
	Env    string           `mapstructure:"Env"`
	Server ServerCfg        `mapstructure:"Server"`
	Log    logger.LogConfig `mapstructure:"Log"`
	MySQL  DBCfg            `mapstructure:"Mysql"`
	Redis  RedisCfg         `mapstructure:"Redis"`
}

type OpenApiCfg struct {
	ApiID  string `mapstructure:"apiID"`
	ApiUrl string `mapstructure:"apiUrl"`
	ApiKey string `mapstructure:"apiKey"`
}

type ServerCfg struct {
	Mode         string        `mapstructure:"mode"`
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`
	Grace        bool          `mapstructure:"grace"`
}

type DBCfg struct {
	Reader string `mapstructure:"reader"` // DSN 格式：user:pass@tcp(host:port)/db?params
	Writer string `mapstructure:"writer"`
}

type RedisCfg struct {
	Addr        string `mapstructure:"addr"`
	Password    string `mapstructure:"password"`
	IdleTimeout int    `mapstructure:"idletimeout"` // 秒
	PoolSize    int    `mapstructure:"poolsize"`
	DB          int    `mapstructure:"db"`
}

// Load 加载配置文件
func Load(configFile string) (cfg *Config, err error) {
	if configFile == "" {
		return nil, fmt.Errorf("CFG_PATH 环境变量未设置且未传入配置文件路径")
	}
	if !filepath.IsAbs(configFile) {
		if root, e := findProjectRoot(); e == nil {
			configFile = filepath.Join(root, configFile)
		}
	}
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType(strings.TrimPrefix(filepath.Ext(configFile), "."))
	if err = v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err = v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return cfg, nil
}

// findProjectRoot 从当前工作目录向上递归查找 go.mod 所在目录作为项目根。
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
