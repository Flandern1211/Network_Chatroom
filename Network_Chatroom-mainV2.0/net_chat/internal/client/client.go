package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net_chat/internal/protocol"
	"os"
	"sync"
)

// Client 客户端
type Client struct {
	conn       net.Conn               //维护的连接
	username   string                 //用户名
	msgChan    chan *protocol.Message //客户端自己维护的消息队列，用于在读取和处理消息协程之间的通信
	quit       chan struct{}          //退出信号
	wg         sync.WaitGroup         //协程控制组
	once       sync.Once              //原子操作，保证管道只会关闭一次，不会多次关闭引发panic
	startOnce  sync.Once              //保证
	users      []string
	InputLines chan string
}

var (
	done chan struct{}
)

// NewClient 构造函数
func NewClient() *Client {
	return &Client{
		msgChan:    make(chan *protocol.Message, 32),
		quit:       make(chan struct{}),
		once:       sync.Once{},
		InputLines: make(chan string, 4), // 启动唯一的 stdin 读取 goroutine，统一写到 inputLines
	}

}

// Connect 建立连接
func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("无法与服务端建立连接", err)
		return err
	}

	c.conn = conn
	return nil
}

// Start 客户端协程启动
// 注意：保持与原来一致——Start(true) 会只启动 readLoop，Start(false) 会同时启动 readLoop 与 handleMessages。
// （避免大范围重构以满足“别全部改”的要求）
// 客户端协程启动
func (c *Client) Start(skipHandle bool) {
	go c.Input()
	// 保证 readLoop 只会启动一次，即使外面调用了多次 Start(...)
	c.startOnce.Do(func() {
		c.wg.Add(1)
		go c.readLoop()
	})

	// handleMessages 可以按需启动（例如登录后才启动）
	if !skipHandle {
		c.wg.Add(1)
		go c.handleMessages()
	}
}

// 循环读取从conn获取的消息
func (c *Client) readLoop() {
	defer c.wg.Done()
	reader := bufio.NewReader(c.conn)
	for {
		select {
		case <-c.quit:
			return
		default:
			msg, err := protocol.ReadMsg(reader)
			if err != nil {
				fmt.Println("读取消息错误", err)
				//用once.Do保证quit只关闭一次
				c.once.Do(func() {
					close(c.quit)
				})
				return
			}
			//readloop协程循环读取消息,然后交给handleMessage协程处理，两个协程之间用管道传输，保证消息的顺序不变
			// 这里写入 msgChan（阻塞写），若quit已关闭，则退出
			select {
			case c.msgChan <- msg:
			case <-c.quit:
				return
			}
		}
	}
}

// 处理获取的消息
func (c *Client) handleMessages() {
	defer c.wg.Done()
	for {
		select {
		case <-c.quit:
			return
		case msg := <-c.msgChan:
			c.processMessages(msg)
		}
	}
}

// Input 统一处理用户的输入
func (c *Client) Input() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		// 将行送进 channel（阻塞写入直到被读取或缓冲）
		c.InputLines <- line
	}
	// scanner 结束 (EOF)，关闭 channel
	close(c.InputLines)

}

// Logout 登出
func (c *Client) Logout() error {
	msg := &protocol.Message{
		Type: "logout",
	}
	err := protocol.SendMsg(c.conn, msg)
	//登出要通知所有协程结束，用once来保证只关闭一次quit
	c.once.Do(func() {
		close(c.quit)
	})
	return err
}

// Close 关闭客户端
func (c *Client) Close() {
	c.once.Do(func() {
		close(c.quit)
	})
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			fmt.Printf("关闭连接失败:%s", err)
		}
	}
	c.wg.Wait()
}
