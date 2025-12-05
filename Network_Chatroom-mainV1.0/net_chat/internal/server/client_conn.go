package server

//服务端维持的与客户端的连接
//主要功能:循环从客户端读和写
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

		///////////////////////修改////////////////////////////////
		//读取信息失败分为两种情况，一种是用户自己退出,用户表中不存在该用户;一种是该用户非法断连，这时用户表中还存在该用户，但因为连接不上可以先
		//发送该用户非法中断的消息再将该用户从表中删除

		if err != nil {
			if c.Name != "" {
				s.RemoveUser(c.Name)
				s.Broadcast(&protocol.Message{
					Type:    "notice",
					Content: fmt.Sprintf("%s 已从聊天室中离开", c.Name),
					From:    "system",
				})
			}
			//连接异常，发送错误信息并中断该连接
			fmt.Printf("无法从与%s连接中读取到数据\n", c.Name)
			c.Close()
			return
		}
		//得到消息结构体，交由服务端处理
		s.Dispatch(msg, c)
	}
}

// 循环向客户端写
func (c *ClientConn) writeLoop() {
	for {
		//select阻塞，循环监听消息队列和退出信号，确保每个协程优雅退出
		select {
		case msg, ok := <-c.Outgoing: // 从发送队列读取一条消息
			if !ok {
				// 如果通道被关闭（Close 调用），直接退出写协程
				fmt.Println("无法从消息队列获取消息")
				return
			}
			// 将消息编码并写入底层 conn（可能阻塞直到写完或出错）
			if err := protocol.SendMsg(c.Conn, msg); err != nil {
				fmt.Println("发送消息失败:", err)
				return
			}
		case <-c.quit: // 收到退出信号则结束写协程
			return

		}
	}
}

// 关闭连接，结束协程
func (c *ClientConn) Close() error {
	close(c.quit) // 通知 writeLoop 退出
	return c.Conn.Close()
}
