package job

import (
	"context"
	"strconv"
	"time"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// 定时任务，北京时间每天0点1分执行
func QuotaJob() {
	ctx := context.Background()
	logger.Info(ctx, "QuotaJob init")
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return
	}
	// 获取当前时间
	now := time.Now().In(location)
	// 计算下一个任务执行时间 0点1分
	// next := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, location).Add(24 * time.Hour)
	next := now.Add(5 * time.Minute)
	// 计算当前时间到下一个执行时间的等待时间
	duration := next.Sub(now)

	// Schedule function to be called at the next run time
	time.AfterFunc(duration, func() {
		err := checkAndUpdateExpiredQuota()
		if err != nil {
			logger.Error(ctx, "checkAndUpdateExpiredQuota error: "+err.Error())
		}
		QuotaJob()
	})
}

// -筛选需要操作的用户 - 查询所有状态为可用且过期时间超过当前时间的所有额度记录的用户
// -根据用户 查询该用户下的所有有效记录
// -遍历记录，如果记录过期时间小于当前时间，更新记录状态为过期
// -计算有效期内的记录的额度总和，如果总和小于用户的额度，更新用户的额度
// -更新job状态为完成
func checkAndUpdateExpiredQuota() error {
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return err
	}
	nowStr := time.Now().In(location).Format("2006-01-02")
	// 写入job表，判断是否可以执行任务
	aff, err := model.InsertScheduleRecordIgnoreDuplicateKey("QuotaExpire", nowStr)
	if err != nil {
		logger.Error(ctx, "InsertScheduleRecordIgnoreDuplicateKey error: "+err.Error())
		return err
	}
	if aff == 0 {
		logger.Info(ctx, nowStr+" QuotaExpire already executed")
		return nil
	}

	users, err := model.ExpiredCreditUsers()
	if err != nil {
		logger.Error(ctx, "Error getting expired credit users: "+err.Error())
		// 更新job状态为失败
		err = model.UpdateScheduleRecordStatus("QuotaExpire", nowStr, model.SCHEDULE_STATUS_FAILED)
		if err != nil {
			logger.Error(ctx, "UpdateScheduleRecordStatus failed error: "+err.Error())
		}
		return err
	}
	successUsers := 0
	failUsers := 0

	for _, user := range users {
		records, err := model.GetUserValidRecords(user.Id)
		if err != nil {
			logger.Error(ctx, "Error getting user valid records: userId"+strconv.Itoa(user.Id)+err.Error())
			continue
		}

		var validQuota int64

		for _, record := range records {
			if time.Unix(record.ExpiredTime, 0).Before(time.Now()) {
				err = model.UpdateRecordExpiredStatus(record.Id)
				if err != nil {
					logger.Error(ctx, "Error updating expired status: "+err.Error())
					// 如果过期状态更新失败，额度仍为有效 先不扣减
					validQuota += record.Quota
				}
			} else {
				validQuota += record.Quota
			}
		}
		if validQuota < user.Quota {
			err = model.UpdateUserQuota(user.Id, validQuota)
			if err != nil {
				logger.Error(ctx, "Error updating user quota: "+err.Error())
				failUsers += 1
			} else {
				successUsers += 1
			}
		}
	}
	// 更新job状态为完成
	err = model.UpdateScheduleRecordStatus("QuotaExpire", nowStr, model.SCHEDULE_STATUS_FINISHED)
	if err != nil {
		logger.Error(ctx, "UpdateScheduleRecordStatus finished error: "+err.Error())
	}
	// 写入job表，记录成功和失败的用户数 总数:成功数:失败数:未处理数
	err = model.UpdateScheduleRecordExt("QuotaExpire", nowStr, strconv.Itoa(len(users))+":"+strconv.Itoa(successUsers)+":"+strconv.Itoa(failUsers)+":"+strconv.Itoa(len(users)-successUsers-failUsers))
	if err != nil {
		logger.Error(ctx, "UpdateScheduleRecordExt error: "+err.Error())
	}
	logger.Info(ctx, "checkAndUpdateExpiredQuota end")
	return nil
}
