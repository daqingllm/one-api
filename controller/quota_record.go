package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/model"
)

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
