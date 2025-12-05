package client

import (
	"fmt"
	"net_chat/internal/protocol"
	"strings"
)

// ShowPrivateChat 私聊
// 也同步阻塞读取 inputLines，先读取目标用户名，再读取消息
func ShowPrivateChat(client *Client, inputLines <-chan string) {
	fmt.Println("\n======= 私聊 =======")
	fmt.Print("请输入对方用户名: ")
	targetUser, ok := <-inputLines
	if !ok {
		return
	}
	targetUser = strings.TrimSpace(targetUser)
	if targetUser == "" || targetUser == "exit" {
		fmt.Println("目标用户名不能为空且不能为'exit'")
		return
	}
	if targetUser == client.username {
		fmt.Println("目标用户名不能为自己")
		return
	}

	fmt.Printf("与 %s 私聊中，输入消息并按回车发送，输入 'exit' 退出私聊\n", targetUser)

	msg := &protocol.Message{
		Type: "privatebegin",
		From: client.username,
		To:   targetUser,
	}
	err := protocol.SendMsg(client.conn, msg)
	if err != nil {
		fmt.Println("发送消息时发生错误", err)
	}

	for {
		line, ok := <-inputLines
		if !ok {
			return
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" {
			return
		}
		if err := client.SendChatMessage(input, targetUser); err != nil {
			fmt.Println("[错误] 发送失败，", err)
		}
		fmt.Printf("[%s]:[%s]\n", client.username, input)
		// 非阻塞检查 quit（若 quit 已关闭则返回）
		select {
		case <-client.quit:
			return
		default:
		}
	}
}
