package model

import (
	"time"

	"gorm.io/gorm"
)

// OrderRecord 订单记录 借用quota_record的表结构
// 交易状态：WAIT_BUYER_PAY（交易创建，等待买家付款）、TRADE_CLOSED（未付款交易超时关闭，或支付完成后全额退款）、TRADE_SUCCESS（交易支付成功）、TRADE_FINISHED（交易结束，不可退款）

type OrderRecord struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index:index_userid_expiredat,priority:1"`
	GrantType int    `json:"grant_type" gorm:"type:int;default:1"`
	TradeNo   string `json:"trade_no"`
	ExpiredAt int64  `json:"expired_at" gorm:"bigint;index:index_userid_expiredat,priority:2"`
	CreateAt  int64  `json:"created_at" gorm:"bigint"`
	Status    string `json:"status"`
	Quota     int64  `json:"quota" gorm:"bigint"`
}

func (record *OrderRecord) Insert() error {
	var err error
	err = DB.Create(record).Error
	if err != nil {
		return err
	}
	return err
}

func UpdateOrderStatusByTradeNo(tradeNo string, status string) error {
	var err error
	err = DB.Model(&OrderRecord{}).Where("trade_no = ?", tradeNo).Update("status", status).Error
	if err != nil {
		return err
	}
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
	var err error
	err = DB.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", quota)).Error
	if err != nil {
		return err
	}
	return err
}

// GetOrdersByStatus 获取所有等待付款的订单
func GetOrdersByStatus(status string) ([]*OrderRecord, error) {
	var orders []*OrderRecord
	err := DB.Where("status = ?", status).Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}
