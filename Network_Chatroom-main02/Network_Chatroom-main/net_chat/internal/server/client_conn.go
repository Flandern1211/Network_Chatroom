package server

//客户端
import (
	"bufio"
	"fmt"
	"net"
	"net_chat/internal/protocol"
)

type ClientConn struct {
	Conn     net.Conn               //维护的连接
	Name     string                 //用户的姓名
	Outgoing chan *protocol.Message //只用于服务器发给客户端的消息队列
	quit     chan struct{}          //用于通知对应协程退出
}

// 构造函数，每次有新用户都直接使用构造函数来创建新连接
func NewClientConn(conn net.Conn) *ClientConn {
	return &ClientConn{
		Conn:     conn,
		Outgoing: make(chan *protocol.Message, 32),
		quit:     make(chan struct{}),
	}
}

// 维持两个协程readLoop和writeLoop
func (c *ClientConn) Start(s *Server) {
	go c.readLoop(s)
	go c.writeLoop()
}

// 从客户端读取消息并交由server.Dispatch处理
func (c *ClientConn) readLoop(s *Server) {
	r := bufio.NewReader(c.Conn)

	for {
		//读取消息
		msg, err := protocol.ReadMsg(r)
		//处理连接异常断开
		//如果无法从客户端读取消息，则代表客户端断开连接了，直接把该用户从用户列表中删除
		if err != nil {
			if c.Name != "" {
				s.RemoveUser(c.Name)
				s.Broadcast(&protocol.Message{
					Type:    "notice",
					Content: fmt.Sprintf("%s 已从聊天室中离开", c.Name),
					From:    "system",
				})
			}
			c.Close()
			return
		}
		//得到消息结构体，交由服务端处理
		s.Dispatch(msg, c)
	}
}

// 向客户端写
func (c *ClientConn) writeLoop() {
	for {
		select {
		case msg, ok := <-c.Outgoing: // 从发送队列读取一条消息
			if !ok {
				// 如果通道被关闭（Close 调用），直接退出写协程
				return
			}
			// 将消息编码并写入底层 conn（可能阻塞直到写完或出错）
			if err := protocol.SendMsg(c.Conn, msg); err != nil {
				return
			}
		case <-c.quit: // 收到退出信号则结束写协程
			return

		}
	}
}

func (c *ClientConn) Close() error {
	close(c.quit) // 通知 writeLoop 退出
	return c.Conn.Close()
}
