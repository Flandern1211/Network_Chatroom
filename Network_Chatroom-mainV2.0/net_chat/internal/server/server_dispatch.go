package server

//统一处理消息方法
import (
	"net_chat/internal/protocol"
)

func (s *Server) Dispatch(msg *protocol.Message, c *ClientConn) {
	switch msg.Type {
	//处理注册请求
	case "register":
		s.HandleRegister(msg, c)
	//处理登录请求
	case "login":
		s.HandleLogin(msg, c)
	case "privatebegin":
		s.sendRecentPrivateMessages(c, msg.To)
	//发送消息请求
	case "chat":
		s.HandleChat(msg, c)
	//查看用户列表请求
	case "list":
		s.HandleList(c)

	case "activityDay":
		s.Handleactivityday(c)
	case "activityWeek":
		s.Handleactivityweek(c)
	case "activityTotal":
		s.Handleactivitytotal(c)
	case "room_messages":
		s.sendRecentRoomMessages(c)
	//用户登出请求
	case "logout":
		s.HandleLogout(c)
	}
}
