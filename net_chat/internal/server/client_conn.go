package server

//客户端
import (
	"bufio"
	"net"
	"net_chat/internal/protocol"
)

type ClientConn struct {
	Conn      net.Conn               //维护的连接
	Name      string                 //用户的姓名
	Outgoing  chan *protocol.Message //服务器发给客户端的消息队列
	privateTo string                 //私聊对象
	quit      chan struct{}          //用于通知对应协程退出
}

// 构造函数，每次有新用户都直接使用构造函数来创建新连接
func NewClientConn(conn net.Conn) *ClientConn {
	return &ClientConn{
		Conn:     conn,
		Outgoing: make(chan *protocol.Message, 32),
		quit:     make(chan struct{}),
	}
}

// 客户端维持两个协程readLoop和writeLoop
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
		if err != nil {
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
				// 写出错（对端断开或网络问题），退出写协程让上层清理
				return
			}
		case <-c.quit: // 收到退出信号则结束写协程
			return
		}
	}
}

// 确定私聊模式
func (c *ClientConn) SetPrivatemode(name string) {
	c.privateTo = name
}

// 设置为群聊模式
func (c *ClientConn) SetBroadmode() {
	c.privateTo = ""
}

func (c *ClientConn) Close() error {
	close(c.quit)
	close(c.Outgoing)
	return c.Conn.Close()
}
