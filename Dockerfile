# Build go
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
RUN go mod download
RUN go build -v -o v2rayS -trimpath -ldflags "-s -w -buildid=" ./main/main.go

# Release
FROM  alpine
# 安装必要的工具包
RUN  apk --update --no-cache add tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN mkdir /etc/v2rayS/
COPY --from=builder /app/v2rayS /usr/local/bin

ENTRYPOINT [ "v2rayS", "--config", "/etc/v2rayS/config.yml"]