package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/message"
)

// Status 0 初始化 1 成功 2 失败

type RemindRecord struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index"`
	Email       string `json:"email" gorm:"type:varchar(255)"`
	Status      int    `json:"status" gorm:"type:int;default:0"`
	CreatedTime int64  `json:"created_time" gorm:"autoCreateTime"`
}

func addRemindRecord(userId int, email string) (int, error) {
	record := &RemindRecord{
		UserId:      userId,
		Email:       email,
		CreatedTime: time.Now().Unix(),
		Status:      0,
	}
	res := DB.Create(&record)
	return record.Id, res.Error
}

// AddRemindRecord 添加提醒记录
func AddRemindRecord(userId int, email string) (int, error) {
	id, err := addRemindRecord(userId, email)
	if err != nil {
		return 0, err
	}
	// 更新缓存
	err = SetUserRemindPool(userId, email, 24*3600)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// 查询用户是否在一天内已提醒
func IsUserNotified(userId int) (RemindRecord, error) {
	var record RemindRecord
	err := DB.Where("user_id = ? AND created_time >= ?", userId, time.Now().Add(-24*time.Hour).Unix()).First(&record).Error
	if err != nil {
		return record, err
	}
	return record, nil
}

func GetRemindRecord(userId int) string {
	// check cache
	email, err := GetUserRemindPool(userId)
	if (err != nil) || (email == "") {
		record, err := IsUserNotified(userId)
		if err != nil {
			logger.SysError("failed to check remind record" + err.Error())
			return ""
		}
		if record.Status == 1 {
			// 更新缓存时间
			time := int((24 * time.Hour).Seconds()) - int(time.Now().Unix()-record.CreatedTime)
			err = SetUserRemindPool(userId, record.Email, time)
			if err != nil {
				logger.SysError("failed to update remind record" + err.Error())
			}
			return record.Email
		} else {
			return ""
		}
	}
	return email
}

// 更新提醒记录状态
func UpdateRemindRecordStatus(id int, status int) error {
	return DB.Model(&RemindRecord{}).Where("id = ?", id).Update("status", status).Error
}

func NotifyByEmail(userInfo *User) {
	// check if already notified
	email := GetRemindRecord(userInfo.Id)
	if email != "" {
		// user already notified
		return
	}
	go func() {
		insertId, err := AddRemindRecord(userInfo.Id, userInfo.Email)
		if err != nil {
			logger.SysError("failed to add remind record" + err.Error())
			return
		}
		topUpLink := fmt.Sprintf("%s/topup", config.ServerAddress)
		prompt := "AiHubMix余额提醒，剩余" + common.ShowQuota(userInfo.Quota)
		currentTime := helper.GetFormattedTimeString()
		err = message.SendEmail(prompt, userInfo.Email,
			fmt.Sprintf("尊敬的 %s： <br><br>"+"截至 %s，<br>当前账户余额为 %s，请您尽快充值，以免影响使用！<br><a href='%s'>点此充值>></a>", userInfo.Username, currentTime, common.ShowQuota(userInfo.Quota), topUpLink))
		if err != nil {
			logger.SysError("failed to send email" + err.Error())
			UpdateRemindRecordStatus(insertId, 2)
		}
		UpdateRemindRecordStatus(insertId, 1)
	}()
}

func QuotaUseTest(userId int, quota int64) (err error) {
	userInfo, err := GetUserInfo(userId)
	if err != nil {
		return err
	}
	if userInfo.Quota < quota {
		return errors.New("用户额度不足")
	}
	if userInfo.Notify {
		quotaTooLow := userInfo.Quota <= userInfo.QuotaRemindThreshold
		if quotaTooLow && userInfo.Email != "" {
			NotifyByEmail(userInfo)
		}
	}
	err = DecreaseUserQuota(userId, quota)
	return err
}
