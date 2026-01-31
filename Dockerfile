FROM golang:1.23-alpine AS builder

WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache git ca-certificates

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o donation-station ./cmd/server

# 运行镜像
FROM alpine:3.19

WORKDIR /app

# 安装CA证书用于HTTPS请求
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 从构建阶段复制二进制文件
COPY --from=builder /app/donation-station .
COPY --from=builder /app/static ./static

# 创建非root用户
RUN adduser -D -g '' appuser
USER appuser

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./donation-station"]
