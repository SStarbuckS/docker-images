# User-Agent Generator

随机 User-Agent 生成器 API 服务，支持 Chrome Windows/Android 和 Safari iPhone/Mac。

生成逻辑来自 [random-user-agent](https://github.com/tarampampam/random-user-agent)。

## 快速开始

### Docker 部署

```bash
docker build -t useragent .
docker run -d -p 8080:8080 useragent
```
## API 使用

### 请求格式

```
GET /?type=<设备类型>&count=<数量>
```

### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `type` | 设备类型，多个用 `@` 分隔 | 随机 |
| `count` | 生成数量（最大100） | 1 |

### 设备类型

| 值 | 说明 |
|----|------|
| `win` | Chrome Windows |
| `android` | Chrome Android |
| `ios` | Safari iPhone |
| `mac` | Safari macOS |

### 示例

```bash
# 随机 1 个
curl http://localhost:8080/

# iOS 设备 5 个
curl "http://localhost:8080/?type=ios&count=5"

# Windows 和 Android 混合 3 个
curl "http://localhost:8080/?type=win@android&count=3"
```

### 响应格式

```json
{
  "code": "200",
  "ua": ["Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) ..."]
}
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `URL_PREFIX` | 路由前缀 | `/` |

### 反向代理示例

```yaml
# docker-compose.yml
environment:
  - URL_PREFIX=/ua
```

访问路径变为 `/ua` 或 `/ua/`。


