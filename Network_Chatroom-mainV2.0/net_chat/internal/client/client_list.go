package client

import (
	"fmt"
	"net_chat/internal/protocol"
)

// RequestUserList 查看用户列表
func (c *Client) RequestUserList(inputLines <-chan string) error {
	done = make(chan struct{})
	msg := &protocol.Message{
		Type: "list",
	}
	if err := protocol.SendMsg(c.conn, msg); err != nil {
		return err
	}

	// 等待服务器返回最新的用户列表
	<-done

	fmt.Println("用户在线列表（输入exit退出查看）")
	if len(c.users) == 1 && c.users[0] == "" {
		fmt.Println("当前没有其他用户在线")
	} else {
		for i, user := range c.users {
			fmt.Printf("%d. %s\n", i+1, user)
		}
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
	}
}
