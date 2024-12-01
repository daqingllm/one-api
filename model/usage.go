package model

import (
	"strconv"
	"time"
)

type Usage struct {
	Id           int    `json:"id"`
	UserId       int    `json:"user_id" gorm:"index:idx_user_hour,priority:1"`
	Hour         int    `json:"hour" gorm:"index:idx_user_hour,priority:2"`
	ModelName    string `json:"model_name"`
	TokenName    string `json:"token_name"`
	Count        int    `json:"count"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Quota        int    `json:"quota" gorm:"default:0"`
}

func getHour() int {
	// yyyyMMddHH
	hourStr := time.Now().Format("2006010215")
	hour, _ := strconv.Atoi(hourStr)
	return hour
}

func AddUsage(userId int, modelName string, tokenName string, count int, inputTokens int, outputTokens int, quota int) error {
	// find by userId, modelName, tokenId, hour
	hour := getHour()
	var usage Usage
	err := DB.Where("user_id = ? AND model_name = ? AND token_name = ? AND hour = ?", userId, modelName, tokenName, hour).First(&usage).Error
	if err != nil {
		// not found, insert
		usage = Usage{
			UserId:       userId,
			Hour:         hour,
			ModelName:    modelName,
			TokenName:    tokenName,
			Count:        count,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Quota:        quota,
		}
		err = DB.Create(&usage).Error
	} else {
		// found, update
		usage.Count += count
		usage.InputTokens += inputTokens
		usage.OutputTokens += outputTokens
		usage.Quota += quota
		err = DB.Save(&usage).Error
	}
	return err
}

func GetUsage(userId int, modelName string, tokenName string, startHour int, endHour int) ([]Usage, error) {
	query := DB.Model(&Usage{}).Where("user_id = ? AND hour >= ? AND hour <= ?", userId, startHour, endHour)
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if tokenName != "" {
		query = query.Where("token_name = ?", tokenName)
	}
	var usages []Usage
	err := query.Find(&usages).Error
	return usages, err
}
