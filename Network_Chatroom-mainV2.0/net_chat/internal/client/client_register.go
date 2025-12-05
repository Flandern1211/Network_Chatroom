package client

import (
	"fmt"
	"net_chat/internal/protocol"
	"strings"
)

// Register 注册:
func (c *Client) Register(inputLines <-chan string) error {
	for {
		fmt.Print("请输入要注册的用户名和密码(格式为”用户名|密码)（输入exit退出注册界面）:")
		//1. 从用户输入通道中获取用户输入
		userinfo, ok := <-inputLines
		if !ok {
			return fmt.Errorf("无法得到用户输入（输入通道已关闭）")
		}
		//2.让用户可以退出注册界面
		if userinfo == "exit" {
			return nil
		}
		//3.  用户名|密码初步校验
		if err := validateUserinfo(userinfo); err != nil {
			fmt.Println("用户名错误：", err)
			continue // 校验失败，重新输入
		}

		//3.2分别验证用户名和密码
		parts := strings.Split(userinfo, "|")
		if len(parts) != 2 {
			fmt.Println("格式错误，请使用‘用户名|密码’的格式")
			continue
		}
		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])

		if username == "" || password == "" {
			fmt.Println("用户名和密码不能为空")
			continue
		}

		//4. 在数据库中检查
		err := protocol.SendMsg(c.conn, &protocol.Message{
			Type:    "register",
			Content: userinfo,
		})
		if err != nil {
			return fmt.Errorf("发送注册请求失败%w", err)
		}

		select {
		case msg := <-c.msgChan:
			if msg.Type == "register_success" {
				fmt.Println(msg.Content) // 欢迎信息
				return nil               // 注册成功，退出循环
			} else if msg.Type == "register_fail" {
				fmt.Println("注册失败：", msg.Content)
				// 不退出循环，重新获取用户名和密码
			} else {
				// 既不是 login_success 也不是 login_fail，打印并继续等待下一次输入
				fmt.Println("收到非登录类型消息（忽略）:", msg.Type)
			}
		case <-c.quit:
			return fmt.Errorf("客户端已退出")
		}
	}
}
