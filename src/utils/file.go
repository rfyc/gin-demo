package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateFile 创建文件，如果文件已存在则覆盖，如果文件不存在则创建，如果路径不存在则创建路径
func CreateFile(filePath string) (file *os.File, err error) {
	// 判断文件是否存在
	if IsExist(filePath) {
		// 删除文件
		if err = os.Remove(filePath); err != nil {
			return nil, fmt.Errorf("file exist remove fail: %w", err)
		}
	}
	// 创建文件以及路径
	if err = os.MkdirAll(GetDirPath(filePath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("mkdir fail: %w", err)
	}
	// 创建文件
	if file, err = os.Create(filePath); err != nil {
		return nil, fmt.Errorf("file create fail: %w", err)
	}
	return
}

// IsExist 判断文件或文件夹是否存在
func IsExist(path string) (isExist bool) {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return
}

// GetDirPath 获取文件路径
func GetDirPath(filePath string) (dirPath string) {
	// 获取文件路径
	return filePath[0:strings.LastIndex(filePath, "/")]
}

func FileName(url string) (fileName string) {
	url = strings.Split(url, "?")[0]
	return url[strings.LastIndex(url, "/")+1:]
}

// 返回小写的文件 a/b/c.jpg 返回 jpg
func FileExt(url string) (extension string) {
	return strings.ReplaceAll(strings.ToLower(filepath.Ext(FileName(url))), ".", "")
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func GlobFiles(dir string, pattern string) (files []string, err error) {
	return filepath.Glob(filepath.Join(dir, pattern))
}

// WriteToFile 将文本写入文件，如果目录不存在则创建
func WriteToFile(filename, content string) error {
	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 创建或打开文件
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}

	// 写入内容
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// FindConfigFile 查找配置文件路径
// 支持多种路径格式：
//   - /config/local/conf.yaml  : 会自动从当前目录向上查找
func FindConfigFile(cmdArg string) string {
	// 如果路径为空，直接返回
	if cmdArg == "" {
		return ""
	}
	// 清理路径格式：去掉开头的 / 或 ./
	cleanPath := cleanPathPrefix(cmdArg)
	// 首先检查原始路径是否存在（处理 ../ 开头的相对路径）
	if IsExist(cmdArg) {
		if abs, err := filepath.Abs(cmdArg); err == nil {
			return abs
		}
		return cmdArg
	}
	// 尝试从当前目录向上查找
	return findUpward(cleanPath)
}

// cleanPathPrefix 清理路径开头的 / 或 ./
func cleanPathPrefix(path string) string {
	// 去掉开头的 /
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	// 去掉开头的 ./
	if strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}
	return path
}

func findUpward(relativePath string) string {
	for i := 0; i <= 5; i++ {
		var searchPath string
		if i == 0 {
			searchPath = relativePath
		} else {
			searchPath = strings.Repeat("../", i) + relativePath
		}
		if IsExist(searchPath) {
			return searchPath
		}
	}
	return ""
}

func FindProjectRoot() (string, error) {
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
