package client

import (
	"fmt"
	"strings"
)

func (c *Client) LoginRegisterMenu(inputLines <-chan string) {
	for {
		fmt.Println("1.登录")
		fmt.Println("2.注册")
		line, ok := <-inputLines
		if !ok {
			// stdin 关闭或 goroutine 结束
			return
		}
		choice := strings.TrimSpace(line)
		fmt.Printf("[调试 ] 接收到的输入: '%s' (长度: %d)\n", choice, len(choice))
		// 处理空输入（用户直接按回车）
		if choice == "" {
			fmt.Println("请输入有效数字（1-2）")
			continue
		}
		switch choice {
		case "1":
			// 调用 Login
			if err := c.Login(inputLines); err != nil {
				fmt.Println("登录失败：", err)
				continue
			}
			// 登录成功后启动 handleMessages
			c.Start(false)
		case "2":
			// 原来这里调用的是 showPrivateChat，应该调用 Register
			if err := c.Register(inputLines); err != nil {
				fmt.Println("注册失败：", err)
				continue
			} else {
				continue
			}
		default:
			fmt.Println("无效选择，请重新输入")
		}
		if choice == "1" && c.username != "" {
			break
		}

	}
}
