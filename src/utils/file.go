package utils

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"golang.org/x/net/publicsuffix"
)

// base64格式的图文 写入本地文件
func Base64ImageToLocal(base64Content string, localDir string) (string, error) {
	// 去掉Base64前缀，如 "data:image/png;base64,"
	if strings.Contains(base64Content, ",") {
		base64Content = strings.Split(base64Content, ",")[1]
	}
	// 解码Base64字符串
	imgData, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return "", fmt.Errorf("decode Base64 fail: %w", err)
	}
	// 创建一个Reader，从解码后的字节数组读取数据
	reader := strings.NewReader(string(imgData))
	// 解析图像
	img, format, err := image.Decode(reader)
	if err != nil {
		return "", fmt.Errorf("decode image fail: %w", err)
	}
	// 根据当前时间生成唯一的文件名
	localFile := filepath.Join(localDir, fmt.Sprintf("%v.%s", uuid.NewString(), format))
	// 创建输出文件
	outputFile, err := CreateFile(localFile)
	if err != nil {
		return "", fmt.Errorf("create local file fail: %w", err)
	}
	defer outputFile.Close()
	// 将图像写入文件（根据实际情况选择合适的编码器）
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		err = jpeg.Encode(outputFile, img, nil)
	case "png":
		err = png.Encode(outputFile, img)
	default:
		return "", fmt.Errorf("image format fail: %s", format)
	}
	if err != nil {
		return "", fmt.Errorf("encode image fail: %w", err)
	}
	return filepath.Abs(localFile)
}

// 返回文件扩展名 例: .png
func FileDotExt(fileUrl string) (ext string) {
	return "." + FileExt(fileUrl)
}

// FileUrlToLocalFile 保存远程文件到本地 localFile: tmp/123/****不用带文件后缀 保存到 {getwd}/tmp/123/****.png
func FileUrlToLocalFile(ctx context.Context, fileUrl string, localFile string) (string, error) {
	var ext string
	if path, err := url.Parse(fileUrl); err != nil {
		return "", fmt.Errorf("url.Parse - %s - fail: %w", fileUrl, err)
	} else {
		ext = filepath.Ext(path.Path)
	}
	localFile = localFile + ext
	file, err := CreateFile(localFile)
	if err != nil {
		return "", errors.Wrapf(err, "create.file")
	}
	defer file.Close()
	resp, err := http.Get(fileUrl)
	if err != nil {
		return "", errors.Wrapf(err, "http.get.url")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("failed to download image: status code %d", resp.StatusCode)
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "file.write.body")
	}
	return localFile, nil
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
	if _, err := os.Stat(path); os.IsExist(err) {
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

// CheckURLTypeIsWebpage 综合判断URL类型是否为网页
func CheckURLTypeIsWebpage(urlStr string) bool {
	// 首先根据扩展名快速判断
	ext := strings.ToLower(path.Ext(urlStr))

	// 文档扩展名列表
	docExts := map[string]bool{
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".txt": true, ".rtf": true, ".csv": true,
		".json": true, ".xml": true, ".epub": true, ".mobi": true,
	}

	// 网页扩展名列表
	webExts := map[string]bool{
		".html": true, ".htm": true, ".php": true, ".asp": true, ".aspx": true,
		".jsp": true,
	}

	if docExts[ext] {
		return false
	}

	if webExts[ext] {
		return true
	}

	// 无扩展名，检查URL结构
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	path := parsedURL.Path

	// 如果路径为空或以斜杠结尾，很可能是网页
	if path == "" || strings.HasSuffix(path, "/") {
		return true
	}

	// 包含查询参数，可能是动态网页
	if parsedURL.RawQuery != "" {
		return true
	}

	// 其他情况，默认为文档（保守策略）
	// 或者可以根据业务需求调整为"unknown"或"webpage"
	return false
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

func GetRemoteFileMimeType(fileURL string) (string, error) {
	if mimeType, err := getRemoteFileMimeType(fileURL); err == nil {
		return mimeType, nil
	}
	if ext := FileDotExt(fileURL); ext != "" {
		if mimeType, ok := CommonMimeTypes[ext]; ok {
			return mimeType, nil
		}
	}
	return "", fmt.Errorf("not found file mime type: %s", fileURL)
}

func getRemoteFileMimeType(fileURL string) (string, error) {
	var detectSize = 512
	var resp, err = http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// 只读取前 512 字节，不下载整个文件
	var buf = make([]byte, detectSize)
	var n, err1 = io.ReadFull(resp.Body, buf)
	if err1 != nil && err1 != io.ErrUnexpectedEOF && err1 != io.EOF {
		return "", err1
	}
	return http.DetectContentType(buf[:n]), nil
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

// ZipDirectory 压缩指定目录，返回压缩后的文件名和错误信息
func ZipDirectory(sourceDir string) (string, error) {
	// 创建目标zip文件名（与目录同名，后缀为.zip）
	zipFileName := filepath.Base(sourceDir) + ".zip"

	// 创建zip文件
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		return "", fmt.Errorf("创建zip文件失败: %v", err)
	}
	defer zipFile.Close()

	// 创建zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历目录并压缩文件
	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 创建文件头信息
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 设置文件头中的名称（保持相对路径）
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}
		header.Name = relPath

		// 如果是目录，添加路径分隔符
		if info.IsDir() {
			header.Name += "/"
		} else {
			// 设置压缩方法
			header.Method = zip.Deflate
		}

		// 创建zip文件中的条目
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// 如果出错，删除可能已创建的部分zip文件
		_ = os.Remove(zipFileName)
		return "", fmt.Errorf("压缩过程中出错: %v", err)
	}

	return zipFileName, nil
}

// Unzip 解压zip文件
func Unzip(zipFile, destDir string) error {
	// 打开zip文件
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("open zip file fail: %w", err)
	}
	defer r.Close()

	// 创建目标目录
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("create dest dir fail: %w", err)
	}

	// 遍历zip文件中的所有文件
	for _, f := range r.File {
		// 构建目标文件路径
		targetPath := filepath.Join(destDir, f.Name)

		// 检查文件路径是否安全（防止路径遍历攻击）
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		// 如果是目录，创建目录
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
				return fmt.Errorf("create dir fail: %w", err)
			}
			continue
		}

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return fmt.Errorf("create parent dir fail: %w", err)
		}

		// 打开源文件
		source, err := f.Open()
		if err != nil {
			return fmt.Errorf("open source file fail: %w", err)
		}

		// 创建目标文件
		dest, err := CreateFile(targetPath)
		if err != nil {
			source.Close()
			return fmt.Errorf("create dest file fail: %w", err)
		}

		// 复制文件内容
		if _, err := io.Copy(dest, source); err != nil {
			source.Close()
			dest.Close()
			return fmt.Errorf("copy file content fail: %w", err)
		}

		// 关闭文件
		source.Close()
		dest.Close()
	}

	return nil
}

// GetPDFCover 使用ghostscript获取PDF首页作为封面图
func GetPDFCover(ctx context.Context, inputPath string) (outOssUrl string, err error) {
	if !IsFile(inputPath) {
		return "", fmt.Errorf("input file not exist: %s", inputPath)
	}
	if FileExt(inputPath) != "pdf" {
		return "", fmt.Errorf("input file not pdf: %s", inputPath)
	}

	// 生成临时图片路径
	var outputPath = fmt.Sprintf("/tmp/%s.png", uuid.NewString())

	// 使用ghostscript将PDF首页转换为PNG图片
	cmd := exec.Command("gs", "-dBATCH",
		"-dNOPAUSE",
		"-dNOCACHE",
		"-sDEVICE=png16m", // 使用PNG格式，16m表示24位真彩色
		"-dFirstPage=1",   // 只转换第一页
		"-dLastPage=1",
		"-dGraphicsAlphaBits=4", // 抗锯齿
		"-dTextAlphaBits=4",
		"-r300x300", // 分辨率300DPI
		"-sOutputFile="+outputPath,
		inputPath)

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ghostscript exec FAIL: %w - output: %s", err, string(output))
	}
	return outputPath, nil
}

// GetFilenameWithoutExt 获取不含后缀的文件名
func GetFilenameWithoutExt(filename string) string {
	// 获取不含后缀的文件名
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	nameWithoutExt := base[:len(base)-len(ext)]

	return nameWithoutExt
}

func WgetFileUrl(fileUrl string, echoProcess ...bool) (localFile string, err error) {
	var targetDir = "/tmp/"
	if strings.Contains(fileUrl, "x-oss-process") {
		if urlPath, _ := FileUrlNoQuery(fileUrl); urlPath != "" {
			fileUrl = urlPath
		}
	}

	localFile, _ = FileUrlBaseName(fileUrl)
	localFile = filepath.Join(targetDir, uuid.NewString()+"_"+localFile)
	var command = fmt.Sprintf(`mkdir -p %s && wget --no-use-server-timestamps --timeout=60 --tries=2 -O "%s" "%s"`, targetDir, localFile, fileUrl)
	//if err, _, _ = ExecCommand(command); err != nil {
	//	return "", fmt.Errorf("wget down file FAIL: %w", err)
	//}
	err = ExecCommandWithProgress(
		command,
		func(step int, total int, cmd string, stdoutLine string, stderrLine string) {
			if len(echoProcess) > 0 && echoProcess[0] {
				fmt.Printf("[进度 %d/%d] 命令: %s\n", step, total, cmd)

				if stdoutLine != "" {
					fmt.Printf("  STDOUT: %s\n", stdoutLine)
				}
				if stderrLine != "" {
					fmt.Printf("  STDERR: %s\n", stderrLine)
				}
			}
		},
	)
	if err != nil {
		return "", fmt.Errorf("wget down file FAIL: %w", err)
	}
	return localFile, nil
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

func ExecCommandWithProgress(commandStrs string, progress func(step int, total int, cmd string, stdoutLine string, stderrLine string)) (err error) {
	commands := strings.Split(commandStrs, "&&")
	total := len(commands)

	for idx, commandStr := range commands {
		command, err := shellwords.Parse(commandStr)
		if err != nil {
			return fmt.Errorf("解析命令失败: %v", err)
		}
		if len(command) == 0 {
			return fmt.Errorf("命令数组不能为空")
		}

		cmd := exec.Command(command[0], command[1:]...)

		// 创建管道以实时读取输出
		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()

		if err = cmd.Start(); err != nil {
			return fmt.Errorf("命令启动失败: %v", err)
		}

		// 实时读取 stdout/stderr
		go func() {
			buf := make([]byte, 1024)
			for {
				n, _ := stdoutPipe.Read(buf)
				if n == 0 {
					break
				}
				progress(idx+1, total, strings.Join(command, " "), string(buf[:n]), "")
			}
		}()

		go func() {
			buf := make([]byte, 1024)
			for {
				n, _ := stderrPipe.Read(buf)
				if n == 0 {
					break
				}
				progress(idx+1, total, strings.Join(command, " "), "", string(buf[:n]))
			}
		}()

		if err = cmd.Wait(); err != nil {
			return fmt.Errorf("命令执行失败: %v", err)
		}
	}
	return nil
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

// AddToFile 将文本写入文件，如果目录不存在则创建
func AddToFile(filename, content string) error {
	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 创建或打开文件（追加模式）
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 写入内容
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}
