package server

import (
	"fmt"
	"net_chat/internal/protocol"
)

// Broadcast 广播函数
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
