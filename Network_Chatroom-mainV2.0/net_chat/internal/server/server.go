package server

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Server struct {
	Addr  string                 //监听地址
	mu    sync.RWMutex           //保护用户列表map的锁
	users map[string]*ClientConn //用户列表
	now   time.Time
}

// NewServer 构造函数
func NewServer(addr string) *Server {
	return &Server{
		Addr:  addr,                         //监听地址
		users: make(map[string]*ClientConn), //用户列表map
	}
}

// Start Start启动监听并接收连接
func (s *Server) Start() error {
	//监听端口
	s.now = time.Now()
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		fmt.Printf("服务端监听%s失败", s.Addr)
		return err
	}
	fmt.Println("服务端正在监听端口", s.Addr)

	//不断接收连接
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("连接失败:", err)
			//一个链接失败不影响链接其他的客户
			continue
		}
		//为新的连接建立ClientConn并启动循环读写协程
		cc := NewClientConn(conn)
		cc.Start(s)
	}
}
