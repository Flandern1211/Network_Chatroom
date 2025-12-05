package client

import (
	"fmt"
	"strings"
)

func (c *Client) MainMenu(inputLines <-chan string) {
	for {
		fmt.Println("\n======= 聊天室  =======")
		fmt.Println("1. 在聊天室中发送消息")
		fmt.Println("2. 私聊")
		fmt.Println("3. 显示在线用户列表")
		fmt.Println("4. 查看活跃度排行")
		fmt.Println("5. 查看聊天室的最近消息")
		fmt.Println("6. 退出")
		fmt.Print("请选择操作: ")

		line, ok := <-inputLines
		if !ok {
			// stdin 关闭或 goroutine 结束
			return
		}
		choice := strings.TrimSpace(line)
		fmt.Printf("[调试 ] 接收到的输入: '%s' (长度: %d)\n", choice, len(choice))
		// 处理空输入（用户直接按回车）
		if choice == "" {
			fmt.Println("请输入有效数字（1-4）")
			continue
		} // 去前后空格
		switch choice {
		case "1":
			ShowChatRoom(c, inputLines)
		case "2":
			ShowPrivateChat(c, inputLines)
		case "3":
			err := c.RequestUserList(inputLines)
			if err != nil {
				fmt.Println("请求用户列表时发生错误:", err)
			}
		case "4":
			err := c.RequestActivityRanking(inputLines)
			if err != nil {
				fmt.Println("请求活跃度排名时发生错误:", err)
			}
		case "5":
			if err := c.RequestRecentMessages(inputLines); err != nil {
				fmt.Println("[错误]请求历史消息失败", err)
			}
		case "6":
			err := c.Logout()
			if err != nil {
				fmt.Println("无法正常退出", err)
				return
			}
			<-c.quit
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}

	}
}
