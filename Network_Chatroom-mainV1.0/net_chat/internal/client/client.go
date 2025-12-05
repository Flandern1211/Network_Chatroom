package main

import (
	"bufio"
	"fmt"
	"log"
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
	msgChan  chan *protocol.Message //客户端自己维护的消息队列，用于在读取和处理消息协程之间的通信
	quit     chan struct{}          //退出信号
	wg       sync.WaitGroup         //协程控制组
	once     sync.Once              //原子操作，保证管道只会关闭一次，不会多次关闭引发panic
}

// 构造函数
func NewClient() *Client {
	return &Client{
		msgChan: make(chan *protocol.Message, 32),
		quit:    make(chan struct{}),
		once:    sync.Once{},
	}

}

// 建立连接
func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("无法与服务端建立连接", err)
		return err
	}

	c.conn = conn
	return nil
}

// 客户端协程启动
// 注意：保持与原来一致——Start(true) 会只启动 readLoop，Start(false) 会同时启动 readLoop 与 handleMessages。
// （避免大范围重构以满足“别全部改”的要求）
func (c *Client) Start(skipHandle bool) {
	c.wg.Add(1) // 启动 readLoop
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
		// 用 once 保证只关闭一次 quit
		c.once.Do(func() {
			close(c.quit)
		})
	default:
		// 未处理的消息类型，打印调试信息
		fmt.Printf("[未知消息类型 %s] %s\n", msg.Type, msg.Content)
	}
}

// 登录：现在从统一的 inputLines 读取（阻塞读取），不再在 Login 内创建 Scanner。
// 保持等待 c.msgChan 的原始逻辑（最小改动）
func (c *Client) Login(inputLines <-chan string) error {
	for {
		fmt.Print("请输入您的姓名：")
		username, ok := <-inputLines
		if !ok {
			return fmt.Errorf("无法得到用户输入（输入通道已关闭）")
		}
		username = strings.TrimSpace(username)

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

		// 等待服务器响应（保留你原先的方式：直接从 c.msgChan 读取）
		select {
		case msg := <-c.msgChan:
			if msg.Type == "login_success" {
				c.username = username
				fmt.Println(msg.Content) // 欢迎信息
				return nil               // 登录成功，退出循环
			} else if msg.Type == "login_fail" {
				fmt.Println("登录失败：", msg.Content)
				// 不退出循环，重新获取用户名
			} else {
				// 既不是 login_success 也不是 login_fail，打印并继续等待下一次输入
				fmt.Println("收到非登录类型消息（忽略）:", msg.Type)
			}
		case <-c.quit:
			return fmt.Errorf("客户端已退出")
		}
	}
}

// 简单的用户名校验函数
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

// 聊天室中处理和发送消息
// 现在同步、阻塞地从 inputLines 读取（由 main 的唯一 stdin goroutine 提供）
// 不再在这里创建 Scanner 或额外 goroutine，也不从 client.msgChan 读取（handleMessages 负责显示）
func showChatRoom(client *Client, inputLines <-chan string) {
	fmt.Println("\n======= 在聊天室中发送消息 =======")
	fmt.Println("输入消息并按回车发送，输入 'exit' 退出聊天室")

	for {
		// 直接阻塞读取下一行（在任意时刻，只有 main 路径在阻塞读 inputLines）
		line, ok := <-inputLines
		if !ok {
			// stdin 已关闭
			return
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" {
			return
		}
		if err := client.SendChatMessage(input, ""); err != nil {
			fmt.Println("[错误] 发送消息失败:", err)
		}
		// 非阻塞检查 quit（若 quit 已关闭则返回）
		select {
		case <-client.quit:
			return
		default:
		}
	}
}

// 私聊
// 也同步阻塞读取 inputLines，先读取目标用户名，再读取消息
func showPrivateChat(client *Client, inputLines <-chan string) {
	fmt.Println("\n======= 私聊 =======")
	fmt.Print("请输入对方用户名: ")
	targetUser, ok := <-inputLines
	if !ok {
		return
	}
	targetUser = strings.TrimSpace(targetUser)
	if targetUser == "" || targetUser == "exit" {
		fmt.Println("目标用户名不能为空且不能为'exit'")
		return
	}

	fmt.Printf("与 %s 私聊中，输入消息并按回车发送，输入 'exit' 退出私聊\n", targetUser)

	for {
		line, ok := <-inputLines
		if !ok {
			return
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "exit" {
			return
		}
		if err := client.SendChatMessage(input, targetUser); err != nil {
			fmt.Println("[错误] 发送失败，", err)
		}
		// 非阻塞检查 quit（若 quit 已关闭则返回）
		select {
		case <-client.quit:
			return
		default:
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

	// 启动唯一的 stdin 读取 goroutine，统一写到 inputLines
	inputLines := make(chan string, 4) // 小缓冲
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			// 将行送进 channel（阻塞写入直到被读取或缓冲）
			inputLines <- line
		}
		// scanner 结束 (EOF)，关闭 channel
		close(inputLines)
	}()

	// 启动客户端：只启动 readLoop（保持你原来的调用习惯）
	client.Start(true)

	// 登录流程（带重试），传入 inputLines
	if err := client.Login(inputLines); err != nil {
		fmt.Println("登录失败：", err)
		client.Close()
		return
	}

	// 登录成功后启动 handleMessages（如果需要）；保持你原来的语义
	client.Start(false)

	//主菜单循环（从 inputLines 阻塞读取）
	for {
		fmt.Println("\n======= 聊天室  =======")
		fmt.Println("1. 在聊天室中发送消息")
		fmt.Println("2. 私聊")
		fmt.Println("3. 显示在线用户列表")
		fmt.Println("4. 退出")
		fmt.Print("请选择操作: ")

		line, ok := <-inputLines
		if !ok {
			// stdin 关闭或 goroutine 结束
			return
		}
		choice := strings.TrimSpace(line)
		fmt.Printf("[调试 ] 接收到的输入: '%s' (长度: %d)\n", choice, len(choice))
		// 处理空输入（用户直接按回车）
		if choice == "" {
			fmt.Println("请输入有效数字（1-4）")
			continue
		} // 去前后空格
		switch choice {
		case "1":
			showChatRoom(client, inputLines)
		case "2":
			showPrivateChat(client, inputLines)
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
