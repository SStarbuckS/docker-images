package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

// Config 应用配置
type Config struct {
	Accounts map[string]*AccountConfig `json:"accounts"`
	App      AppConfig                 `json:"app"`
}

// AccountConfig 邮箱账号配置
type AccountConfig struct {
	Server       string   `json:"server"`
	Port         int      `json:"port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	PollInterval int      `json:"pollinterval"`
	SendPush     string   `json:"sendpush"`
	Folders      []string `json:"folders"`
	IdleTimeout  int      `json:"idletimeout"`
}

// AppConfig 应用级配置
type AppConfig struct {
	HeartbeatURL      string `json:"heartbeat_url"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	for name, acc := range config.Accounts {
		if acc.Port == 0 {
			acc.Port = 993
		}
		if acc.PollInterval == 0 {
			acc.PollInterval = 60
		}
		if acc.IdleTimeout == 0 {
			acc.IdleTimeout = 20
		}
		if len(acc.Folders) == 0 {
			acc.Folders = []string{"INBOX"}
		}
		// 验证必填字段
		if acc.Server == "" || acc.Username == "" || acc.Password == "" {
			return nil, fmt.Errorf("账号 %s 缺少必填字段 (server/username/password)", name)
		}
		if acc.Port < 1 || acc.Port > 65535 {
			return nil, fmt.Errorf("账号 %s 的 port 必须在 1-65535 之间", name)
		}
		if acc.PollInterval < 1 {
			return nil, fmt.Errorf("账号 %s 的 pollinterval 必须大于 0", name)
		}
		if acc.IdleTimeout < 1 {
			return nil, fmt.Errorf("账号 %s 的 idletimeout 必须大于 0", name)
		}
		if len(acc.Folders) > 1 {
			return nil, fmt.Errorf("账号 %s 当前仅支持监控一个文件夹，请只配置一个 folders 项", name)
		}
		if acc.Folders[0] == "" {
			return nil, fmt.Errorf("账号 %s 的 folders 不能包含空文件夹名", name)
		}
		if acc.SendPush != "" {
			if err := validateHTTPURL(acc.SendPush); err != nil {
				return nil, fmt.Errorf("账号 %s 的 sendpush 无效: %w", name, err)
			}
		}
	}

	// 设置心跳默认值
	if config.App.HeartbeatInterval == 0 {
		config.App.HeartbeatInterval = 60
	}
	if config.App.HeartbeatInterval < 1 {
		return nil, fmt.Errorf("heartbeat_interval 必须大于 0")
	}
	if config.App.HeartbeatURL != "" {
		if err := validateHTTPURL(config.App.HeartbeatURL); err != nil {
			return nil, fmt.Errorf("heartbeat_url 无效: %w", err)
		}
	}

	return &config, nil
}

func validateHTTPURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme 必须是 http 或 https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("URL 缺少 host")
	}
	return nil
}
