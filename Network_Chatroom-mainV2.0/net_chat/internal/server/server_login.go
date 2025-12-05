package server

import (
	"fmt"
	"net_chat/internal/protocol"
	"strings"
)

func (s *Server) HandleLogin(msg *protocol.Message, c *ClientConn) {
	content := strings.Split(msg.Content.(string), "|")
	username := strings.TrimSpace(content[0])
	password := strings.TrimSpace(content[1])
	//1.先在数据库中检查是否存在和账号密码的正确性
	// 1. 先检查用户是否已经在线
	s.mu.RLock()
	if _, ok := s.users[username]; ok {
		s.mu.RUnlock()
		c.Outgoing <- &protocol.Message{
			Type:    "login_fail",
			Content: "用户在线中",
			From:    "system",
		}
		return
	}
	s.mu.RUnlock()
	err := s.CheckUser(username, password)
	if err != nil {
		c.Outgoing <- &protocol.Message{
			Type:    "login_fail",
			Content: err.Error(),
			From:    "system",
		}
		//2.账号密码正确且存在检查在线用户列表是否已经存在该用户
	}
	if err == nil {
		err = s.AddUser(username, c)
		c.Name = username
		//发送登录成功的消息
		c.Outgoing <- &protocol.Message{
			Type:    "login_success",
			Content: "Welcome" + username,
			From:    "system",
		}
		//发送未读消息提醒
		s.sendUnreadMessages(c, username)
		//用户活跃度+1
		OnUserLogin(username)
		//广播用户上线通知
		s.Broadcast(&protocol.Message{
			Type:    "notice",
			Content: fmt.Sprintf("%s 加入了聊天室", username),
			From:    "system",
		})

	} else {
		c.Outgoing <- &protocol.Message{
			Type:    "login_fail",
			Content: "登录失败: " + err.Error(),
			From:    "system",
		}
	}
}

// AddUser 添加用户到用户列表map中,便于后续的私聊和查看用户列表
func (s *Server) AddUser(name string, c *ClientConn) error {
	//加锁保证数据添加无误
	s.mu.Lock()
	//检查重名
	if _, ok := s.users[name]; ok {
		return fmt.Errorf("在线中") //名字存在
	}
	//每个名字维持一个对应连接
	s.users[name] = c
	fmt.Printf("添加用户%s成功", name)
	s.mu.Unlock()
	return nil
}
