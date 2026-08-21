// file_test.go 是 utils 文件操作函数的单元测试:
// 覆盖 CreateFile / IsExist / GetDirPath / FileName / FileExt / IsFile /
// GlobFiles / WriteToFile / FindConfigFile / FindProjectRoot 的正常、边界与异常场景。
package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateFile 覆盖基本场景: 文件与父目录不存在时, 创建成功且路径存在。
func TestCreateFile(t *testing.T) {

	// case: 文件与父目录均不存在, 应创建成功且路径存在
	path := filepath.Join(t.TempDir(), "sub", "a.txt")

	f, err := CreateFile(path)
	assert.NoError(t, err)
	assert.NotNil(t, f)
	assert.NoError(t, f.Close())
	assert.True(t, IsExist(path))
}

// TestCreateFileOverwrite 覆盖覆盖写场景: 文件已存在时先删除后重建, 不报错。
func TestCreateFileOverwrite(t *testing.T) {

	// case: 文件已存在, 应先删除后重建(覆盖写)
	path := filepath.Join(t.TempDir(), "a.txt")
	assert.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	f, err := CreateFile(path)
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "", string(data)) // 覆盖后为空文件
}

// TestIsExist 覆盖基本与异常场景: 目录存在返回 true, 不存在的路径返回 false。
func TestIsExist(t *testing.T) {

	// case: 目录存在返回 true, 不存在的路径返回 false
	dir := t.TempDir()
	assert.True(t, IsExist(dir))
	assert.False(t, IsExist(filepath.Join(dir, "not-exist")))
}

// TestGetDirPath 覆盖正常与边界场景: 含分隔符取父目录, 无分隔符返回当前目录。
func TestGetDirPath(t *testing.T) {

	// case: 含分隔符取父目录, 无分隔符返回当前目录(不越界)
	assert.Equal(t, "/a/b", GetDirPath("/a/b/c.txt"))
	assert.Equal(t, ".", GetDirPath("a.txt"))
}

// TestFileName 覆盖正常与边界场景: 含查询参数时仅取文件名部分。
func TestFileName(t *testing.T) {

	// case: 含查询参数时仅取文件名部分
	assert.Equal(t, "b.jpg", FileName("http://x/a/b.jpg"))
	assert.Equal(t, "b.jpg", FileName("http://x/a/b.jpg?size=1"))
	assert.Equal(t, "a.txt", FileName("a.txt"))
}

// TestFileExt 覆盖正常场景: 后缀统一转为小写且不含点号。
func TestFileExt(t *testing.T) {

	// case: 后缀统一转小写且不含点号
	assert.Equal(t, "jpg", FileExt("http://x/a/b.JPG?size=1"))
	assert.Equal(t, "jpg", FileExt("a/b/c.jpg"))
}

// TestIsFile 覆盖正常与异常场景: 普通文件为 true, 目录与不存在路径为 false。
func TestIsFile(t *testing.T) {

	// case: 普通文件返回 true, 目录与不存在路径返回 false
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	assert.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	assert.True(t, IsFile(file))
	assert.False(t, IsFile(dir))
	assert.False(t, IsFile(filepath.Join(dir, "nope")))
}

// TestGlobFiles 覆盖正常与异常场景: 匹配文件返回列表, 无匹配返回空列表。
func TestGlobFiles(t *testing.T) {

	// case: 匹配模式返回文件列表, 无匹配返回空列表
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "b.go"), []byte("x"), 0o644))

	files, err := GlobFiles(dir, "*.txt")
	assert.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "a.txt")}, files)

	empty, err := GlobFiles(dir, "*.md")
	assert.NoError(t, err)
	assert.Empty(t, empty)
}

// TestWriteToFile 覆盖正常场景: 父目录不存在时自动创建, 内容写入成功。
func TestWriteToFile(t *testing.T) {

	// case: 父目录不存在时自动创建, 内容写入成功
	path := filepath.Join(t.TempDir(), "deep", "d.txt")

	err := WriteToFile(path, "hello")
	assert.NoError(t, err)

	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

// TestFindConfigFileEmpty 覆盖异常场景: 空路径返回空字符串。
func TestFindConfigFileEmpty(t *testing.T) {

	// case: 空路径返回空字符串
	assert.Equal(t, "", FindConfigFile(""))
}

// TestFindConfigFileAbsolute 覆盖正常场景: 已存在的绝对路径原样返回。
func TestFindConfigFileAbsolute(t *testing.T) {

	// case: 已存在的绝对路径原样返回
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	got := FindConfigFile(path)
	assert.Equal(t, path, got)
}

// TestFindConfigFileRelativeToRoot 覆盖正常场景: 相对项目根的路径解析为项目内存在的路径。
func TestFindConfigFileRelativeToRoot(t *testing.T) {

	// case: 相对项目根的路径解析为项目内存在的绝对路径
	// 依赖项目内固有配置文件存在(go.mod 所在目录下)
	got := FindConfigFile("config/local/conf.yaml")
	assert.NotEmpty(t, got)
	assert.True(t, IsExist(got))
	assert.True(t, filepath.IsAbs(got))
}

// TestFindConfigFileNotFound 覆盖异常场景: 不存在的相对路径返回空字符串。
func TestFindConfigFileNotFound(t *testing.T) {

	// case: 不存在的相对路径返回空字符串
	assert.Equal(t, "", FindConfigFile("not-exist/conf.yaml"))
}

// TestFindProjectRoot 覆盖基本场景: 能从包目录向上找到含 go.mod 的项目根目录。
func TestFindProjectRoot(t *testing.T) {

	// case: 从包目录向上找到含 go.mod 的项目根目录
	root, err := FindProjectRoot()
	assert.NoError(t, err)
	assert.True(t, IsExist(filepath.Join(root, "go.mod")))
}
