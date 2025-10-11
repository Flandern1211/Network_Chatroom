// main.go
package main

import (
	"net_chat/internal/server"
)

func main() {
	// 创建服务器实例
	s := server.NewServer(":8080")

	// 启动服务器
	s.Start()
}
