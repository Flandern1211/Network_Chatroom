package main

import (
	"bufio"
	"fmt"
	"net"
	"net_chat/internal/protocol"
	"os"
	"strings"
	"sync"
)

type Client struct {
	conn     net.Conn
	username string
	msgChan  chan *protocol.Message
	quit     chan struct{}
	wg       sync.WaitGroup
}

func NewClient() *Client {
	return &Client{
		msgChan: make(chan *protocol.Message, 32),
		quit:    make(chan struct{}),
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

func (c *Client) Start() {
	c.wg.Add(2)
	go c.readLoop()
	go c.handleMessages()
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
				close(c.quit)
				select {
				case <-c.quit:
					// channel已关闭
				default:
					close(c.quit)
				}
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

func (c *Client) Login(username string) error {
	c.username = username
	msg := &protocol.Message{
		Type:    "login",
		Content: username,
	}
	err := protocol.SendMsg(c.conn, msg)
	if err != nil {
		return err
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
	select {
	case <-c.quit:
		// 通道已经关闭
	default:
		close(c.quit)
	}
	return err
}

func (c *Client) Close() {
	select {
	case <-c.quit:
	default:
		close(c.quit)
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.wg.Wait()
}

func showMainMenu() {
	fmt.Println("\n======= 聊天室主菜单 =======")
	fmt.Println("1. 加入聊天室")
	fmt.Println("2. 私聊")
	fmt.Println("3. 显示在线用户列表")
	fmt.Println("4. 退出")
	fmt.Print("请选择操作: ")
}

func showChatRoom(client *Client) {
	fmt.Println("\n======= 聊天室 =======")
	fmt.Println("输入消息并按回车发送，输入 'exit' 退出聊天室")
	//启动一个协程来读取用户输入
	inputChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputChan <- scanner.Text()
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
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	targetUser := scanner.Text()
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

	//启动客户端循环处理
	client.Start()

	//登录
	fmt.Print("请输入您的姓名：")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()

	if err := client.Login(username); err != nil {
		fmt.Println("登录请求发送失败")
		return
	}

	//等待登录
	select {
	case msg := <-client.msgChan:
		if msg.Type == "login_success" {
			client.processMessages(msg)
		} else {
			client.processMessages(msg)
		}
	case <-client.quit:
		return
	}

	//主菜单循环
	for {
		showMainMenu()
		menuScanner := bufio.NewScanner(os.Stdin)
		if !menuScanner.Scan() {
			break
		}
		choice := menuScanner.Text()
		switch choice {
		case "1":
			showChatRoom(client)
		case "2":
			showPrivateChat(client)
		case "3":
			client.RequestUserList()
			select {
			case msg := <-client.msgChan:
				client.processMessages(msg)
			case <-client.quit:
				return
			}
		case "4":
			client.Logout()
			fmt.Println("已退出聊天室")
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}

	}

}
