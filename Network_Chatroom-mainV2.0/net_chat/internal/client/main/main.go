package main

import (
	"fmt"
	"net_chat/internal/client"
	"os"
)

func main() {
	client := client.NewClient()
	//记得关闭连接和所有协程
	defer client.Close()

	//连接服务器
	addr := os.Getenv("CHAT_SERVER_ADDR")
	if addr == "" {
		addr = "localhost:8080" // 本地开发时的默认值；容器中会被覆盖为 server:8080
	}

	if err := client.Connect(addr); err != nil {
		fmt.Printf("连接服务器失败%v\n", err)
		return
	}

	// 启动客户端：只启动 readLoop
	client.Start(true)

	// 登录流程（带重试），传入 inputLines
	//登录-注册菜单
	client.LoginRegisterMenu(client.InputLines)

	//主菜单循环（从 inputLines 阻塞读取）
	client.MainMenu(client.InputLines)
}
