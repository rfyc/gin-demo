package test

import (
	"fmt"
	"gin-demo/src/core/conf"
	"gin-demo/src/utils"
	"os"
	"testing"
)

func TestVerifyConf(t *testing.T) {

	// 模拟用户的相对路径
	relPath := "../../config/local/conf.yaml"
	fmt.Printf("[1] 原始参数: %s\n", relPath)
	fmt.Printf("[2] IsExist(原始路径): %v\n", utils.IsExist(relPath))

	absPath := utils.FindConfigFile(relPath)
	fmt.Printf("[3] FindConfigFile 结果: %q\n", absPath)

	if absPath == "" {
		fmt.Println("[FAIL] 配置文件路径为空")
		os.Exit(1)
	}

	cfg, err := conf.Load(absPath)
	if err != nil {
		fmt.Printf("[FAIL] conf.Load: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[OK] 配置加载成功, Env=%s, Server.Addr=%s\n", cfg.Env, cfg.Server.Addr)
}
