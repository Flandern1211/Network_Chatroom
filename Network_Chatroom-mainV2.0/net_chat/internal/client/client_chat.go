package client

import "net_chat/internal/protocol"

// SendChatMessage 发送聊天消息
func (c *Client) SendChatMessage(content string, to string) error {
	msg := &protocol.Message{
		Type:    "chat",
		Content: content,
		From:    c.username,
		To:      to,
	}
	err := protocol.SendMsg(c.conn, msg)
	if err != nil {
		return err
	}
	return nil
}
