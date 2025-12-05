package client

import (
	"fmt"
	"net_chat/internal/protocol"
)

func (c *Client) RequestActivityRanking(inputLines <-chan string) error {
	fmt.Println("\n======= 活跃度排行  =======")
	fmt.Println("1. 日排行")
	fmt.Println("2. 周排行")
	fmt.Println("3. 总排行")
	fmt.Print("请选择排行类型: ")
	rankType, ok := <-inputLines
	if !ok {
		return fmt.Errorf("无法得到用户输入")
	}
	var msgType string
	switch rankType {
	case "1":
		msgType = "activityDay"
	case "2":
		msgType = "activityWeek"
	case "3":
		msgType = "activityTotal"
	default:
		return fmt.Errorf("无效的排行类型:%s", rankType)
	}

	msg := &protocol.Message{
		Type: msgType,
	}
	return protocol.SendMsg(c.conn, msg)
}
