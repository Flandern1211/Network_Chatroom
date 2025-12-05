package database

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net_chat/internal/database/redis"
	"time"
)

type User struct {
	ID       int
	Username string
	Password string
}

func GetUserFromRedis(username string) (*User, error) {
	userKey := fmt.Sprintf("user:%s", username)
	userData, err := redis.Rdb.Get(redis.Rctx, userKey).Result()

	if err != nil && err.Error() == "redis: nil" {
		// Redis 中不存在该用户，从数据库获取
		return GetUserFromDB(username)
	} else if err != nil {
		// 其他 Redis 错误
		return nil, fmt.Errorf("在从redis缓存中查询数据时发生错误:%w", err)
	}

	var user User
	if err := json.Unmarshal([]byte(userData), &user); err != nil {
		return nil, fmt.Errorf("在对redis缓存的数据反序列化时发生错误：%w", err)
	}
	return &user, nil
}

// 从数据库中获取数据
func GetUserFromDB(username string) (*User, error) {
	var user User
	user.Username = username // 设置用户名
	query := "SELECT id, username, password_hash FROM users WHERE username = ?"
	err := DB.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, fmt.Errorf("用户'%s'不存在或查询失败:%w", username, err)
	}

	// 缓存到redis
	userKey := fmt.Sprintf("user:%s", username)
	userData, _ := json.Marshal(user)
	redis.Rdb.Set(redis.Rctx, userKey, userData, time.Hour*24)

	return &user, nil
}
func RegisterUser(username, password string) error {
	//1.密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("哈希密码失败,%w", err)
	}

	//2.插入数据库
	query := "INSERT INTO users(username,password_hash) VALUES(?,?)"
	result, err := DB.Exec(query, username, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("注册失败，可能是用户名已经存在了,%w", err)
	}

	// 获取插入的用户ID并缓存用户信息
	userID, _ := result.LastInsertId()
	user := User{
		ID:       int(userID),
		Username: username,
		Password: string(hashedPassword),
	}

	// 缓存到 Redis
	userKey := fmt.Sprintf("user:%s", username)
	//转为json字符序列存入redis
	userData, _ := json.Marshal(user)
	redis.Rdb.Set(redis.Rctx, userKey, userData, time.Hour*24)

	return nil
}

// AuthenticateUser 验证用户登录
func AuthenticateUser(username, password string) error {
	user, err := GetUserFromRedis(username)
	if err != nil {
		return fmt.Errorf("用户'%s'不存在或查询失败:%w", username, err)
	}

	// 比较密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return fmt.Errorf("密码不匹配,%w", err)
	}

	// 缓存到 Redis
	userKey := fmt.Sprintf("user:%s", username)
	userData, _ := json.Marshal(user)
	redis.Rdb.Set(redis.Rctx, userKey, userData, time.Hour*24)
	return nil
}
