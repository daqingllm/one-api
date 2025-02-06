package model

import (
	"context"
	"time"

	"github.com/songquanpeng/one-api/common/logger"
)

type ModelUsage struct {
	Id        int       `json:"id"`
	Date      time.Time `json:"date" gorm:"type:date;index:index_date_modelname,priority:1"`
	ModelName string    `json:"model_name" gorm:"type:varchar(128);index:index_date_modelname,priority:2"`
	CallCount int       `json:"call_count" gorm:"type:int;default:0"`
	TokenUsed int       `json:"token_used" gorm:"type:int;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"datetime"`
}

type ModelUsageCount struct {
	Model     string `json:"model"`
	CallCount int64  `json:"call_count"`
	TokenUsed int64  `json:"token_used"`
}

func RefreshModelUsage(ctx context.Context, lastdays int) error {
	location, err := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return err
	}
	now := time.Now().In(location)
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	yesterdayStr := yesterday.Format("2006-01-02")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayTimestamp := yesterday.Unix()
	todayTimestamp := today.Unix()
	// 循环lastdays次数
	for i := 0; i < lastdays; i++ {
		// 查询log表，统计昨天的模型使用情况，插入模型使用统计表
		query := "INSERT INTO model_usages (date, model_name, call_count, token_used, created_at) SELECT ?, model_name, count(1), sum(quota), now() FROM logs where created_at >? and created_at <? group by model_name"
		result := DB.Exec(query, yesterdayStr, yesterdayTimestamp, todayTimestamp)
		if result.Error != nil {
			logger.Error(ctx, "CalcModelUsage insert error: "+result.Error.Error())
			return result.Error
		}
		todayTimestamp = yesterdayTimestamp
		yesterday = yesterday.Add(-time.Hour * 24)
		yesterdayTimestamp = yesterday.Unix()
		yesterdayStr = yesterday.Format("2006-01-02")
	}
	return nil
}

func GetModelUsageDetail(ctx context.Context, recentDay int, endDate string) ([]ModelUsage, error) {
	var modelUsages []ModelUsage
	location, err := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	if err != nil {
		logger.Error(ctx, "Error loading location: "+err.Error())
		return modelUsages, err
	}
	// Initialize endTime as current time
	var endTime time.Time
	if endDate != "" {
		// If endDate is provided, parse it
		const layout = "2006-01-02"
		endTime, err = time.ParseInLocation(layout, endDate, location)
		if err != nil {
			logger.Error(ctx, "Error parsing endDate: "+err.Error())
			return modelUsages, err
		}
	} else {
		endTime = time.Now().In(location)
	}

	startTime := time.Date(endTime.Year(), endTime.Month(), endTime.Day()-recentDay, 0, 0, 0, 0, endTime.Location())
	err = DB.Model(&ModelUsage{}).Where("date >= ? AND date <= ?", startTime, endTime).Find(&modelUsages).Error
	return modelUsages, err
}

func GetModelUsageCount(ctx context.Context, date time.Time) ([]ModelUsageCount, error) {
	query := "SELECT model_name as model, sum(call_count) as call_count, sum(token_used) as token_used FROM model_usages WHERE date >= ? GROUP BY model_name"
	var modelUsageCount []ModelUsageCount
	err := DB.Raw(query, date).Scan(&modelUsageCount).Error
	return modelUsageCount, err
}
