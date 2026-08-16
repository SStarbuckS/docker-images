# 构建阶段
FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS builder

# 声明平台参数
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

# 安装 CA 证书和时区数据（用于复制到最终镜像）
RUN apk add --no-cache ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod ./

# 复制源代码
COPY main.go ./

# 编译为静态二进制文件（支持多平台）
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -o heartbeat .

# 最终阶段 - 使用 scratch 获得最小镜像
FROM scratch

# 添加 CA 证书以支持 HTTPS 请求
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 添加时区数据
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 从构建阶段复制二进制文件
COPY --from=builder /build/heartbeat /heartbeat

# 运行应用
ENTRYPOINT ["/heartbeat"] 