package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
)

func CreateQuotaRecord(record *model.OrderRecord) error {
	err := model.AddQuotaRecord(record.UserId, record.GrantType, record.TradeNo, record.Quota)
	if err != nil {
		return err
	}
	return err
}

// 获取用户充值记录
func GetUserQuotaRecords(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	num := config.ItemsPerPage
	var err error
	if c.Query("num") != "" {
		num, err = strconv.Atoi(c.Query("num"))
		if err != nil {
			num = config.ItemsPerPage
		}
	}
	records, err := model.GetQuotaRecordsByUserId(c.GetInt("id"), p*num, num)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    records,
	})
}

func CheckAndUpdateExpiredQuota(c *gin.Context) {
	users, err := model.ExpiredCreditUsers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Error getting expired credit users: " + err.Error(),
		})
		return
	}
	for _, user := range users {
		records, err := model.GetUserValidRecords(user.Id)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "Error getting user valid records:" + err.Error(),
			})
			return
		}

		var totalExpiredQuota int64
		var totalQuota int64
		var usedQuota int64

		for _, record := range records {
			totalQuota += record.Quota
			if time.Unix(record.ExpiredTime, 0).Before(time.Now()) {
				totalExpiredQuota += record.Quota
				err = model.UpdateRecordExpiredStatus(record.Id)
				if err != nil {
					c.JSON(http.StatusOK, gin.H{
						"success": false,
						"message": "Error updating expired status: " + err.Error(),
					})
					return
				}
			}
		}
		usedQuota = totalQuota - user.Quota
		if usedQuota < totalExpiredQuota {
			err = model.UpdateUserQuota(user.Id, usedQuota-totalExpiredQuota)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Error updating user quota:  " + err.Error(),
				})
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": len(users),
	})
}
