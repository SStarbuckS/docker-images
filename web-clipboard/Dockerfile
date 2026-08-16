# syntax=docker/dockerfile:1

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod ./
COPY main.go ./
COPY static ./static

RUN apk add --no-cache tzdata && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/web-clipboard .

# Runtime stage
FROM scratch

WORKDIR /data

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /out/web-clipboard /web-clipboard

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/web-clipboard"]
CMD ["-config", "/data/clipboard-config.json"]
