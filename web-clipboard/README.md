# Web Clipboard Service

这是一个单实例 Web 剪贴板服务，用于在多台设备之间同步纯文本内容。服务不处理图片、音频或其他多媒体数据。

## 功能范围

- 一个共享剪贴板空间。
- 通过浏览器访问 Web 页面。
- 通过固定账号密码登录。
- Web 页面编辑区显示行号，点击行号可复制该行文本。
- 自动适配系统深色模式。
- 刷新和保存操作会显示结果提示。
- 非空文本会自动保留末尾空行，便于继续追加内容。
- 文本内容保存到本地 JSON 文件。
- 一个设备保存文本后，其他设备可手动刷新获取最新内容。

## 启动方式

直接启动服务：

```powershell
go run .
```

如果当前目录没有 `clipboard-config.json`，服务会视为首次启动，生成默认配置和随机账号密码并保存到该文件。首次生成时，服务启动日志也会打印账号密码，便于第一次登录。

如果需要修改端口、数据文件、账号或密码，可以编辑 `clipboard-config.json`。

启动后访问：

```text
http://localhost:8080
```

同一局域网内其他设备访问时，把 `localhost` 替换为运行服务的机器 IP。

## Docker 启动方式

构建镜像：

```powershell
docker build -t web-clipboard .
```

如需用 `buildx` 构建指定平台的本地镜像，可以使用：

```powershell
docker buildx build --platform linux/amd64 -t web-clipboard --load .
```

创建数据目录并启动容器：

```powershell
New-Item -ItemType Directory -Force docker-data
docker run --rm -p 8080:8080 -v "${PWD}\docker-data:/data" web-clipboard
```

容器默认使用 `/data/clipboard-config.json` 作为用户配置文件。首次启动时会在挂载的数据目录中生成配置和账号密码；如果不挂载 `/data`，容器删除后配置和剪贴板内容也会丢失。

如果通过 Nginx 挂在子路径，例如 `https://api.domain.com/clipboard/`，前端会根据当前页面路径把接口请求改为 `/clipboard/api/...`。Nginx 需要把该前缀剥离后再转发到容器内服务。

```nginx
location = /clipboard {
    return 301 /clipboard/;
}

location /clipboard/ {
    proxy_pass http://127.0.0.1:8080/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

## 用户配置文件

默认读取当前目录下的 `clipboard-config.json`。如果文件不存在，服务会自动生成。

```json
{
  "addr": ":8080",
  "dataPath": "clipboard.json",
  "maxMB": 1,
  "username": "user-example",
  "password": "generated-password",
  "createdAt": "2026-06-28T00:00:00Z"
}
```

可以通过 `-config` 指定其他配置文件：

```powershell
go run . -config "my-clipboard-config.json"
```

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| `addr` | `:8080` | HTTP 监听地址 |
| `dataPath` | `clipboard.json` | 文本数据保存文件 |
| `maxMB` | `1` | 单次保存的最大文本大小，单位 MB |
| `username` | 自动生成 | 登录用户名 |
| `password` | 自动生成 | 登录密码 |
| `createdAt` | 自动生成 | 配置创建时间 |

需要修改账号密码时，可以停止服务后编辑 `clipboard-config.json`，再重新启动服务。

## 登录会话

浏览器登录会话有效期为 30 天。清理浏览器 Cookie、切换浏览器或设备、修改服务端密码后，需要重新登录。

## 登录限速

服务会按来源 IP 做内存登录限速。15 分钟内登录失败 5 次后，该 IP 会被锁定 15 分钟，锁定期间登录接口返回 429。

该限速记录只保存在当前进程内存中，服务重启后会清空。通过 Nginx 反代部署时，需要传递 `X-Real-IP` 或 `X-Forwarded-For`，否则服务只能看到反代服务器 IP。

## HTTP API

浏览器页面会自动调用这些接口。脚本或命令行客户端可以使用 HTTP Basic Auth 访问。

```http
GET /api/clipboard
Authorization: Basic <base64(username:password)>
```

```http
PUT /api/clipboard
Authorization: Basic <base64(username:password)>
Content-Type: application/json

{"text":"hello"}
```

返回数据格式：

```json
{
  "text": "hello",
  "updatedAt": "2026-06-28T00:00:00Z"
}
```

## 安全审计输出

服务会向控制台输出 `SECURITY_AUDIT` 审计日志，覆盖启动、登录成功、登录失败、退出登录、未授权访问、剪贴板读取和保存结果。日志会保留稳定的 `event` 事件码，并通过 `message` 输出中文事件说明。

审计日志不会输出密码、Cookie 或剪贴板正文。保存事件只记录文本字节数。

```text
SECURITY_AUDIT event="login_success" message="登录成功" remote="192.0.2.10" method="POST" path="/api/session" username="user-example"
SECURITY_AUDIT event="clipboard_saved" message="保存剪贴板成功" remote="192.0.2.10" method="PUT" path="/api/clipboard" username="user-example" bytes="12"
```

## 部署注意

- 不要在公网裸露 HTTP 服务；公网使用时应放在 HTTPS 反向代理后面。
- 密码应使用足够长的随机字符串，避免使用常见词或短密码。
- `clipboard-config.json` 包含明文账号密码，不要公开、同步或提交到版本库。
- `clipboard.json` 包含明文剪贴板内容，需要放在受控目录中。
- 使用 Docker 部署时应挂载 `/data`，避免容器删除后丢失配置和剪贴板内容。
- 第一版采用最后写入者覆盖规则；多设备同时保存时，以最后一次保存为准。
