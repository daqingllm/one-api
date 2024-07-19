package model

import (
	"gorm.io/gorm/clause"
	"time"
)

const (
	SCHEDULE_STATUS_RUNNING  = 0
	SCHEDULE_STATUS_FINISHED = 1
	SCHEDULE_STATUS_FAILED   = 2
)

// ScheduleRecord is a struct that represents a record of a schedule.
type ScheduleRecord struct {
	Id        int       `json:"id"`
	Job       string    `json:"job" gorm:"type:varchar(128);unique:idx_job,priority:1"`
	Key       string    `json:"key" gorm:"type:varchar(128);unique:idx_job,priority:2"`
	Status    int       `json:"status" gorm:"type:int;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"datetime"`
}

// insert into ScheduleRecord ignore duplicate key and return affected rows
func InsertScheduleRecordIgnoreDuplicateKey(job string, key string) (int64, error) {
	scheduleRecord := &ScheduleRecord{
		Job:       job,
		Key:       key,
		CreatedAt: time.Now(),
	}
	result := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(scheduleRecord)
	return result.RowsAffected, result.Error
}

// update ScheduleRecord status by job and key
func UpdateScheduleRecordStatus(job string, key string, status int) error {
	return DB.Model(&ScheduleRecord{}).Where("job = ? and key = ?", job, key).Update("status", status).Error
}
