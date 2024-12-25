package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

func QuotaUseTest(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	quota := c.Query("quota")
	quotaInt, err := strconv.ParseInt(quota, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的配额参数",
		})
		return
	}
	logger.Info(c, "quotaInt: "+strconv.FormatInt(quotaInt, 10))
	err = model.QuotaUseTest(userId, quotaInt)
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
	})
}
