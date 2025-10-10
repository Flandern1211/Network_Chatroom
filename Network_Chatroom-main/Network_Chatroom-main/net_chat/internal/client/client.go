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

type Client struct {
	conn     net.Conn
	username string
	msgChan  chan *protocol.Message
	quit     chan struct{}
	wg       sync.WaitGroup
	once     sync.Once
	scanner  *bufio.Scanner
}

func NewClient() *Client {
	return &Client{
		msgChan: make(chan *protocol.Message, 32),
		quit:    make(chan struct{}),
		once:    sync.Once{},
		scanner: bufio.NewScanner(os.Stdin),
	}

}

func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *Client) Start(skipHandle bool) { // 新增参数控制是否启动handleMessages
	c.wg.Add(1) // 只启动readLoop
	go c.readLoop()
	if !skipHandle {
		c.wg.Add(1)
		go c.handleMessages()
	}
}

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

func (c *Client) processMessages(msg *protocol.Message) {
	switch msg.Type {
	case "login_success":
		fmt.Println(msg.Content)
	case "login_fail":
		fmt.Println("登录失败", msg.Content)
	case "notice":
		fmt.Println("系统通知", msg.Content)
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
		users := strings.Split(msg.Content, "|")
		if len(users) == 1 && users[0] == "" {
			fmt.Println("当前没有其他用户在线")
		} else {
			for i, user := range users {
				fmt.Printf("%d. %s\n", i+1, user)
			}

		}
	case "logout_success":
		fmt.Println(msg.Content)
		close(c.quit)
	}
}

func (c *Client) Login() error {
	for {
		fmt.Print("请输入您的姓名：")
		if !c.scanner.Scan() {
			return fmt.Errorf("无法得到用户输入")
		}
		username := strings.TrimSpace(c.scanner.Text())

		// 客户端初步校验
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
				fmt.Println(msg.Content) // 显示欢迎信息
				return nil               // 登录成功，退出循环
			} else if msg.Type == "login_fail" {
				fmt.Println("登录失败：", msg.Content)
				// 不退出循环，继续重试
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("登录超时，请重试")
		}
	}
}

// 用户名本地校验函数
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
	if len(name) > 32 {
		return fmt.Errorf("长度不能超过32个字符")
	}
	return nil
}

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

func (c *Client) RequestUserList() error {
	msg := &protocol.Message{
		Type: "list",
	}
	return protocol.SendMsg(c.conn, msg)
}

func (c *Client) Logout() error {
	msg := &protocol.Message{
		Type: "logout",
	}
	err := protocol.SendMsg(c.conn, msg)
	c.once.Do(func() {
		close(c.quit)
	})
	return err
}

func (c *Client) Close() {
	c.once.Do(func() {
		close(c.quit)
	})
	if c.conn != nil {
		c.conn.Close()
	}
	c.wg.Wait()
}

func showMainMenu() {
	fmt.Println("\n======= 聊天室  =======")
	fmt.Println("1. 在聊天室中发送消息")
	fmt.Println("2. 私聊")
	fmt.Println("3. 显示在线用户列表")
	fmt.Println("4. 退出")
	fmt.Print("请选择操作: ")
}

func showChatRoom(client *Client) {
	fmt.Println("\n======= 在聊天室中发送消息 =======")
	fmt.Println("输入消息并按回车发送，输入 'exit' 退出聊天室")
	//启动一个协程来读取用户输入
	inputChan := make(chan string)
	go func() {

		for client.scanner.Scan() {
			inputChan <- client.scanner.Text()
		}
	}()

	defer func() {
		// 清空缓冲区（如果有未读取的内容）
		for client.scanner.Scan() {
			if len(strings.TrimSpace(client.scanner.Text())) == 0 {
				break
			}
		}
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
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputChan <- scanner.Text()
		}
	}()

	defer func() {
		for client.scanner.Scan() {
			if len(strings.TrimSpace(client.scanner.Text())) == 0 {
				break
			}
		}
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
	defer client.Close()

	//连接服务器
	addr := "localhost:8080"
	if err := client.Connect(addr); err != nil {
		fmt.Printf("连接服务器失败%v\n", err)
		return
	}
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
		showMainMenu()

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
