# InfoPush - 多配置消息推送服务

一个基于 Go 语言开发的轻量级消息推送服务，支持企业微信、Telegram、钉钉多种推送渠道。通过配置文件管理多个推送配置，支持动态路由和自适应路由前缀。

## 主要特性

-**多平台支持**: 企业微信图文消息、企业微信群机器人、Telegram Bot、钉钉机器人  
-**动态路由**: 基于 URL 路径自动选择推送配置  
-**灵活配置**: JSON 配置文件，支持多个同类型推送配置  
-**自适应路由前缀**: 根据请求路径自动适配反向代理和子目录部署  
-**心跳检测**: 独立的被动心跳检测功能，支持自定义间隔  
-**详细日志**: 毫秒级时间戳，配置级别的日志追踪  
-**Docker 支持**: 多平台容器化部署  
-**轻量高效**: 无外部依赖，单文件部署 

## 快速开始

### 项目结构

```
infopush/
├── common/              # 通用模块
│   ├── utils.go         # 通用工具函数
│   └── heartbeat.go     # 心跳检测模块
├── pusher/              # 推送模块
│   ├── wecom_mpnews.go      # 企业微信图文消息
│   ├── wecom_robot_text.go  # 企业微信群机器人文本
│   ├── telegram_text.go     # Telegram Bot 文本消息
│   └── dingtalk_text.go     # 钉钉机器人文本消息
├── data/                # 数据目录
│   ├── config.json      # 配置文件
│   └── error.log        # 错误日志（自动生成）
├── main.go              # 主程序，HTTP服务器和路由处理
├── config.go            # 配置文件管理
├── Dockerfile           # Docker构建文件
├── docker-compose.yml   # Docker Compose配置
├── .dockerignore        # Docker忽略文件
└── README.md            # 项目文档
```

### 本地运行

1. **克隆项目**
```bash
git clone SStarbuckS/infopush
cd infopush
```

2. **配置文件**
```bash
nano data/config.json
# 编辑配置文件，填入您的推送配置
```

3. **运行服务**
```bash
go run .
# 或编译后运行
go build -o infopush .
./infopush
```

### Docker 部署

1. **构建镜像**
```bash
docker build -t infopush .
```

2. **运行容器**
```bash
docker run -d \
  --name infopush \
  -p 8080:8080 \
  -v /path/to/data:/app/data \
  infopush
```

3. **使用 Docker Compose**
```bash
docker-compose up -d
```

## 配置说明

### 配置文件结构

`data/config.json` 文件结构如下：

```json
{
  "heartbeat_url": "",
  "heartbeat_interval": 60,
  "配置名称1": {
    "type": "推送类型",
    "config": {
      "具体配置参数": "值"
    }
  },
  "配置名称2": {
    "type": "推送类型",
    "config": {
      "具体配置参数": "值"
    }
  }
}
```

### 自适应路由配置

- 程序使用 URL 最后一个路径段匹配配置名称
  - `/配置名/`: 无前缀访问
  - `/push/配置名/`: 自动适配 `/push` 前缀
  - `/tuisong/配置名/`: 自动适配 `/tuisong` 前缀

### 心跳检测配置

- `heartbeat_url`: 心跳检测目标 URL（留空则不启用）
- `heartbeat_interval`: 心跳检测间隔（单位：秒）

**示例**:
```json
{
  "heartbeat_url": "https://example.com/ping",
  "heartbeat_interval": 60
}
```

**功能说明**:
- 如果 `heartbeat_url` 为空字符串，心跳检测不会启动
- 心跳检测在独立的定时器中运行，不会影响其他功能
- 每次心跳请求会在控制台输出响应状态和内容
- 请求失败不会影响下一次执行

### 企业微信图文消息配置

```json
{
  "wecom_example": {
    "type": "wecom_mpnews",
    "config": {
      "APIBaseURL": "https://qyapi.weixin.qq.com",
      "CorpID": "企业ID",
      "CorpSecret": "应用密钥",
      "AgentID": "应用ID",
      "ThumbMediaID": "图文消息缩略图的媒体ID",
      "Author": "作者名称",
      "DefaultTitle": "默认标题"
    }
  }
}
```

**获取配置参数**:
1. 登录企业微信管理后台
2. 创建应用，获取 `AgentID` 和 `CorpSecret`
3. 上传素材获取 `ThumbMediaID`

### 企业微信群机器人文本消息配置

```json
{
  "wecom_robot_text_example": {
    "type": "wecom_robot_text",
    "config": {
      "APIBaseURL": "https://qyapi.weixin.qq.com",
      "Keys": [
        "机器人Key1",
        "机器人Key2",
        "机器人Key3"
      ]
    }
  }
}
```

**获取配置参数**:
1. 在企业微信群中添加群机器人
2. 复制机器人Webhook URL中的key参数
3. 可配置多个机器人key实现负载均衡，避免速率限制

### Telegram Bot 文本消息配置

```json
{
  "telegram_text_example": {
    "type": "telegram_text",
    "config": {
      "Token": "Bot Token",
      "ChatID": "聊天ID",
      "APIBaseURL": "https://api.telegram.org"
    }
  }
}
```

**获取配置参数**:
1. 与 @BotFather 对话创建 Bot，获取 `Token`
2. 获取 `ChatID`: 向 Bot 发送消息后访问 `https://api.telegram.org/bot<TOKEN>/getUpdates`

### 钉钉机器人文本消息配置

```json
{
  "dingtalk_text_example": {
    "type": "dingtalk_text",
    "config": {
      "AccessToken": "机器人Webhook Token",
      "APIBaseURL": "https://oapi.dingtalk.com/robot/send"
    }
  }
}
```

**获取配置参数**:
1. 创建钉钉群聊
2. 添加自定义机器人
3. 复制 Webhook URL 中的 `access_token` 参数

### 如何添加多个相同类型的配置？

在配置文件中使用不同的配置名称即可:

```json
{
  "wecom_sales": { "type": "wecom_mpnews", "config": {...} },
  "wecom_ops": { "type": "wecom_mpnews", "config": {...} }
}
```

## API 使用

### 基本调用

**HTTP 方法**: GET 或 POST

**URL 格式**: 
- 无前缀: `http://localhost:8080/配置名/`
- 自适应前缀: `http://localhost:8080/任意路径前缀/配置名/`

配置名称必须位于 URL 的最后一个路径段，反向代理前缀无需写入配置文件。

**必需参数**:
- `msg`: 消息内容

**可选参数**:
- `title`: 消息标题 (仅企业微信图文消息支持，其他平台忽略)

### 使用示例

#### cURL 示例

```bash
# 企业微信图文消息推送 (POST)
curl -X POST "http://localhost:8080/wecom_example/" \
  -d "msg=测试消息内容" \
  -d "title=重要通知"

# 企业微信群机器人文本消息推送 (POST)
curl -X POST "http://localhost:8080/wecom_robot_text_example/" \
  -d "msg=系统告警：服务器负载过高"

# Telegram文本消息推送 (GET)
curl "http://localhost:8080/telegram_text_example/?msg=Hello%20World"

# 钉钉文本消息推送 (POST)
curl -X POST "http://localhost:8080/dingtalk_text_example/" \
  -d "msg=系统告警：CPU使用率超过90%"
```

#### Python 示例

```
def send_notification(push_content, retries=3, timeout=5):
    push_url = "http://localhost:8080/wecom_example"
    data = {
        'title': '新提醒',  # 可选的
        'msg': push_content  # 必要的
    }
    for attempt in range(retries):
        try:
            response = requests.post(push_url, data=data, timeout=timeout)
            response.raise_for_status()  # 非2xx状态码会抛出异常
            print(f"推送完成: {response.text}\n")
            return
        except Exception as error:
            print(f"推送发生错误: {error}")
        time.sleep(1)  # 等待一秒再重试
    print("推送重试均失败。\n")
```

### 响应格式

**成功响应** (HTTP 200):
```json
{"code":"200","msg":"Success"}
```

**错误响应** (HTTP 404):
```json
{"code":"404","msg":"资源不存在"}
```

## 日志格式

服务运行时会在控制台输出统一格式的日志信息：

```
# 成功响应
[时间] 配置名 - {"code":"200","msg":"API原始响应"}

# 错误响应
[时间] 配置名 - {"code":"404","msg":"错误信息"}
```

示例：
```
[2025-09-29 00:42:12.714] wecom_example - {"code":"200","msg":"{\"errcode\":0,\"errmsg\":\"ok\"}"}
[2025-09-29 00:42:13.856] dingtalk_example - {"code":"200","msg":"{\"errcode\":0,\"errmsg\":\"ok\"}"}
[2025-09-29 00:42:15.321] telegram_example - {"code":"200","msg":"{\"ok\":true,\"result\":{...}}"}
```

## Docker 部署

### Dockerfile

项目包含多阶段构建的 Dockerfile，支持多平台构建：

```dockerfile
# 支持的平台
- linux/amd64
- linux/arm64  
- linux/arm/v7
```

### 环境变量

- `TZ`: 时区设置，默认 `Asia/Shanghai`

### 数据卷

- `/app/data`: 数据目录挂载点（配置文件路径为 `/app/data/config.json`）

## 🔧 开发说明

### 添加新的推送平台

1. 在 `pusher/` 目录创建新的推送模块文件 (如 `newplatform_msgtype.go`)
   - 建议使用 `平台_消息类型.go` 的命名规范
2. 实现统一的推送函数接口：
   ```go
   func SendNewPlatformMsgType(configData map[string]any, params map[string]string) (string, error)
   ```
3. 在 `main.go` 的 `switch` 语句中添加新的 case
4. 在配置文件中添加对应的配置示例

5. 重新编译项目


