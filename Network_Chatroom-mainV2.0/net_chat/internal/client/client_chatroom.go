package client

import (
	"fmt"
	"strings"
)

func ShowChatRoom(client *Client, inputLines <-chan string) {
	fmt.Println("\n======= 在聊天室中发送消息 =======")
	fmt.Println("输入消息并按回车发送，输入 'exit' 不再发送消息")

	for {
		// 直接阻塞读取下一行（在任意时刻，只有 main 路径在阻塞读 inputLines）
		line, ok := <-inputLines
		if !ok {
			// stdin 已关闭
			return
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" {
			return
		}
		if err := client.SendChatMessage(input, ""); err != nil {
			fmt.Println("[错误] 发送消息失败:", err)
		}
		// 非阻塞检查 quit（若 quit 已关闭则返回）
		select {
		case <-client.quit:
			return
		default:
		}
	}
}
