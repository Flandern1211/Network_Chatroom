package main

import (
	"bufio"
	"fmt"
	"net"
	"net_chat/internal/protocol"
	"os"
	"strings"
	"sync"
	"time"
)

// 客户端
type Client struct {
	conn     net.Conn               //维护的连接
	username string                 //用户名
	msgChan  chan *protocol.Message //客户端自己维护的消息队列
	quit     chan struct{}          //退出信号
	wg       sync.WaitGroup         //协程控制组
	once     sync.Once              //原子操作，保证管道只会关闭一次，不会多次关闭引发panic
	scanner  *bufio.Scanner
}

// 构造函数
func NewClient() *Client {
	return &Client{
		msgChan: make(chan *protocol.Message, 32),
		quit:    make(chan struct{}),
		once:    sync.Once{},
		scanner: bufio.NewScanner(os.Stdin),
	}

}

// 建立连接
func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// 客户端协程启动
func (c *Client) Start(skipHandle bool) { // 新增参数控制是否启动handleMessages
	c.wg.Add(1) // 只启动readLoop
	go c.readLoop()
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
				c.once.Do(func() {
					close(c.quit)
				})
				return
			}
			c.msgChan <- msg
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

// 具体处理方法
func (c *Client) processMessages(msg *protocol.Message) {
	switch msg.Type {
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
		fmt.Println("用户在线列表")
		//服务端将用户用|隔开，这里用|作为每个用户的标志
		users := strings.Split(msg.Content, "|")
		if len(users) == 1 && users[0] == "" {
			fmt.Println("当前没有其他用户在线")
		} else {
			for i, user := range users {
				fmt.Printf("%d. %s\n", i+1, user)
			}

		}
	case "logout_success":
		//用户退出后关闭所有客户端协程
		fmt.Println(msg.Content)
		close(c.quit)
	}
}

// 登录
func (c *Client) Login() error {
	for {
		fmt.Print("请输入您的姓名：")
		if !c.scanner.Scan() {
			return fmt.Errorf("无法得到用户输入")
		}
		username := strings.TrimSpace(c.scanner.Text())
		// 用户名校验
		if err := validateUsername(username); err != nil {
			fmt.Println("用户名错误：", err)
			continue // 校验失败，重新输入
		}

		// 发送登录请求
		err := protocol.SendMsg(c.conn, &protocol.Message{
			Type:    "login",
			Content: username,
		})
		if err != nil {
			return fmt.Errorf("发送登录请求失败：%v", err)
		}

		// 等待服务器响应
		select {
		case msg := <-c.msgChan:
			if msg.Type == "login_success" {
				c.username = username
				fmt.Println(msg.Content) // 欢迎信息
				return nil               // 登录成功，退出循环
			} else if msg.Type == "login_fail" {
				fmt.Println("登录失败：", msg.Content)
				// 不退出循环，重新获取用户名
			}
		}
	}
}

// 用户名校验函数
func validateUsername(name string) error {
	if name == "" {
		return fmt.Errorf("不能为空")
	}
	if strings.Contains(name, " ") {
		return fmt.Errorf("不能包含空格")
	}
	if name == "exit" {
		return fmt.Errorf("不能为'exit'")
	}
	return nil
}

// 发送聊天消息
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

// 查看用户列表
func (c *Client) RequestUserList() error {
	msg := &protocol.Message{
		Type: "list",
	}
	return protocol.SendMsg(c.conn, msg)
}

// 登出
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

// 关闭客户端
func (c *Client) Close() {
	c.once.Do(func() {
		close(c.quit)
	})
	if c.conn != nil {
		c.conn.Close()
	}
	c.wg.Wait()
}

// 在聊天室中发送消息
func showChatRoom(client *Client) {
	fmt.Println("\n======= 在聊天室中发送消息 =======")
	fmt.Println("输入消息并按回车发送，输入 'exit' 退出聊天室")
	//启动一个协程来读取用户输入
	//保证在等待用户输入的时候可以正常接收消息
	inputChan := make(chan string)
	go func() {
		//循环等待用户的输入,得到用户的输入就传入inputChan
		for client.scanner.Scan() {
			inputChan <- client.scanner.Text()
		}
		close(inputChan)
	}()

	//循环处理消息和用户输入
	for {
		select {
		case <-client.quit:
			return
		case input := <-inputChan:
			if input == "exit" {
				return
			}
			if input != "" {
				client.SendChatMessage(input, "")
			}
		case msg := <-client.msgChan:
			client.processMessages(msg)

		}
	}
}

// 私聊
func showPrivateChat(client *Client) {
	fmt.Println("\n======= 私聊 =======")
	fmt.Print("请输入对方用户名: ")

	client.scanner.Scan()
	targetUser := client.scanner.Text()
	if targetUser == "" || targetUser == "exit" {
		fmt.Println("目标用户名不能为空且不能为'exit'")
		return
	}

	fmt.Printf("与 %s 私聊中，输入消息并按回车发送，输入 'exit' 退出私聊\n", targetUser)

	//启动一个协程来读取用户输入

	inputChan := make(chan string)
	go func() {
		for client.scanner.Scan() {
			inputChan <- client.scanner.Text()
		}
		close(inputChan)
	}()

	//循环处理消息和用户输入
	for {
		select {
		case <-client.quit:
			return
		case input := <-inputChan:
			if input == "exit" {
				return
			}
			if input != "" {
				client.SendChatMessage(input, targetUser)
			}
		case msg := <-client.msgChan:
			client.processMessages(msg)
		}
	}

}

func main() {
	client := NewClient()
	//记得关闭连接和所有协程
	defer client.Close()

	//连接服务器
	addr := "localhost:8080"
	if err := client.Connect(addr); err != nil {
		fmt.Printf("连接服务器失败%v\n", err)
		return
	}

	//
	client.Start(true)
	// 登录流程（带重试）
	if err := client.Login(); err != nil {
		fmt.Println("登录失败：", err)
		client.Close()
		return
	}
	//启动客户端循环处理
	client.Start(false)
	//主菜单循环
	for {
		fmt.Println("\n======= 聊天室  =======")
		fmt.Println("1. 在聊天室中发送消息")
		fmt.Println("2. 私聊")
		fmt.Println("3. 显示在线用户列表")
		fmt.Println("4. 退出")
		fmt.Print("请选择操作: ")

		if !client.scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(client.scanner.Text())
		fmt.Printf("[调试] 接收到的输入: '%s' (长度: %d)\n", choice, len(choice))
		// 处理空输入（用户直接按回车）
		if choice == "" {
			fmt.Println("请输入有效数字（1-4）")
			continue
		} // 去前后空格
		switch choice {
		case "1":
			showChatRoom(client)
		case "2":
			showPrivateChat(client)
		case "3":
			client.RequestUserList()
			time.Sleep(500 * time.Millisecond)
		case "4":
			client.Logout()
			<-client.quit
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}

	}

}
