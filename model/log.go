package model

import (
	"context"
	"fmt"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
	"time"
)

type Log struct {
	Id               int    `json:"id"`
	UserId           int    `json:"user_id" gorm:"index"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index:idx_created_at_type"`
	Type             int    `json:"type" gorm:"index:idx_created_at_type"`
	Content          string `json:"content"`
	Username         string `json:"username" gorm:"index:index_username_model_name,priority:2;default:''"`
	TokenName        string `json:"token_name" gorm:"index;default:''"`
	ModelName        string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota            int    `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CachedTokens     int    `json:"cached_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	ChannelId        int    `json:"channel" gorm:"index"`
	Duration         int64  `json:"duration" gorm:"default:0"`
}

const (
	LogTypeUnknown = iota
	LogTypeTopup
	LogTypeConsume
	LogTypeManage
	LogTypeSystem
)

type FailedLog struct {
	Id            int    `json:"id"`
	UserId        int    `json:"user_id" gorm:"index:idx_user_id_created_at"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;index:idx_user_id_created_at"`
	ModelName     string `json:"model" gorm:"type:varchar(128)"`
	Url           string `json:"url" gorm:"type:varchar(255)"`
	RequestId     string `json:"request_id" gorm:"type:varchar(128)"`
	ChannelsTried string `json:"channels_tried"`
	StatusCode    int    `json:"status_code"`
	ErrorResponse string `json:"error_response"`
	RequestBody   string `json:"request_body"`
	Duration      int64  `json:"duration"`
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !config.LogConsumeEnabled {
		return
	}
	log := &Log{
		UserId:    userId,
		Username:  GetUsernameById(userId),
		CreatedAt: helper.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.SysError("failed to record log: " + err.Error())
	}
}

func RecordFailedLog(ctx context.Context, userId int, modelName string, channelsTried string, statusCode int, errorResponse string, requestBody string, requestId string, url string) {
	// requestBody may be too long, so only log the first 1000 characters
	shortReq := requestBody
	if len(requestBody) > 1000 {
		shortReq = requestBody[:1000] + "..."
	}
	logger.Error(ctx, fmt.Sprintf("record failed log: userId=%d, modelName=%s, channelsTried=%s, statusCode=%d, errorResponse=%s, requestBody=%s", userId, modelName, channelsTried, statusCode, errorResponse, shortReq))
	failedLog := &FailedLog{
		UserId:        userId,
		CreatedAt:     helper.GetTimestamp(),
		ModelName:     modelName,
		Url:           url,
		RequestId:     requestId,
		ChannelsTried: channelsTried,
		StatusCode:    statusCode,
		ErrorResponse: errorResponse,
		RequestBody:   requestBody,
	}
	st := ctx.Value(helper.StartTimeKey)
	if st != nil {
		failedLog.Duration = time.Now().UnixMilli() - st.(int64)
	}
	err := LOG_DB.Create(failedLog).Error
	if err != nil {
		logger.SysError("failed to record failed log: " + err.Error())
	}
}

func RecordTopupLog(userId int, content string, quota int) {
	log := &Log{
		UserId:    userId,
		Username:  GetUsernameById(userId),
		CreatedAt: helper.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Quota:     quota,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.SysError("failed to record log: " + err.Error())
	}
}

func RecordConsumeLog(ctx context.Context, userId int, channelId int, promptTokens int, cachedTokens int, completionTokens int, modelName string, tokenName string, quota int64, content string) {
	logger.Info(ctx, fmt.Sprintf("record consume log: userId=%d, channelId=%d, promptTokens=%d, completionTokens=%d, modelName=%s, tokenName=%s, quota=%d, content=%s", userId, channelId, promptTokens, completionTokens, modelName, tokenName, quota, content))
	if !config.LogConsumeEnabled {
		return
	}
	log := &Log{
		UserId:           userId,
		Username:         GetUsernameById(userId),
		CreatedAt:        helper.GetTimestamp(),
		Type:             LogTypeConsume,
		Content:          content,
		PromptTokens:     promptTokens,
		CachedTokens:     cachedTokens,
		CompletionTokens: completionTokens,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            int(quota),
		ChannelId:        channelId,
	}
	st := ctx.Value(helper.StartTimeKey)
	if st != nil {
		log.Duration = time.Now().UnixMilli() - st.(int64)
	}
	addNewLog(log)
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int) (logs []*Log, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("type = ?", logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	return logs, err
}

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int) (logs []*Log, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("user_id = ? and type = ?", userId, logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Omit("id").Find(&logs).Error
	return logs, err
}

func SearchAllLogs(keyword string) (logs []*Log, err error) {
	err = LOG_DB.Where("type = ? or content LIKE ?", keyword, keyword+"%").Order("id desc").Limit(config.MaxRecentItems).Find(&logs).Error
	return logs, err
}

func SearchUserLogs(userId int, keyword string) (logs []*Log, err error) {
	err = LOG_DB.Where("user_id = ? and type = ?", userId, keyword).Order("id desc").Limit(config.MaxRecentItems).Omit("id").Find(&logs).Error
	return logs, err
}

// @deprecated
func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int) (quota int64) {
	ifnull := "ifnull"
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(quota),0)", ifnull))
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&quota)
	return quota
}

//func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
//	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
//	if username != "" {
//		tx = tx.Where("username = ?", username)
//	}
//	if tokenName != "" {
//		tx = tx.Where("token_name = ?", tokenName)
//	}
//	if startTimestamp != 0 {
//		tx = tx.Where("created_at >= ?", startTimestamp)
//	}
//	if endTimestamp != 0 {
//		tx = tx.Where("created_at <= ?", endTimestamp)
//	}
//	if modelName != "" {
//		tx = tx.Where("model_name = ?", modelName)
//	}
//	tx.Where("type = ?", LogTypeConsume).Scan(&token)
//	return token
//}

func DeleteOldLog(targetTimestamp int64) (int64, error) {
	result := LOG_DB.Where("created_at < ?", targetTimestamp).Delete(&Log{})
	return result.RowsAffected, result.Error
}

func DeleteExpiredFailedLog(targetTimestamp int64) (int64, error) {
	result := LOG_DB.Where("created_at < ?", targetTimestamp).Delete(&FailedLog{})
	return result.RowsAffected, result.Error
}

type LogStatistic struct {
	Day              string `gorm:"column:day"`
	ModelName        string `gorm:"column:model_name"`
	RequestCount     int    `gorm:"column:request_count"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
}

//func SearchLogsByDayAndModel(userId, start, end int) (LogStatistics []*LogStatistic, err error) {
//	groupSelect := "DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-%d') as day"
//
//	if common.UsingPostgreSQL {
//		groupSelect = "TO_CHAR(date_trunc('day', to_timestamp(created_at)), 'YYYY-MM-DD') as day"
//	}
//
//	err = LOG_DB.Raw(`
//		SELECT `+groupSelect+`,
//		model_name, count(1) as request_count,
//		sum(quota) as quota,
//		sum(prompt_tokens) as prompt_tokens,
//		sum(completion_tokens) as completion_tokens
//		FROM logs
//		WHERE type=2
//		AND user_id= ?
//		AND created_at BETWEEN ? AND ?
//		GROUP BY day, model_name
//		ORDER BY day, model_name
//	`, userId, start, end).Scan(&LogStatistics).Error
//
//	return LogStatistics, err
//}

func PaginateLogs(startId int, num int) (logs []*Log, err error) {
	err = LOG_DB.Where("id < ?", startId).Where("type = ?", LogTypeConsume).Order("id desc").Limit(num).Find(&logs).Error
	return logs, err
}
