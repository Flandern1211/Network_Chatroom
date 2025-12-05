package database

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // 添加这行导入
	"log"
	"os"
)

var DB *sql.DB

// InitMySQL 服务端连接MySQL初始化
func InitMySQL() error {

	//从环境变量读取需要的值
	dbUser := os.Getenv("MYSQL_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	if dbPassword == "" {
		dbPassword = "20041211wzwaicjW."
	}
	dbHost := os.Getenv("MYSQL_HOST") // 读取环境变量
	if dbHost == "" {
		dbHost = "127.0.0.1" // 默认值
	}
	dbPort := os.Getenv("MYSQL_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbName := os.Getenv("MYSQL_DATABASE")
	if dbName == "" {
		dbName = "net_chat"
	}

	//数据库连接字符串构建或数据源URL拼接（Database Source Handle: 数据库源句柄）
	dsh := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True", dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	//不要使用:=导致DB成为局部变量
	DB, err = sql.Open("mysql", dsh)
	if err != nil {
		return fmt.Errorf("打开数据库失败:%w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败:%w", err)
	}

	return nil
}

func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			log.Fatal("无法关闭mysql连接")
		}
		log.Println("MySQL数据库连接已关闭")
	}
}
