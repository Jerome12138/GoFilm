# 多阶段构建: golang:1.21-alpine 编译 → distroless 运行, 目标 < 30MB.
# 上下文 = 仓库根 (见 docker-compose.yml build.context: ..)
FROM golang:1.21-alpine AS build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /src

# 先拷依赖再拷代码, 利用 docker layer 缓存加速重新构建
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
# -ldflags "-s -w" 去掉调试符号, 缩小产物
RUN go build -trimpath -ldflags="-s -w" -o /out/main main.go && \
    mkdir -p /out/static/upload/gallery

# 运行镜像: 静态 distroless, 非 root, 无 shell, 无包管理器.
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="gofilm-server"

WORKDIR /opt/server

COPY --from=build --chown=nonroot:nonroot /out/main /opt/server/main
COPY --from=build --chown=nonroot:nonroot /out/static /opt/server/static

USER nonroot:nonroot

EXPOSE 3601

# distroless 里没有 curl/wget, HEALTHCHECK 在 compose 层用 wget 镜像旁路探.
ENTRYPOINT ["/opt/server/main"]
