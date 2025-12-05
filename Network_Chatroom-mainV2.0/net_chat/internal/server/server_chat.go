package server

import (
	"fmt"
	"log"
	"net_chat/internal/database/redis"
	"net_chat/internal/protocol"
)

func (s *Server) HandleChat(msg *protocol.Message, c *ClientConn) {
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
				_, err := redis.AddPrivateMessage(msg.From, msg.To, msg.Content.(string), true)
				if err != nil {
					log.Printf("在存储用户私聊消息时发生错误%s:", err)
				}
				//fmt.Println("存储私聊消息成功")
			} else {
				_, err := redis.AddPrivateMessage(msg.From, msg.To, msg.Content.(string), false)
				if err != nil {
					log.Printf("在存储用户私聊消息时发生错误%s:", err)
				}
				//fmt.Println("存储私聊消息成果")
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
			//存储聊天室消息
			_, err := redis.AddRoomMessage("main_room", msg.From, msg.Content.(string))
			if err != nil {
				return
			}
			OnUserPost(msg.From)
		}
	}
}

// GetUser 返回指定用户对应的连接
func (s *Server) GetUser(name string) *ClientConn {
	//加锁保证查询用户无误
	s.mu.RLock()
	clientele := s.users[name]
	s.mu.RUnlock()
	return clientele
}
