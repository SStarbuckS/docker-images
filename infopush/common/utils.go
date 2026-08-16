package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Timestamp 返回格式化的时间戳
func Timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}

// HttpRequest 通用HTTP请求函数
func HttpRequest(method, url string, data []byte, timeout time.Duration) ([]byte, error) {
	var req *http.Request
	var err error

	if data != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// HandleAPIResponse 通用API响应处理函数
// 返回值: (API原始响应, 错误)
func HandleAPIResponse(responseStr, successPattern string) (string, error) {
	if strings.Contains(responseStr, successPattern) {
		return responseStr, nil
	}

	// 返回错误信息
	return responseStr, fmt.Errorf("%s", responseStr)
}

// LogStartupTime 记录程序启动时间到日志文件
func LogStartupTime() {
	// 确保 data 目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("创建data目录失败: %v\n", err)
		return
	}

	logFile, err := os.OpenFile("data/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("写入启动日志失败: %v\n", err)
		return
	}
	defer logFile.Close()

	ts := Timestamp()
	logEntry := fmt.Sprintf("\n========================================\n本次启动时间: %s\n========================================\n", ts)
	if _, err := logFile.WriteString(logEntry); err != nil {
		fmt.Printf("写入启动日志失败: %v\n", err)
	}
}

// WriteErrorLog 将错误信息写入日志文件
func WriteErrorLog(timestamp, configName, errorMsg string, params map[string]string) {
	// 确保 data 目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("[%s] 创建data目录失败: %v\n", timestamp, err)
		return
	}

	logFile, err := os.OpenFile("data/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[%s] 写入错误日志失败: %v\n", timestamp, err)
		return
	}
	defer logFile.Close()

	// 构造请求参数字符串
	paramsStr := ""
	if msg, ok := params["msg"]; ok && msg != "" {
		paramsStr += fmt.Sprintf(" msg=%s", msg)
	}
	if title, ok := params["title"]; ok && title != "" {
		paramsStr += fmt.Sprintf(" title=%s", title)
	}

	// 记录完整信息：配置名、错误信息、请求参数
	logEntry := fmt.Sprintf("[%s] %s - %s | 请求参数:%s\n",
		timestamp, configName, errorMsg, paramsStr)
	if _, err := logFile.WriteString(logEntry); err != nil {
		fmt.Printf("[%s] 写入错误日志失败: %v\n", timestamp, err)
	}
}
