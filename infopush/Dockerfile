# 多阶段构建 - 构建阶段
FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS builder

# 声明构建参数
ARG TARGETOS
ARG TARGETARCH

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache ca-certificates tzdata

# 复制go模块文件和源代码
COPY go.mod ./
COPY *.go ./
COPY common/ ./common/
COPY pusher/ ./pusher/

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o infopush .

# 运行阶段 - 使用 scratch 获得最小镜像
FROM scratch

# 复制 CA 证书（用于 HTTPS 请求）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 复制时区数据
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/infopush .

# 复制 data 目录
COPY data/ /app/data/

# 暴露端口
EXPOSE 8080

# 启动应用
CMD ["./infopush"] 