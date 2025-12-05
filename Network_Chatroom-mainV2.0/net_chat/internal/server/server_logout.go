package server

import (
	"fmt"
	"log"
	"net_chat/internal/protocol"
)

func (s *Server) HandleLogout(c *ClientConn) {
	if c.Name != "" {
		username := c.Name
		err := s.RemoveUser(c.Name)
		if err != nil {
			log.Printf("在删除用户时发生错误%s:", err)
		}
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

// RemoveUser 删除用户
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
