package client

import (
	"fmt"
	"net_chat/internal/protocol"
	"strings"
)

// Login 登录
func (c *Client) Login(inputLines <-chan string) error {
	for {
		fmt.Print("请输入您的用户名和密码(格式为用户名|密码)(输入exit退出登录界面)：")
		userinfo, ok := <-inputLines
		if !ok {
			return fmt.Errorf("无法得到用户输入（输入通道已关闭）")
		}
		if userinfo == "exit" {
			return nil
		}

		//username = strings.TrimSpace(username)
		// 用户名校验
		if err := validateUserinfo(userinfo); err != nil {
			fmt.Println("用户名错误：", err)
			continue // 校验失败，重新输入
		}

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

		// 发送登录请求
		err := protocol.SendMsg(c.conn, &protocol.Message{
			Type:    "login",
			Content: userinfo,
		})
		if err != nil {
			return fmt.Errorf("发送登录请求失败：%v", err)
		}

		// 等待服务器响应（保留你原先的方式：直接从 c.msgChan 读取）
		select {
		case msg := <-c.msgChan:
			if msg.Type == "login_success" {
				c.username = username
				fmt.Println(msg.Content) // 欢迎信息
				return nil               // 登录成功，退出循环
			} else if msg.Type == "login_fail" {
				fmt.Println("登录失败：", msg.Content)
				// 不退出循环，重新获取用户名
			} else {
				// 既不是 login_success 也不是 login_fail，打印并继续等待下一次输入
				fmt.Println("收到非登录类型消息（忽略）:", msg.Type)
			}
		case <-c.quit:
			return fmt.Errorf("客户端已退出")
		}
	}
}

// 简单的用户名和密码校验函数
func validateUserinfo(userinfo string) error {
	if userinfo == "" {
		return fmt.Errorf("不能为空")
	}
	if strings.Contains(userinfo, " ") {
		return fmt.Errorf("不能包含空格")
	}
	if userinfo == "exit" {
		return fmt.Errorf("不能为'exit'")
	}
	return nil
}
