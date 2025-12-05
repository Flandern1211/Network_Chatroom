package client

import (
	"fmt"
	"net_chat/internal/protocol"
)

// RequestRecentMessages 在 client.go 中添加请求历史消息的方法
func (c *Client) RequestRecentMessages(inputLines <-chan string) error {
	msg := &protocol.Message{
		Type: "room_messages",
	}
	for {
		select {
		case input := <-inputLines:
			if input == "exit" {
				return nil
			} else {
				fmt.Println("输入exit退出")
			}
		}
		return protocol.SendMsg(c.conn, msg)
	}

}
