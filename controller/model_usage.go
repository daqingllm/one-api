package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

type ModelUsageItem struct {
	Date      string `json:"date"`
	CallCount int64  `json:"call_count"`
	TokenUsed int64  `json:"token_used"`
	Model     string `json:"model"`
}

func RefreshModelUsage(c *gin.Context) {
	ctx := c.Request.Context()
	lastdaysStr := c.Query("lastdays")
	lastdays, err := strconv.Atoi(lastdaysStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if lastdays == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
		})
		return
	}
	err = model.RefreshModelUsage(ctx, lastdays)
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

func GetModelUsageDetail(context *gin.Context) {
	ctx := context.Request.Context()
	dayStr := context.Query("day")
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		day = 30
	}
	if day == 0 {
		day = 30
	}
	endDate := context.Query("end_date")
	modelUsages, err := model.GetModelUsageDetail(ctx, day, endDate)
	if err != nil {
		context.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	modelUsageItems := make([]ModelUsageItem, 0, len(modelUsages))
	for _, modelUsage := range modelUsages {
		modelUsageItems = append(modelUsageItems, ModelUsageItem{
			Date:      modelUsage.Date.Format("2006-01-02"),
			CallCount: int64(modelUsage.CallCount),
			TokenUsed: int64(modelUsage.TokenUsed),
			Model:     modelUsage.ModelName,
		})
	}
	context.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    modelUsageItems,
	})
}

func GetModelUsageCount(context *gin.Context) {
	ctx := context.Request.Context()
	period := context.Query("period")
	var date time.Time
	location, _ := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	now := time.Now().In(location)
	switch period {
	case "daily":
		date = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	case "weekly":
		date = time.Date(now.Year(), now.Month(), now.Day()-7, 0, 0, 0, 0, now.Location())
	case "monthly":
		date = time.Date(now.Year(), now.Month()-1, now.Day(), 0, 0, 0, 0, now.Location())
	default:
		date = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	}
	modelUsageCounts, err := model.GetModelUsageCount(ctx, date)
	if err != nil {
		context.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    modelUsageCounts,
	})
}
