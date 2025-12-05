#构建阶段

#1.先设置构建所使用的环境和对应的工作目录
#基于Alpine Linux的轻量镜像开始构建
#As builder 表示给这个构建过程起一个名字,之后可以直接引用builder这个名字来代指这个构建过程
FROM golang:1.24-alpine As builder

#设置对应的工作目录,后续的所有命令(如COPY,RUN)都会在这个目录下执行
WORKDIR /app

#设置Go模块代理为国内的http://goproxy.cn,加快依赖下载速度
#,direct 在代理无法获取某些模块时，直接连接源站
ENV GOPROXY=http://goproxy.cn,direct

#2.将go.mod和go.sum提前复制下载

#将<本地项目根目录>下的go.mod和go.sum文件复制到容器的当前工作目录/app
#这样可以提前先下载好依赖，利用Docker的缓存机制：只要go.mod和go.sum没变，就不需要重新下载依赖
COPY go.mod go.sum ./

#下载项目所需要的依赖
RUN go mod download

#3.将本地项目根目录下的所有文件复制到容器的/app目录中
COPY . .

#4.编译Go程序：
    #CGO_ENABLED=0 禁用CGO，使得《编译出的是静态的二进制文件》，不依赖系统动态库，更适合在Alpine等最小化系统中运行
    #GOOS=linux 使用linux操作系统来编译
    #GOARCH=amd64 指定目标CPU架构为amd64

    #go build -o /app/chat-server:将编译的结构输出为/app/chat-server的可执行文件
    # ./internal/server/main 根据你的项目目录结构来指定入口源码路径  这里执行我的在./internal/server/main目录下的main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/chat-server ./internal/server/main


#####
#上面的都属于构建阶段，即
#设置项目的运行环境
#将需要运行的项目的源码编译为静态的二进制文件
#目的是为了保留运行程序所需的最小环境，大幅减小最终镜像体积
#5. 运行阶段
# 开始一个新的构建阶段，使用更小的 alpine:latest 作为基础镜像（不包含 Go 编译环境）
FROM alpine:latest

#安装ca-certificates包，提供SSL/TLS证书信任链
#这对需要发起HTTPS请求（如调用API,访问数据库等）的程序是必需的
# --no-cache 表示不保留 apk 的缓存，避免增大镜像体积
RUN apk --no-cache add ca-certificates


# 从前面定义的 builder 阶段中，复制编译好的 /app/chat-server 二进制文件到当前镜像的根目录（. 即 /）
# 这是多阶段构建的核心：只拷贝最终需要的产物，丢弃所有构建工具和中间文件
COPY  --from=builder /app/chat-server .


# 声明容器在运行时会监听 15000 端口
# 注意：EXPOSE 只是文档化作用，实际端口映射需在 docker run -p 时指定
EXPOSE 8080

# 设置容器启动时默认执行的命令：运行 ./chat-server 程序
CMD ["./chat-server"]

