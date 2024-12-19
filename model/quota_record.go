package model

import (
	"time"
)

// OuotaRecord 额度记录
// GrantType 充值类型 0 初始化 1 支付宝 2 stripe 3 兑换 Status 1 有效 0 无效

type QuotaRecord struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index"`
	GrantType   int    `json:"grant_type" gorm:"int"`
	GrantId     string `json:"grant_id" gorm:"unique"`
	ExpiredTime int64  `json:"expired_time" gorm:"bigint"`
	CreatedTime int64  `json:"created_time" gorm:"autoCreateTime"`
	Status      int    `json:"status" gorm:"type:int;default:1"`
	Quota       int64  `json:"quota" gorm:"default:0"`
}

// AddQuotaRecord 添加额度记录
func AddQuotaRecord(userId int, grantType int, grantId string, quota int64) error {
	record := &QuotaRecord{
		UserId:      userId,
		GrantType:   grantType,
		GrantId:     grantId,
		CreatedTime: time.Now().Unix(),
		ExpiredTime: time.Now().AddDate(0, 6, 0).Unix(),
		Quota:       quota,
		Status:      1,
	}
	return DB.Create(&record).Error
}

// GetQuotaRecordsByUserId 根据用户id分页查询所有状态的取额度记录
func GetQuotaRecordsByUserId(userId int, startIdx int, num int) (records []*QuotaRecord, err error) {
	err = DB.Where("user_id = ?", userId).Order("id desc").Limit(num).Offset(startIdx).Find(&records).Error
	return records, err
}

// GetUserValidRecords 根据用户id查询有效额度记录
func GetUserValidRecords(userId int) (records []*QuotaRecord, err error) {
	err = DB.Where("user_id = ? and status = ?", userId, 1).Find(&records).Order("id desc").Error
	return records, err
}

// 查询状态为1且过期时间超过当前时间的所有额度记录的用户
func ExpiredCreditUsers() ([]User, error) {
	var users []User
	if err := DB.Joins("JOIN quota_records ON quota_records.user_id = users.id").
		Where("quota_records.status = ? AND quota_records.expired_time < ?", 1, time.Now().Unix()).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateRecordExpiredStatus 根据id更新额度记录状态为无效
func UpdateRecordExpiredStatus(id int) error {
	return DB.Model(&QuotaRecord{}).Where("id = ?", id).Update("status", 0).Error
}
