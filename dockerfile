FROM golang:1.21-alpine AS builder

ARG PROJECT_NAME 
ARG BUILD_VERSION
ARG BUILD_TIME
ARG GIT_BRANCH
ARG GO_VERSION

WORKDIR /app
COPY config.yml /app

# 设置Go代理和环境变量
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/media_center \
       -ldflags "-s -X 'main.ProjectName=${PROJECT_NAME}' \
                    -X 'main.BuildVersion=${BUILD_VERSION}' \
                    -X 'main.BuildTime=${BUILD_TIME}' \
                    -X 'main.GitBranch=${GIT_BRANCH}' \
                    -X 'main.GoVersion=${GO_VERSION}'" \
                    src/main/main.go

# 第二阶段：运行阶段 - 使用极小的Alpine镜像
FROM alpine:latest

# 安装CA证书（如果需要访问HTTPS外部API）
RUN apk --no-cache add ca-certificates

# 创建一个非root用户来运行应用（安全最佳实践）
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -D appuser

# 设置工作目录
WORKDIR /home/appuser/
RUN mkdir log conf bin
# 从构建阶段复制编译好的二进制文件
COPY --from=builder --chown=appuser:appuser /app/media_center ./bin
COPY --from=builder --chown=appuser:appuser /app/config.yml ./conf

# 切换到非root用户
USER appuser
WORKDIR /home/appuser/bin

EXPOSE 8100/tcp
CMD ["sh", "-c", "./media_center"]