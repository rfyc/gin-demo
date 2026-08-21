// config_test.go 是 conf.Load 的单元测试:
// 覆盖正常解析、空路径与非法配置文件的异常场景。
package conf

import (
	"os"
	"path/filepath"
	"testing"

	"gin-demo/src/utils"

	"github.com/stretchr/testify/assert"
)

// TestLoad 覆盖正常场景: 合法 yaml 配置文件被正确解析。
func TestLoad(t *testing.T) {

	// case: 合法 yaml 配置, 应被正确解析
	path := filepath.Join(t.TempDir(), "conf.yaml")
	content := "Env: LOCAL\nServer:\n  mode: debug\n  addr: \":8080\"\n"
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := Load(path)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "LOCAL", cfg.Env)
	assert.Equal(t, "debug", cfg.Server.Mode)
	assert.Equal(t, ":8080", cfg.Server.Addr)
}

// TestLoadEmptyPath 覆盖异常场景: 路径为空时返回错误。
func TestLoadEmptyPath(t *testing.T) {

	// case: 路径为空, 应返回错误
	cfg, err := Load("")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TestLoadInvalidFile 覆盖异常场景: 配置文件内容非法时返回错误。
func TestLoadInvalidFile(t *testing.T) {

	// case: 配置文件内容非法, 应返回错误
	path := filepath.Join(t.TempDir(), "conf.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("invalid: [yaml"), 0o644))

	_, err := Load(path)
	assert.Error(t, err)
}

// TestLoadMissingFile 覆盖异常场景: 配置文件不存在时返回错误。
func TestLoadMissingFile(t *testing.T) {

	// case: 配置文件不存在, 应返回错误
	path := filepath.Join(t.TempDir(), "not-exist.yaml")

	_, err := Load(path)
	assert.Error(t, err)
}

// TestLoadFromProjectConfig 覆盖正常场景: 相对项目根的配置路径经 FindConfigFile
// 解析为绝对路径后能被正确加载(集成验证 FindConfigFile 与 Load 的链路)。
func TestLoadFromProjectConfig(t *testing.T) {

	// case: 相对项目根的配置路径经 FindConfigFile 解析后可被正确加载
	// 相对项目根(go.mod 所在目录)的配置路径
	relPath := "config/local/conf.yaml"

	absPath := utils.FindConfigFile(relPath)
	assert.NotEmpty(t, absPath, "相对项目根的配置路径应能解析成功")
	assert.True(t, filepath.IsAbs(absPath))

	cfg, err := Load(absPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "LOCAL", cfg.Env)
	assert.Equal(t, ":8080", cfg.Server.Addr)
}
