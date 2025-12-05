package server

import (
	"fmt"
	"log"
	"net_chat/internal/database/redis"
	"net_chat/internal/protocol"
	"time"
)

func (s *Server) Handleactivityday(c *ClientConn) {
	activityday, err := redis.GetTop(fmt.Sprintf("activity:day:%04d-%02d-%02d", s.now.Year(), s.now.Month(), s.now.Day()), 20)
	if err != nil {
		log.Fatal("无法获取活跃度排行榜", err)
	}
	c.Outgoing <- &protocol.Message{
		Type:    "activityday",
		Content: activityday,
		From:    "system",
	}
}

func (s *Server) Handleactivityweek(c *ClientConn) {
	year, week := s.now.ISOWeek()
	activityweek, err := redis.GetTop(fmt.Sprintf("activity:week:%02d-%02d", year, week), 20)
	if err != nil {
		log.Fatal("无法获取活跃度排行榜", err)
	}
	c.Outgoing <- &protocol.Message{
		Type:    "activityweek",
		Content: activityweek,
		From:    "system",
	}
}

func (s *Server) Handleactivitytotal(c *ClientConn) {
	activitytotal, err := redis.GetTop("activity:total", 20)
	if err != nil {
		log.Fatal("无法获取活跃度排行榜", err)
	}
	c.Outgoing <- &protocol.Message{
		Type:    "activitytotal",
		Content: activitytotal,
		From:    "system",
	}
}

// OnUserLogin 当用户登录时调用,活跃度+1,更新活跃度排行榜
func OnUserLogin(username string) {
	var login string
	err := redis.TryIncrementWithCooldown(username, login, 1, time.Now())
	if err != nil {
		log.Printf("IncrementActivity login failed for %s: %v", username, err)
	}
}

// OnUserPost 当用户发帖时调用,活跃度+2,更新活跃度排行榜
func OnUserPost(username string) {
	var login string
	err := redis.TryIncrementWithCooldown(username, login, 2, time.Now())
	if err != nil {
		log.Printf("IncrementActivity post failed for %s: %v", username, err)
	}
}
