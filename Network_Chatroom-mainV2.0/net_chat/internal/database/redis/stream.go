package redis

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"sort"
	"time"
)

// AddRoomMessage 保存聊天室消息到消息队列中
func AddRoomMessage(room string, sender string, content string) (string, error) {
	if room == "" || sender == "" {
		return "", fmt.Errorf("无发送方")
	}
	streamKey := fmt.Sprintf("stream:room:%s", room)
	vals := map[string]interface{}{
		"sender":  sender,
		"content": content,
		"ts":      time.Now().Unix(),
	}

	id, err := Rdb.XAdd(Rctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: vals,
	}).Result()
	if err != nil {
		return "", err
	}
	return id, nil
}

// AddPrivateMessage 保存私聊消息到消息队列中
func AddPrivateMessage(sender, recipient string, content string, recipientOnlie bool) (string, error) {
	//对发送和接收者排序，保证私聊两个对象相同的话保存至一个stream中，节省空间
	users := []string{sender, recipient}
	sort.Strings(users)
	streamKey := fmt.Sprintf("stream:pm:%s:%s", users[0], users[1])
	vals := map[string]interface{}{
		"sender":  sender,
		"content": content,
		"ts":      time.Now().Unix(),
	}

	id, err := Rdb.XAdd(Rctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: vals,
	}).Result()
	if err != nil {
		return "", err
	}

	//如果接收者不在线，记录未读数量
	if !recipientOnlie {
		unreadKey := fmt.Sprintf("unread:%s", recipient)
		err := Rdb.HIncrBy(Rctx, unreadKey, sender, 1).Err()
		if err != nil {
			return id, fmt.Errorf("记录未读数量失败%w", err)
		}
	}
	return id, nil
}

// GetRoomLastNMessage 获取房间中最近n条的消息并按时间顺序输出
func GetRoomLastNMessage(room string, n int64) ([]redis.XMessage, error) {
	key := fmt.Sprintf("stream:room:%s", room)
	//反向遍历获取最近的n条消息
	msgs, err := Rdb.XRevRangeN(Rctx, key, "+", "-", n).Result()
	if err != nil {
		//没有数据
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	if len(msgs) == 0 {
		return nil, nil
	}
	//反转成时间顺序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// GetPrivateLastNMessage 获取房间中最近n条的消息并按时间顺序输出
func GetPrivateLastNMessage(userA, userB string, n int64) ([]redis.XMessage, error) {
	users := []string{userA, userB}
	sort.Strings(users)
	key := fmt.Sprintf("stream:pm:%s:%s", users[0], users[1])
	msgs, err := Rdb.XRevRangeN(Rctx, key, "+", "-", n).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	if len(msgs) == 0 {
		return nil, nil
	}
	//反转成时间顺序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// GetUnreadForUser 获取用户未读消息提示数
func GetUnreadForUser(user string) (map[string]string, error) {
	if user == "" {
		return nil, fmt.Errorf("empty user")
	}

	key := fmt.Sprintf("unread:%s", user)
	res, err := Rdb.HGetAll(Rctx, key).Result()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ClearUnreadForUser 在用户查看了私聊后，清除unread技数
func ClearUnreadForUser(user, sender string) error {
	if user == "" {
		return fmt.Errorf("empty user")
	}

	key := fmt.Sprintf("unread:%s", user)
	//检查是否存在与该sender的未读消息
	exist, err := Rdb.HExists(Rctx, key, sender).Result()
	if err != nil {
		return fmt.Errorf("在检查是否存在与该私聊用户的未读消息提醒时报错:%w", err)
	}

	if !exist {
		return nil
	} else {
		return Rdb.HDel(Rctx, key, sender).Err()
	}

}
