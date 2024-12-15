package job

import (
	"context"
	"time"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// -筛选需要操作的用户 - 查询所有状态为可用且过期时间超过当前时间的所有额度记录的用户
// -根据用户 查询该用户下的所有有效记录
// -计算已过期额度 需要扣除的额度
// -更新用户额度 且 更新额度记录有效状态
func checkAndUpdateExpiredQuota() {
	users, err := model.ExpiredCreditUsers()
	if err != nil {
		logger.Error(context.Background(), "Error getting expired credit users: "+err.Error())
		return
	}
	for _, user := range users {
		records, err := model.GetUserValidRecords(user.Id)
		if err != nil {
			logger.Error(context.Background(), "Error getting user valid records: "+err.Error())
			continue
		}

		var totalExpiredQuota int64
		var totalQuota int64
		var usedQuota int64

		for _, record := range records {
			totalQuota += record.Quota
			if time.Unix(record.ExpiredTime, 0).Before(time.Now()) {
				totalExpiredQuota += record.Quota
				err = model.UpdateRecordExpiredStatus(record.Id)
				if err != nil {
					logger.Error(context.Background(), "Error updating expired status: "+err.Error())
				}
			}
		}
		usedQuota = totalQuota - user.Quota
		if usedQuota < totalExpiredQuota {
			err = model.UpdateUserQuota(user.Id, usedQuota-totalExpiredQuota)
			if err != nil {
				logger.Error(context.Background(), "Error updating user quota: "+err.Error())
			}
		}
	}
}

func scheduleNextRun(location *time.Location) {
	// 获取当前时间
	now := time.Now().In(location)
	// 计算下一个任务执行时间
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).Add(24 * time.Hour)
	// 计算当前时间到下一个执行时间的等待时间
	duration := next.Sub(now)

	// Schedule function to be called at the next run time
	time.AfterFunc(duration, func() {
		checkAndUpdateExpiredQuota()
		scheduleNextRun(location)
	})
}

// 定时任务，北京时间每天0点执行
func QuotaJob() {
	ctx := context.Background()
	logger.Info(ctx, "QuotaJob init")
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return
	}
	scheduleNextRun(location)
	select {}
}
