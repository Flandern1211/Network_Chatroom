package server

import (
	"fmt"
	"log"
	"net_chat/internal/database/redis"
	"net_chat/internal/protocol"
	"time"
)

// 发送未读消息提醒
func (s *Server) sendUnreadMessages(c *ClientConn, username string) {
	//获取未读消息数
	unread, err := redis.GetUnreadForUser(username)
	if err != nil {
		log.Printf("获取用户%s未读消息失败%s", username, err)
	}

	//发送未读消息数
	if len(unread) > 0 {
		var unreadInfo string
		for sender, count := range unread {
			unreadInfo = fmt.Sprintf("%s(%s)", sender, count)
		}
		c.Outgoing <- &protocol.Message{
			Type:    "notice",
			Content: fmt.Sprintf("您有来自%s的未读私聊消息", unreadInfo),
			From:    "system",
		}
	}

}

// 发送最近的聊天消息
func (s *Server) sendRecentRoomMessages(c *ClientConn) {
	msgs, err := redis.GetRoomLastNMessage("main_room", 10)
	if err != nil {
		c.Outgoing <- &protocol.Message{
			Type:    "error",
			Content: "获取历史消息失败",
			From:    "system",
		}
		log.Fatalf("获取聊天室历史消息失败%v", err)

	}
	var result string
	if len(msgs) == 0 {
		result = "暂无历史消息"
	} else {
		for _, msg := range msgs {
			sender := msg.Values["sender"].(string)
			content := msg.Values["content"].(string)
			var timestamp int64
			switch t := msg.Values["ts"].(type) {
			case int64:
				timestamp = t
			case string:
				// 如果是字符串，尝试转换为 int64
				_, err2 := fmt.Sscanf(t, "%d", &timestamp)
				if err2 != nil {
					log.Printf("在转换用户历史消息的时间时发生错误:%s", err2)
				}
			}
			t := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
			result += fmt.Sprintf("[%s] %s: %s\n", t, sender, content)
		}
	}
	c.Outgoing <- &protocol.Message{
		Type:    "recent_room_messages",
		Content: result,
		From:    "system",
	}
}

// 发送私聊历史消息
func (s *Server) sendRecentPrivateMessages(c *ClientConn, userB string) {
	msgs, err := redis.GetPrivateLastNMessage(c.Name, userB, 10)
	fmt.Printf("聊天记录长度为%d", len(msgs))
	if err != nil {
		c.Outgoing <- &protocol.Message{
			Type:    "error",
			Content: "获取私聊历史消息失败",
			From:    "system",
		}
		log.Fatalf("获取私聊历史消息失败%v", err)

	}
	var result string
	if len(msgs) == 0 {
		result = "暂无历史消息"
	} else {
		for _, msg := range msgs {
			sender := msg.Values["sender"].(string)
			content := msg.Values["content"].(string)
			var timestamp int64
			switch t := msg.Values["ts"].(type) {
			case int64:
				timestamp = t
			case string:
				// 如果是字符串，尝试转换为 int64
				_, err2 := fmt.Sscanf(t, "%d", &timestamp)
				if err2 != nil {
					log.Printf("在转换用户历史消息的时间时发生错误:%s", err2)
				}
			}
			t := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
			result += fmt.Sprintf("[%s] %s: %s\n", t, sender, content)
		}
	}
	c.Outgoing <- &protocol.Message{
		Type:    "recent_private_messages",
		Content: result,
		From:    "system",
	}
	if err = redis.ClearUnreadForUser(c.Name, userB); err != nil {
		log.Fatal("清除对应用户的离线消息提醒失败")
	}
}
