# Heartbeat - 轻量级 URL 监控工具

一个超轻量级的 Docker 容器，用于周期性访问指定 URL 并打印响应。基于 Go 语言编写，使用 `scratch` 作为基础镜像。

## 技术栈

- **语言**: Go 1.21
- **基础镜像**: scratch（最小化）
- **构建方式**: 多阶段构建
- **二进制**: 静态编译（CGO_ENABLED=0）

## 环境变量

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `TARGET_URL` | 目标 URL（必需） | - | `https://your-url.com/api/health` |
| `INTERVAL` | 访问间隔（秒） | `30` | `60` |
| `SHOW_RESPONSE` | 是否显示响应内容 | `true` | `true`, `false` |

## 快速开始

### 方式 1: 使用 Docker Compose（推荐）

1. 编辑 `docker-compose.yml` 修改环境变量：

```yaml
services:
  heartbeat:
    image: sstarbucks/heartbeat:latest
    container_name: heartbeat
    restart: unless-stopped
    environment:
      - TARGET_URL=https://your-url.com/api/health
      - INTERVAL=60
      - SHOW_RESPONSE=true 

```

2. 启动服务：

```bash
docker-compose up -d
```

3. 查看日志：

```bash
docker-compose logs -f
```

### 方式 2: 直接使用 Docker

1. 运行容器：

```bash
docker run -d \
  --name heartbeat \
  --restart unless-stopped \
  -e TARGET_URL="https://your-url.com/api/health" \
  -e INTERVAL=30 \
  -e SHOW_RESPONSE=true \
  sstarbucks/heartbeat:latest
```

2. 查看日志：

```bash
docker logs -f heartbeat
```

## 日志输出示例

```
2025/10/12 10:30:00 Starting heartbeat service
2025/10/12 10:30:00 Target URL: https://api.example.com/health
2025/10/12 10:30:00 Interval: 30 seconds
2025/10/12 10:30:00 Show Response: true
2025/10/12 10:30:00 ---
2025/10/12 10:30:00 [2025-10-12 10:30:00] Status: 200 | Duration: 145ms | Response: {"ok":true}
```
