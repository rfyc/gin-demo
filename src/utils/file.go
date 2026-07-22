package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"golang.org/x/net/publicsuffix"
)

// 返回文件扩展名 例: .png
func FileDotExt(fileUrl string) (ext string) {
	return "." + FileExt(fileUrl)
}

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

// HasMainDomain 检查给定的HTTP URI是否属于指定的主要域名。
// 这个函数首先验证URL的格式，然后解析其域名部分，并与提供的主要域名进行比较。
// 它使用公共后缀列表来确定域名的有效顶级域名部分。
// 参数:
//
//	httpURI: 需要检查的HTTP URI字符串。
//	mainDomain: 需要比较的主要域名字符串。
//
// 返回值:
//
//	如果HTTP URI属于指定的主要域名，则返回true；否则返回false。
func HasMainDomain(httpURI, mainDomain string) bool {
	// 首先检查URL是否符合HTTP URI的基本格式，不符合则直接返回false。
	if !IsHttpURI(httpURI) {
		return false
	}
	// 尝试解析URL，如果解析过程中出现错误，则返回false。
	if parsedUrl, err := url.Parse(httpURI); err != nil {
		return false
	} else {
		// 获取解析后的域名。
		var domain = parsedUrl.Hostname()
		// 使用公共后缀列表来获取域名的有效的主域名部分。
		if eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(domain); err != nil {
			return false
		} else {
			// 比较确定的顶级域名与提供的域名参数，忽略大小写。
			return strings.ToLower(eTLDPlusOne) == strings.ToLower(mainDomain)
		}
	}
}

func HasMainDomains(httpURI string, domains []string) bool {
	if parsedUrl, err := url.Parse(httpURI); err != nil {
		return false
	} else {
		if mainDomain, _ := publicsuffix.EffectiveTLDPlusOne(parsedUrl.Hostname()); mainDomain != "" {
			for _, domain := range domains {
				if strings.ToLower(mainDomain) == strings.ToLower(domain) {
					return true
				}
			}
		}
		return false
	}
}

func HasDomains(httpURI string, domains []string) bool {
	if parsedUrl, err := url.Parse(httpURI); err != nil {
		return false
	} else {
		// 获取解析后的域名。
		var httpDomain = parsedUrl.Hostname()
		for _, domain := range domains {
			if strings.ToLower(domain) == strings.ToLower(httpDomain) {
				return true
			}
		}
		return false
	}
}

// IsHttpURI 检查给定的字符串是否是有效的HTTP或HTTPS URI。
// 该函数首先尝试解析URI以确保它具有基本的URI结构。
// 然后，它进一步检查URI的方案是否为"http"或"https"，并且具有非空的主机名。
// 返回值为布尔类型，如果给定的字符串是有效的HTTP或HTTPS URI，则返回true，否则返回false。
func IsHttpURI(httpURI string) bool {
	if _, err := url.ParseRequestURI(httpURI); err != nil {
		return false
	}
	if parsedURL, err := url.Parse(httpURI); err != nil {
		return false
	} else {
		return parsedURL.Host != "" || parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
	}
}

func IsHttpsURI(httpsURI string) bool {
	if _, err := url.ParseRequestURI(httpsURI); err != nil {
		return false
	}
	if parsedURL, err := url.Parse(httpsURI); err != nil {
		return false
	} else {
		return parsedURL.Scheme == "https"
	}
}

// ReadFileUrl 从给定的URL读取文件内容。
// 参数:
//
//	url: 文件的URL地址。
//
// 返回值:
//
//	[]byte: 文件内容的字节切片。
//	error: 如果读取过程中发生错误，返回错误信息。
func ReadFileUrl(url string, timeout ...int) ([]byte, error) {
	// 检查URL是否有效。
	if !IsHttpURI(url) {
		return nil, errors.New("invalid url")
	}
	// 发起HTTP GET请求获取URL对应的内容。
	var client = &http.Client{
		Timeout: time.Duration(append(timeout, 60)[0]) * time.Second,
	}
	if resp, err := client.Get(url); err != nil {
		// 如果请求失败，返回错误信息。
		return nil, fmt.Errorf("HTTP request fail: %w", err)
	} else {
		// 确保在函数返回前关闭响应体。
		defer resp.Body.Close()
		// 检查HTTP响应状态码是否表示成功。
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("http status fail: %d", resp.StatusCode)
		}
		// 读取响应体的全部内容。
		if body, err := io.ReadAll(resp.Body); err != nil {
			// 如果读取失败，返回错误信息。
			return nil, fmt.Errorf("read body fail: %w", err)
		} else {
			// 返回读取到的内容。
			return body, nil
		}
	}
}

func FileName(url string) (fileName string) {
	url = strings.Split(url, "?")[0]
	return url[strings.LastIndex(url, "/")+1:]
}

// 返回小写的文件 a/b/c.jpg 返回 jpg
func FileExt(url string) (extension string) {
	return strings.ReplaceAll(strings.ToLower(filepath.Ext(FileName(url))), ".", "")
}

func DownloadFile(url, filePath string) (err error) {
	var resp []byte
	var file *os.File
	if resp, err = ReadFileUrl(url); err != nil {
		return
	}
	if file, err = CreateFile(filePath); err != nil {
		return err
	}
	if _, err = io.Copy(file, bytes.NewReader(resp)); err != nil {
		return fmt.Errorf("file write fail: %w", err)
	}
	return
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

func FileCount(dir string, pattern string) (count int) {
	if files, _ := GlobFiles(dir, pattern); len(files) > 0 {
		return len(files)
	}
	return
}

// 返回文件名称  a/b/c.jpg 返回 c
func FileBase(filePath string) (base string) {
	base = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	return
}

func FileUrlNoQuery(rawURL string) (string, error) {
	// 解析 URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// 去掉查询部分
	parsedURL.RawQuery = ""
	// 返回没有查询参数的 URL
	return parsedURL.String(), nil
}

func FileUrlBaseName(rawURL string) (string, error) {
	// 解析 URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// 获取路径部分并提取文件名
	return path.Base(parsedURL.Path), nil
}

func ExecCommand(commandStrs string) (err error, stdout, stderr string) {
	for _, commandStr := range strings.Split(commandStrs, "&&") {
		command, err := shellwords.Parse(commandStr)
		if err != nil {
			return fmt.Errorf("解析命令失败: %v", err), "", ""
		}
		if len(command) == 0 {
			return errors.New("命令数组不能为空"), "", ""
		}

		cmd := exec.Command(command[0], command[1:]...)
		var cmdOut, cmdErr bytes.Buffer
		cmd.Stdout = &cmdOut
		cmd.Stderr = &cmdErr

		if err = cmd.Run(); err != nil {
			errMsg := fmt.Sprintf(
				"命令执行失败: %v - [命令: %s] - [错误输出: %s]",
				err,
				strings.Join(command, " "),
				strings.ReplaceAll(cmdErr.String(), "\n", "    ----    "),
			)
			return errors.New(errMsg), cmdOut.String(), cmdErr.String()
		}
		stdout += cmdOut.String()
		stderr += cmdErr.String()
	}
	return nil, stdout, stderr
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

// findUpward 从当前目录向上查找指定的相对路径
// 返回找到的路径，如果未找到返回空字符串
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
