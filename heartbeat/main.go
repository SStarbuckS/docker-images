package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	// 从环境变量读取配置
	url := os.Getenv("TARGET_URL")
	if url == "" {
		log.Fatal("必须设置 TARGET_URL 环境变量")
	}

	intervalStr := os.Getenv("INTERVAL")
	if intervalStr == "" {
		intervalStr = "30"
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		log.Fatal("INTERVAL 必须是正整数（单位：秒）")
	}

	showResponse := os.Getenv("SHOW_RESPONSE")
	if showResponse == "" {
		showResponse = "true"
	}
	printResponse := showResponse == "true" || showResponse == "1" || showResponse == "yes"

	log.Printf("心跳服务启动")
	log.Printf("目标地址: %s", url)
	log.Printf("请求间隔: %d 秒", interval)
	log.Printf("显示响应: %v", printResponse)
	log.Println("---")

	// 循环执行：请求完成后等待间隔时间再执行下一次
	for {
		makeRequest(url, printResponse)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func makeRequest(url string, printResponse bool) {
	startTime := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[错误] 请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[错误] 读取响应失败: %v", err)
		return
	}

	if printResponse && len(body) > 0 {
		log.Printf("状态码: %d | 耗时: %v | 响应: %s", resp.StatusCode, duration, string(body))
	} else {
		log.Printf("状态码: %d | 耗时: %v", resp.StatusCode, duration)
	}
}
