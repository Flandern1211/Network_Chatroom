package client

import (
	"fmt"
	"net_chat/internal/protocol"
	"strings"
)

// 具体处理方法
func (c *Client) processMessages(msg *protocol.Message) {
	switch msg.Type {
	case "register_success":
		fmt.Println(msg.Content)
	case "register_fail":
		fmt.Println("注册失败", msg.Content)
	case "login_success":
		fmt.Println(msg.Content)
	case "login_fail":
		fmt.Println("登录失败", msg.Content)
	case "notice":
		fmt.Println("\n系统通知", msg.Content)
	case "chat":
		fmt.Printf("[%s]:%s\n", msg.From, msg.Content)
	case "private_chat":
		fmt.Printf("[私聊][%s]:%s\n", msg.From, msg.Content)
	case "private_chat_sent":
		fmt.Println("[系统]:", msg.Content)
	case "error":
		fmt.Println("[错误]", msg.Content)
	case "user_list":
		//服务端将用户用|隔开，这里用|作为每个用户的标志
		c.users = strings.Split(msg.Content.(string), "|")
		close(done)
	case "activityday":
		fmt.Println("日榜")
		fmt.Println(msg.Content)
	case "activityweek":
		fmt.Println("周榜")
		fmt.Println(msg.Content)
	case "activitytotal":
		fmt.Println("总榜")
		fmt.Println(msg.Content)
	case "recent_room_messages":
		fmt.Println("最近聊天记录(输入exit退出查看):")
		fmt.Println(msg.Content)
	case "recent_private_messages":
		fmt.Println(msg.Content)
	case "logout_success":
		//用户退出后关闭所有客户端协程
		fmt.Println(msg.Content)
		// 用 once 保证只关闭一次 quit
		c.once.Do(func() {
			close(c.quit)
		})
	default:
		// 未处理的消息类型，打印调试信息
		fmt.Printf("[未知消息类型 %s] %s\n", msg.Type, msg.Content)
	}
}
