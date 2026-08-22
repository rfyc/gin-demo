package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateFile 创建文件并确保父目录存在。
// 若文件已存在则先删除后重建, 相当于覆盖写。
//
// 参数:
//   - filePath: 目标文件路径
//
// 返回:
//   - file: 创建成功的文件句柄, 调用方负责关闭
//   - err: 删除旧文件、创建目录或创建文件失败时返回带上下文的错误
func CreateFile(filePath string) (file *os.File, err error) {

	// 1. 文件已存在时先删除, 实现覆盖写
	if IsExist(filePath) {
		if err = os.Remove(filePath); err != nil {
			return nil, fmt.Errorf("CreateFile remove existing fail: %w", err)
		}
	}
	// 2. 创建父目录, 不存在时自动创建
	if err = os.MkdirAll(GetDirPath(filePath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("CreateFile mkdir fail: %w", err)
	}
	// 3. 创建文件
	if file, err = os.Create(filePath); err != nil {
		return nil, fmt.Errorf("CreateFile create fail: %w", err)
	}
	return file, nil
}

// IsExist 判断文件或文件夹是否存在。
func IsExist(path string) bool {

	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// GetDirPath 返回文件路径的父目录部分。
// 路径中不含分隔符时返回当前目录 ".", 避免切片越界 panic。
func GetDirPath(filePath string) string {
	var idx int // 最后一个路径分隔符位置

	idx = strings.LastIndex(filePath, "/")
	if idx < 0 {
		return "."
	}
	return filePath[:idx]
}

// FileName 从 URL 或路径中提取文件名, 忽略查询参数。
// 如 "http://x/a/b.jpg?size=1" 返回 "b.jpg"。
func FileName(url string) (fileName string) {

	var path = strings.Split(url, "?")[0] // 去掉查询参数后的路径

	return path[strings.LastIndex(path, "/")+1:]
}

// FileExt 返回小写文件名后缀(不含点号), 如 "a/b/c.jpg" 返回 "jpg"。
func FileExt(url string) (extension string) {

	return strings.ReplaceAll(strings.ToLower(filepath.Ext(FileName(url))), ".", "")
}

// IsFile 判断路径是否为普通文件(非目录)。
// 路径不存在或无法访问时返回 false。
func IsFile(path string) bool {
	var (
		info os.FileInfo // 路径的 Stat 信息
		err  error       // Stat 错误
	)

	if info, err = os.Stat(path); err != nil {
		return false
	}
	return !info.IsDir()
}

// GlobFiles 返回目录下匹配 pattern 的文件列表。
// pattern 支持 filepath.Match 语法, 如 "*.txt"。
//
// 参数:
//   - dir: 目标目录
//   - pattern: 文件名匹配模式
//
// 返回:
//   - files: 匹配的文件路径列表
//   - err: 匹配失败时返回错误
func GlobFiles(dir string, pattern string) (files []string, err error) {

	return filepath.Glob(filepath.Join(dir, pattern))
}

// WriteToFile 将文本写入文件, 目录不存在时自动创建。
// 文件已存在时会被覆盖。
//
// 参数:
//   - filename: 目标文件路径
//   - content: 待写入的文本内容
//
// 返回:
//   - err: 创建目录、创建文件或写入失败时返回带上下文的错误
func WriteToFile(filename, content string) (err error) {

	var file *os.File // 待写入的文件句柄

	// 涉及文件系统写入, 记录耗时与结果日志
	defer func(startTime time.Time) {
		LogErrorInfo(context.Background(), startTime, "WriteToFile", err, "filename", filename)
	}(time.Now())

	// 1. 确保父目录存在
	if err = os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("WriteToFile 创建目录失败: %w", err)
	}

	// 2. 创建文件(已存在则覆盖), 写入完成后关闭句柄
	if file, err = os.Create(filename); err != nil {
		return fmt.Errorf("WriteToFile 创建文件失败: %w", err)
	}
	defer file.Close()

	// 3. 写入内容
	if _, err = file.WriteString(content); err != nil {
		return fmt.Errorf("WriteToFile 写入文件失败: %w", err)
	}
	return nil
}

// FindConfigFile 查找配置文件路径:
//   - 已存在(绝对路径或可从当前目录解析的相对路径)时, 转为绝对路径返回;
//   - 否则视为相对项目根目录的路径, 基于 go.mod 所在目录解析;
//   - 均无法解析时返回空字符串。
//
// 参数:
//   - cmdArg: 配置文件路径(绝对或相对)
//
// 返回:
//   - string: 解析后的可用路径; 未找到时返回空字符串
func FindConfigFile(cmdArg string) string {
	var (
		root      string // 项目根目录(go.mod 所在目录)
		candidate string // 拼接后的候选路径
		err       error  // FindProjectRoot 错误
	)

	if cmdArg == "" {
		return ""
	}
	// 1. 路径已存在时转为绝对路径返回(处理绝对路径与含 ../ 的相对路径)
	if IsExist(cmdArg) {
		if abs, err := filepath.Abs(cmdArg); err == nil {
			return abs
		}
		return cmdArg
	}
	// 2. 基于项目根目录(go.mod 所在目录)解析相对路径
	{
		if root, err = FindProjectRoot(); err != nil {
			return ""
		}
		candidate = filepath.Join(root, cleanPathPrefix(cmdArg))
		if IsExist(candidate) {
			return candidate
		}
		return ""
	}
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

// FindProjectRoot 从当前工作目录向上查找项目根目录(含 go.mod 的目录)。
//
// 返回:
//   - string: 项目根目录的绝对路径
//   - err: 未找到 go.mod 时返回错误
func FindProjectRoot() (dir string, err error) {

	var parent string // parent 上一级目录路径, 用于逐级上溯

	// 1. 获取当前工作目录, 从它开始向上查找
	if dir, err = os.Getwd(); err != nil {
		return "", err
	}

	// 2. 逐级向上查找含 go.mod 的目录
	for {
		if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent = filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("FindProjectRoot go.mod not found")
		}
		dir = parent
	}
}
