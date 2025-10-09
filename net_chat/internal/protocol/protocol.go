package protocol

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
)

//协议文件

// 2.定义消息结构体（Message）
type Message struct {
	Type    string `json:"type"`    //消息类型：login（登录），message（聊天），list（查询用户列表），exit（退出）
	Content string `json:"content"` //消息内容
	From    string `json:"from"`    //谁发的消息
	To      string `json:"to"`      //发给谁（私聊时使用，其他时候为空）
}

// 3.定义统一的发送消息的函数
func SendMsg(w io.Writer, msg *Message) error {
	jMsg, err := json.Marshal(msg) //转为json格式
	if err != nil {
		return err
	}

	//定义4个字节用来标识消息长度
	//3.1首先获得该条消息的长度
	MsgLength := len(jMsg)
	//3.2创建一个字节切片来存储这个长度值
	//为什么要用4个字节来传递？
	//为什么要转为字节传递,不直接长度的值？
	LengthBuf := make([]byte, 4)
	//什么用？
	binary.BigEndian.PutUint32(LengthBuf, uint32(MsgLength))
	//uint32（MesgLength）将MsgLength转为uint32类型的值
	//binary.BigEndian.PutUint32表示按照大端字节序的规则,转为4个字节,并放在在LengthBuf（PutUint32只能转uint32的）
	//先发送长度，再发送消息内容
	if _, err := w.Write(LengthBuf); err != nil {
		return err
	}

	//再发送消息本身
	if _, err := w.Write(jMsg); err != nil {
		return err
	}
	return nil
}

// 4. 接收函数
// 连接中读出一条完整消息，解包成消息结构体
func ReadMsg(r *bufio.Reader) (*Message, error) {
	LengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, LengthBuf); err != nil {
		return nil, err
	}

	//2.解析消息长度
	MsgLenght := binary.BigEndian.Uint32(LengthBuf)

	//3.根据长度来读取消息内容
	MsgBuf := make([]byte, MsgLenght)
	if _, err := io.ReadFull(r, MsgBuf); err != nil {
		return nil, err
	}

	//4.将json格式的消息反序列化
	var msg Message
	if err := json.Unmarshal(MsgBuf, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

//
