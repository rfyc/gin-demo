package utils

import (
	"bytes"
	"errors"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"time"
)

func DownXlsxFileFromUrl(url string) (*excelize.File, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.New("Error creating request: " + err.Error())
	}
	client := &http.Client{Timeout: time.Second * 600}

	resp, err := client.Do(request)
	if err != nil {
		return nil, errors.New("Error downloading file: " + err.Error())
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP request failed with status code: " + resp.Status)
	}

	// 将响应体读取到字节切片中
	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("Error reading file: " + err.Error())
	}

	// 使用bytes.NewReader将字节切片转换为io.Reader
	reader := bytes.NewReader(fileBytes)

	// 使用excelize.OpenReader打开文件流
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, errors.New("OpenReader file: " + err.Error())
	}
	return f, nil
}
