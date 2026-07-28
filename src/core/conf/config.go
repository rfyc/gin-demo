package conf

import (
	"fmt"
	"gin-demo/src/pkg/logger"
	"gin-demo/src/utils"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	LOCAL  = "LOCAL"
	TEST   = "TEST"
	GRAY   = "GRAY"
	ONLINE = "ONLINE"
)

type Config struct {
	Env    string           `mapstructure:"Env"`
	Server ServerCfg        `mapstructure:"Server"`
	Log    logger.LogConfig `mapstructure:"Log"`
	MySQL  DBCfg            `mapstructure:"Mysql"`
	Redis  RedisCfg         `mapstructure:"Redis"`
}

func (c *Config) IsLocal() bool {
	return c.Env == LOCAL
}

func (c *Config) IsTest() bool {
	return c.Env == TEST
}

func (c *Config) IsGray() bool {
	return c.Env == GRAY
}

func (c *Config) IsOnline() bool {
	return c.Env == ONLINE
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
	Reader string `mapstructure:"reader"`
	Writer string `mapstructure:"writer"`
}

type RedisCfg struct {
	Addr        string `mapstructure:"addr"`
	Password    string `mapstructure:"password"`
	IdleTimeout int    `mapstructure:"idletimeout"`
	PoolSize    int    `mapstructure:"poolsize"`
	DB          int    `mapstructure:"db"`
}

func Load(configFile string) (cfg *Config, err error) {
	if configFile == "" {
		return nil, fmt.Errorf("CFG_PATH 环境变量未设置且未传入配置文件路径")
	}
	if !filepath.IsAbs(configFile) {
		if root, e := utils.FindProjectRoot(); e == nil {
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
