# 使用官方 Go 镜像作为构建环境
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# 声明构建参数
ARG TARGETOS
ARG TARGETARCH

# 设置工作目录
WORKDIR /app

# 安装必要的依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -o navdesk .

# 使用轻量级的 alpine 镜像作为运行环境
FROM alpine:latest

# 安装 ca-certificates 和时区数据
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/navdesk .

# 复制静态文件
COPY --from=builder /app/public ./public

# 复制默认数据文件（用于首次启动初始化）
COPY --from=builder /app/data ./default-data

# 创建数据目录（用于挂载）
RUN mkdir -p /app/data

# 设置权限
RUN chmod +x navdesk

# 设置环境变量
ENV PORT=3000
ENV GIN_MODE=release

# 暴露端口
EXPOSE 3000

# 设置入口点脚本
COPY --from=builder /app/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 启动应用
ENTRYPOINT ["/entrypoint.sh"] 