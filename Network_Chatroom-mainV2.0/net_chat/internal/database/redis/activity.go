package redis

///////////活跃度排名////////////////////

import (
	"fmt"
	"time"
)

// IncrementActivity 活跃度排行榜更新
func IncrementActivity(user string, add float64, now time.Time) error {
	//日榜
	dayKey := fmt.Sprintf("activity:day:%04d-%02d-%02d", now.Year(), now.Month(), now.Day())
	//周榜
	year, week := now.ISOWeek()
	weekKey := fmt.Sprintf("activity:week:%02d-%02d", year, week)
	//总榜
	totalKey := "activity:total"

	//原子 pipeline 提交到三个 key，保证按顺序执行
	pipe := Rdb.Pipeline()
	pipe.ZIncrBy(Rctx, dayKey, add, user)
	pipe.ZIncrBy(Rctx, weekKey, add, user)
	pipe.ZIncrBy(Rctx, totalKey, add, user)

	//全部交给redis执行
	cmds, err := pipe.Exec(Rctx)
	if err != nil {
		return err
	}
	// 检查每个命令的执行结果
	for _, cmd := range cmds {
		if cmd.Err() != nil {
			return cmd.Err()
		}
	}
	return nil
}

// TryIncrementWithCooldown 防刷，设置一个短期key
func TryIncrementWithCooldown(user, action string, add float64, now time.Time) error {
	cooldownKey := fmt.Sprintf("action:cool:%s:%s", action, user)
	//使用SET NX保证短时间只计一次
	ok, err := Rdb.SetNX(Rctx, cooldownKey, "1", 30*time.Second).Result()
	if err != nil {
		return fmt.Errorf("短期key设置失败%w", err)
	}
	//!ok 短期key存在,分数不增加
	if !ok {
		return nil
	}
	//短期key不存在，分数增加
	return IncrementActivity(user, add, now)

}

// GetTop 用户获取排行榜
func GetTop(key string, topN int64) (string, error) {
	res, err := Rdb.ZRevRangeWithScores(Rctx, key, 0, topN-1).Result()
	if err != nil {
		return "", fmt.Errorf("无法获取数据:%w", err)
	}
	if len(res) == 0 {
		return "暂无数据", nil
	}

	var result string
	for i, item := range res {
		result += fmt.Sprintf("%d. %v (活跃度: %v)\n", i+1, item.Member, item.Score)
	}
	return result, nil

}
