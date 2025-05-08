package model

import (
	"time"
)

// OrderRecord 订单记录 借用quota_record的表结构
// 交易状态：WAIT_BUYER_PAY（交易创建，等待买家付款）、TRADE_CLOSED（未付款交易超时关闭，或支付完成后全额退款）、TRADE_SUCCESS（交易支付成功）、TRADE_FINISHED（交易结束，不可退款）

type OrderRecord struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index:index_userid_expiredat,priority:1"`
	GrantType int    `json:"grant_type" gorm:"type:int;default:1"`
	TradeNo   string `json:"trade_no" gorm:"unique"`
	ExpiredAt int64  `json:"expired_at" gorm:"bigint;index:index_userid_expiredat,priority:2"`
	CreateAt  int64  `json:"created_at" gorm:"bigint"`
	Status    string `json:"status"`
	Quota     int64  `json:"quota" gorm:"bigint"`
}

func (record *OrderRecord) Insert() error {
	err := DB.Create(record).Error
	return err
}

func UpdateOrderStatusByTradeNo(tradeNo string, status string) error {
	err := DB.Model(&OrderRecord{}).Where("trade_no = ?", tradeNo).Update("status", status).Error
	return err
}

// GetOrderByGrantId 根据trade_no获取订单记录
func GetOrderByTradeNo(tradeNo string) (*OrderRecord, error) {
	var record OrderRecord
	err := DB.Where("trade_no = ?", tradeNo).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// 查询订单是否过期
func (record *OrderRecord) IsOrderExpired() bool {
	return record.ExpiredAt < time.Now().Unix()
}

// 更新用户额度
func UpdateUserQuota(userId int, quota int64) error {
	err := DB.Model(&User{}).Where("id = ?", userId).Update("quota", quota).Error
	return err
}

// GetOrdersByStatus 获取用户48小时内未完成订单
func GetOrdersByStatusByUserId(userId int, status string, expiredAt int64) ([]*OrderRecord, error) {
	var orders []*OrderRecord
	err := DB.Where("user_id = ? AND status = ? AND create_at > ?", userId, status, expiredAt).Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}
