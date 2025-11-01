package server

import (
	"fmt"
	"net"
	"net_chat/internal/protocol"
	"strings"
	"sync"
)

type Server struct {
	Addr  string                 //监听地址
	mu    sync.RWMutex           //保护用户列表map的锁
	users map[string]*ClientConn //用户列表
}

// 构造函数
func NewServer(addr string) *Server {
	return &Server{
		Addr:  addr,                         //监听地址
		users: make(map[string]*ClientConn), //用户列表map
	}
}

// Start启动监听并接收连接
func (s *Server) Start() error {
	//监听端口
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

// 添加用户到用户列表map中,便于后续的私聊和查看用户列表
func (s *Server) AddUser(name string, c *ClientConn) error {
	//加锁保证数据添加无误
	s.mu.Lock()
	//检查重名
	if _, ok := s.users[name]; ok {
		return fmt.Errorf("已存在") //名字存在
	}
	//每个名字维持一个对应连接
	s.users[name] = c
	fmt.Printf("添加用户%s成功", name)
	s.mu.Unlock()
	return nil
}

// 删除用户
func (s *Server) RemoveUser(name string) error {
	if name == "" {
		return fmt.Errorf("用户名不能为空")
	}
	//加锁保证数据删除无误
	s.mu.Lock()
	defer s.mu.Unlock()
	//在用户列表map中查找该name，存在的话就删除
	if _, ok := s.users[name]; ok {
		delete(s.users, name)
		fmt.Printf("删除用户%s成功\n", name)
		return nil
	} else {
		return fmt.Errorf("用户%s不存在", name)
	}

}

// 返回指定用户对应的连接
func (s *Server) GetUser(name string) *ClientConn {
	//加锁保证查询用户无误
	s.mu.RLock()
	clientele := s.users[name]
	s.mu.RUnlock()
	return clientele
}

// 返回用户列表
func (s *Server) ListUsers() []string {
	s.mu.Lock()
	//用户列表切片
	var out []string
	for n := range s.users {
		out = append(out, n)
	}
	s.mu.Unlock()
	return out
}

// 广播函数
func (s *Server) Broadcast(msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	//遍历用户列表map来对在线用户发送消息
	for _, c := range s.users {
		//该用户此时没有私聊对象才发送给他
		if msg.To == "" {
			//消息分类函数
			//使用select来防止因慢用户导致的整体阻塞
			select {
			case c.Outgoing <- msg:
			default:
				fmt.Printf("无法发送消息给 %s\n", c.Name)
			}
		}

	}
}

// Dispatch 分类处理用户消息
func (s *Server) Dispatch(msg *protocol.Message, c *ClientConn) {
	switch msg.Type {
	//处理登录请求
	case "login":
		err := s.AddUser(msg.Content, c)
		if err == nil {
			c.Name = msg.Content
			//发送登录成功的消息
			c.Outgoing <- &protocol.Message{
				Type:    "login_success",
				Content: "Welcome" + msg.Content,
				From:    "system",
			}
			//广播用户上线通知
			s.Broadcast(&protocol.Message{
				Type:    "notice",
				Content: fmt.Sprintf("%s 加入了聊天室", msg.Content),
				From:    "system",
			})

		} else {
			//登录失败
			c.Outgoing <- &protocol.Message{
				Type:    "login_fail",
				Content: "用户名" + err.Error(),
				From:    "system",
			}
		}
	//发送消息请求
	case "chat":
		if c.Name != "" {
			if msg.To != "" {
				//私聊
				targetUser := s.GetUser(msg.To)
				if targetUser != nil {
					targetUser.Outgoing <- &protocol.Message{
						Type:    "private_chat",
						Content: msg.Content,
						From:    msg.From,
						To:      msg.To,
					}

					//发送回执给自己
					c.Outgoing <- &protocol.Message{
						Type:    "private_chat_sent",
						Content: "发送成功",
						From:    "system",
					}
				} else {
					//目标不存在
					c.Outgoing <- &protocol.Message{
						Type:    "error",
						Content: fmt.Sprintf("发送失败，%s不在线", msg.To),
						From:    "system",
					}
				}

			} else {
				//群聊消息
				s.Broadcast(&protocol.Message{
					Type:    "chat",
					Content: msg.Content,
					From:    msg.From,
				})
			}
		}
	//查看用户列表请求
	case "list":
		users := s.ListUsers()
		c.Outgoing <- &protocol.Message{
			Type: "user_list",
			//用|隔开
			Content: strings.Join(users, "|"),
			From:    "system",
		}
	//用户登出请求
	case "logout":
		if c.Name != "" {
			username := c.Name
			s.RemoveUser(c.Name)
			c.Outgoing <- &protocol.Message{
				Type:    "logout_success",
				Content: "你已经从聊天室退出",
				From:    "system",
			}
			// 广播用户下线消息
			s.Broadcast(&protocol.Message{
				Type:    "notice",
				Content: fmt.Sprintf("%s 离开了聊天室", username),
				From:    "system",
			})
		}
	}
}
