// 已修改：添加心跳检测功能 (heartbeat)
package main

import (
	"fmt"
	"net/http"
	"strings"

	"infopush/common"
	"infopush/pusher"
)

// 全局配置管理器
var configManager *ConfigManager

// dynamicHandler 动态路由处理器
func dynamicHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应头为 JSON
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// 从URL路径末尾提取配置名称，自动忽略反向代理路径前缀
	fullPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(fullPath, "/")
	configPath := pathParts[len(pathParts)-1]

	// 统一检查配置路径
	if configPath == "" {
		sendErrorResponse(w, fmt.Sprintf("缺少配置路径 - 请求路径: %s", fullPath), "", nil)
		return
	}

	// 获取配置
	config, exists := configManager.GetConfig(configPath)
	if !exists {
		sendErrorResponse(w, fmt.Sprintf("配置不存在 - 请求配置: %s, 可用配置: %v", configPath, configManager.GetAllConfigNames()), configPath, nil)
		return
	}

	// 获取消息内容 - 缺少msg参数
	msg := r.FormValue("msg")
	if msg == "" {
		sendErrorResponse(w, fmt.Sprintf("缺少msg参数 - 配置: %s, 类型: %s", configPath, config.Type), configPath, nil)
		return
	}

	// 获取所有表单参数
	params := make(map[string]string)
	params["msg"] = msg
	params["title"] = r.FormValue("title")

	// 根据配置类型处理消息
	var apiResponse string
	var err error

	switch config.Type {
	case "dingtalk_text":
		apiResponse, err = pusher.SendDingTalkText(config.Config, params)
	case "telegram_text":
		apiResponse, err = pusher.SendTelegramText(config.Config, params)
	case "wecom_mpnews":
		apiResponse, err = pusher.SendWecomMPNews(config.Config, params)
	case "wecom_robot_text":
		apiResponse, err = pusher.SendWecomRobotText(config.Config, params)
	default:
		sendErrorResponse(w, fmt.Sprintf("不支持的推送类型: %s", config.Type), configPath, params)
		return
	}

	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("%v", err), configPath, params)
		return
	}

	// 成功响应
	ts := common.Timestamp()
	// 控制台输出：[时间] 配置名 - {"code":"200","msg":"API原始响应"}
	fmt.Printf("[%s] %s - {\"code\":\"200\",\"msg\":\"%s\"}\n", ts, configPath, apiResponse)
	// 用户响应：{"code":"200","msg":"Success"}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"code":"200","msg":"Success"}`)
}

// sendErrorResponse 发送错误响应
func sendErrorResponse(w http.ResponseWriter, errorMsg, configPath string, params map[string]string) {
	ts := common.Timestamp()
	// 控制台输出：{"code":"404","msg":"API 返回内容"}
	fmt.Printf("[%s] %s - {\"code\":\"404\",\"msg\":\"%s\"}\n", ts, configPath, errorMsg)
	// 写入错误日志
	common.WriteErrorLog(ts, configPath, errorMsg, params)
	// 用户响应：{"code":"404","msg":"资源不存在"}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `{"code":"404","msg":"资源不存在"}`)
}

func main() {
	// 记录启动时间到日志文件
	common.LogStartupTime()

	// 加载配置文件
	var err error
	configManager, err = NewConfigManager("data/config.json")
	if err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		return
	}

	// 启动心跳检测服务
	heartbeat := &common.HeartbeatService{
		URL:      configManager.HeartbeatURL,
		Interval: configManager.HeartbeatInterval,
	}
	heartbeat.Start()

	// 显示心跳检测状态信息
	if configManager.HeartbeatURL != "" {
		fmt.Printf("心跳检测已启动: %s (间隔: %d秒)\n", configManager.HeartbeatURL, configManager.HeartbeatInterval)
	} else {
		fmt.Println("心跳检测未配置")
	}

	// 注册动态路由
	http.HandleFunc("/", dynamicHandler)

	// 启动服务器
	fmt.Println("多配置消息推送服务启动中...")
	fmt.Println("服务地址: http://localhost:8080")

	fmt.Println("路由模式: 自适应路径前缀")

	fmt.Println("支持的配置路由:")
	for _, name := range configManager.GetAllConfigNames() {
		config, _ := configManager.GetConfig(name)
		fmt.Printf("  - http://localhost:8080/[路径前缀/]%s/ (类型: %s)\n", name, config.Type)
	}
	fmt.Println("使用方法: POST/GET 请求，参数 msg=消息内容 [title=标题]")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}
