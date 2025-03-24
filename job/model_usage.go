package job

import (
	"context"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"time"
)

// 定时任务，北京时间每天4点执行
func CalcModelUsageSchedule() {
	ctx := context.Background()
	logger.Info(ctx, "CalcModelUsage init")
	location, err := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return
	}
	now := time.Now().In(location)
	// 计算下一个任务执行时间
	next := now.Add(time.Hour * 24)
	if now.Hour() < 4 {
		next = now
	}
	next = time.Date(next.Year(), next.Month(), next.Day(), 4, 0, 0, 0, next.Location())

	// 计算当前时间到下一个执行时间的等待时间
	waitDuration := next.Sub(now)

	// 设置定时器
	time.AfterFunc(waitDuration, func() {
		err := dailyTask(ctx)
		if err != nil {
			logger.Error(context.Background(), "dailyTask error: "+err.Error())
		}
		// 完成后继续调度下一次执行
		CalcModelUsageSchedule()
	})
}

func dailyTask(ctx context.Context) error {
	logger.Info(ctx, "CalcModelUsage start")
	// 获取北京时间昨天的日期，格式为 yyyy-MM-dd
	location, err := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return err
	}
	now := time.Now().In(location)
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	yesterdayStr := yesterday.Format("2006-01-02")

	// 写入job表，判断是否可以执行任务
	aff, err := model.InsertScheduleRecordIgnoreDuplicateKey("CalcModelUsage", yesterdayStr)
	if err != nil {
		logger.Error(ctx, "InsertScheduleRecordIgnoreDuplicateKey error: "+err.Error())
		return err
	}
	if aff == 0 {
		logger.Info(ctx, "CalcModelUsage already executed")
		return nil
	}

	// 获取北京时间昨天0点和今天0点的时间戳
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayTimestamp := yesterday.Unix()
	todayTimestamp := today.Unix()

	// 查询log表，统计昨天的模型使用情况，插入模型使用统计表
	query := "INSERT INTO model_usages (date, model_name, call_count, token_used, created_at) SELECT ?, model_name, count(1), sum(quota), now() FROM logs where created_at >? and created_at <? group by model_name"
	result := model.DB.Exec(query, yesterdayStr, yesterdayTimestamp, todayTimestamp)
	if result.Error != nil {
		_ = model.UpdateScheduleRecordStatus("CalcModelUsage", yesterdayStr, model.SCHEDULE_STATUS_FAILED)
		logger.Error(ctx, "CalcModelUsage insert error: "+result.Error.Error())
		return result.Error
	}

	// 更新job表状态
	_ = model.UpdateScheduleRecordStatus("CalcModelUsage", yesterdayStr, model.SCHEDULE_STATUS_FINISHED)
	logger.Info(context.Background(), "dailyTask end")
	return nil
}
