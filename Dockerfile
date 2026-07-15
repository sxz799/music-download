
# 第一阶段：构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# 复制前端依赖文件
COPY frontend/package*.json ./
RUN npm install

# 复制前端源代码并构建
COPY frontend/ ./
RUN npm run build

# 第二阶段：构建Go后端
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app

# 设置Go环境变量
ENV CGO_ENABLED=0
ENV GOOS=linux

# 复制Go依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制后端源代码
COPY main.go ./

# 从第一阶段复制构建好的前端资源
COPY --from=frontend-builder /app/dist ./frontend/dist

# 编译Go应用
RUN go build -ldflags "-s -w" -o music-download .

# 第三阶段：构建最小运行镜像
FROM alpine:latest

# 安装必要的包（CA证书用于HTTPS下载）
RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从第二阶段复制编译好的二进制文件
COPY --from=backend-builder /app/music-download .

# 创建downloads目录
RUN mkdir -p /app/downloads
VOLUME ["/app/downloads"]

# 设置环境变量
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./music-download"]
