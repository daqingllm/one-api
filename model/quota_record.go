package model

const (
	TypeRedemption = 1
	TypeTopup
)

type QuotaRecord struct {
	Id        int `json:"id"`
	UserId    int `json:"user_id" gorm:"index:index_userid_expiredat,priority:1"`
	GrantType int `json:"grant_type" gorm:"type:int;default:1"`
	//GrantId   int64 `json:"grant_id" gorm:"bigint"`
	ExpiredAt int64 `json:"expired_at" gorm:"bigint;index:index_userid_expiredat,priority:2"`
	CreateAt  int64 `json:"created_at" gorm:"bigint"`
}
