package job

import (
	"context"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"time"
)

func ExpireCache() {
	//定时任务，北京时间每天04:10执行
	ctx := context.Background()
	logger.Info(ctx, "ExpireCache init")
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
		model.DeleteExpiredCache()
		// 完成后继续调度下一次执行
		ExpireCache()
	})
}
