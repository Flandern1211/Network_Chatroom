package server

import (
	"log"
	"net_chat/internal/database"
	"net_chat/internal/protocol"
	"strings"
)

// HandleRegister 处理注册消息
func (s *Server) HandleRegister(msg *protocol.Message, c *ClientConn) {
	err, username := s.RegisterUser(msg.Content.(string))
	if err == nil {
		c.Outgoing <- &protocol.Message{
			Type:    "register_success",
			Content: "用户" + username + "注册成功,请登录",
			From:    "system",
		}
	} else {
		//注册失败
		c.Outgoing <- &protocol.Message{
			Type:    "register_fail",
			Content: "用户名" + err.Error(),
			From:    "system",
		}
	}
}

// RegisterUser 用户注册
func (s *Server) RegisterUser(cotent string) (error, string) {
	//验证逻辑放在这里，内容为用户名|密码
	userinfo := strings.Split(cotent, "|")
	username := strings.TrimSpace(userinfo[0])
	password := strings.TrimSpace(userinfo[1])
	err := database.RegisterUser(username, password)
	if err != nil {
		return err, ""
	}
	return nil, username
}

// CheckUser 检查用户是否在数据库中以及账号密码的正确性
func (s *Server) CheckUser(username, password string) error {
	err := database.AuthenticateUser(username, password)
	if err != nil {
		log.Printf("在检测用户的账号密码时发生错误: %v", err)
		return err
	}
	return nil
}
