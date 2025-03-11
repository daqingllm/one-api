package job

import (
	"context"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"time"
)

func ExpireHistoryLogs() {
	//定时任务，北京时间每天04:10执行
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return
	}
	now := time.Now().In(location)
	// 计算下一个任务执行时间
	next := now.Add(time.Hour * 24)
	next = time.Date(next.Year(), next.Month(), next.Day(), 4, 10, 0, 0, next.Location())
	// 计算当前时间到下一个执行时间的等待时间
	waitDuration := next.Sub(now)
	// 设置定时器
	time.AfterFunc(waitDuration, func() {
		// 计算6个月前的timestamp
		sixMonthAgo := time.Now().AddDate(0, -6, 0).Unix()
		rows, err := model.DeleteOldLog(sixMonthAgo)
		if err != nil {
			logger.Error(ctx, "Error deleting old log: "+err.Error())
		} else {
			logger.Info(ctx, "Deleted old log rows: "+string(rows))
		}
		// 计算7天前的timestamp
		sevenDayAgo := time.Now().AddDate(0, 0, -7).Unix()
		rows, err = model.DeleteExpiredFailedLog(sevenDayAgo)
		if err != nil {
			logger.Error(ctx, "Error deleting expired failed log: "+err.Error())
		} else {
			logger.Info(ctx, "Deleted expired failed log rows: "+string(rows))
		}

		// 完成后继续调度下一次执行
		ExpireHistoryLogs()
	})
}
