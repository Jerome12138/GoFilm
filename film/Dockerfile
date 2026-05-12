FROM golang:1.21

# 构建上下文 = 仓库根 (见 docker-compose.yml build.context: ..)
# 直接从根 server/ 拷贝源码, 避免与 film/ 维护双份.

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

LABEL maintainer="GoFilm"

WORKDIR /opt/server

COPY server/ /opt/server/

RUN go build -o main main.go

EXPOSE 3601

CMD ["./main"]
