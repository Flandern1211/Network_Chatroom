package main

import (
	"log"
	"net_chat/internal/database"
	"net_chat/internal/database/redis"
	"net_chat/internal/server"
	"os"
)

func main() {

	//1.服务器监听地址设置
	addr := os.Getenv("CHAT_SERVER_ADDR")
	if addr == "" {
		addr = ":8080" //等同于"0.0.0.0:8080"
	}

	//2.初始化数据库
	if err := database.InitMySQL(); err != nil {
		log.Fatalf("初始化数据库失败:%v", err)
	} else {
		log.Printf("初始化MySQL数据库连接成功！")
	}
	defer database.CloseDB()

	//3.初始化redis端
	if err := redis.InitRedis("localhost:6379", "", 0); err != nil {
		log.Fatalf("初始化redis失败:%v", err)
	} else {
		log.Println("初始化redis成功！")
	}

	// 4. 创建服务器实例
	s := server.NewServer(addr)

	// 5. 启动服务器
	if err := s.Start(); err != nil {
		log.Fatalf("服务器无法正常启动:%s", err)
	}

}
