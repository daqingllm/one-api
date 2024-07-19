package model

import "time"

type ModelUsage struct {
	Id        int       `json:"id"`
	Date      time.Time `json:"date" gorm:"type:date;index:index_date_modelname,priority:1"`
	ModelName string    `json:"model_name" gorm:"type:varchar(128);index:index_date_modelname,priority:2"`
	CallCount int       `json:"call_count" gorm:"type:int;default:0"`
	TokenUsed int       `json:"token_used" gorm:"type:int;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"datetime"`
}
