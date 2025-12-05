package server

import (
	"net_chat/internal/protocol"
	"strings"
)

func (s *Server) HandleList(c *ClientConn) {
	users := s.ListUsers()
	c.Outgoing <- &protocol.Message{
		Type: "user_list",
		//用|隔开
		Content: strings.Join(users, "|"),
		From:    "system",
	}
}

// ListUsers 返回用户列表
func (s *Server) ListUsers() []string {
	s.mu.Lock()
	//用户列表切片
	var out []string
	for n := range s.users {
		out = append(out, n)
	}
	s.mu.Unlock()
	return out
}
