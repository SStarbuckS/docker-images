# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY main.go .

RUN apk add --no-cache tzdata && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o useragent main.go

# Runtime stage
FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/useragent /useragent

EXPOSE 8080

ENTRYPOINT ["/useragent"]
