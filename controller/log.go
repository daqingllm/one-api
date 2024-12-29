package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"net/http"
	"strconv"
	"time"
)

func GetAllLogs(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	num := config.ItemsPerPage
	var err error
	if c.Query("num") != "" {
		num, err = strconv.Atoi(c.Query("num"))
		if err != nil {
			num = config.ItemsPerPage
		}
	}
	logs, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, p*num, num, channel)
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
		"data":    logs,
	})
	return
}

func GetUserLogs(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	userId := c.GetInt(ctxkey.Id)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	num := config.ItemsPerPage
	var err error
	if c.Query("num") != "" {
		num, err = strconv.Atoi(c.Query("num"))
		if err != nil {
			num = config.ItemsPerPage
		}
	}
	logs, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, p*num, num)
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
		"data":    logs,
	})
	return
}

func SearchAllLogs(c *gin.Context) {
	keyword := c.Query("keyword")
	logs, err := model.SearchAllLogs(keyword)
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
		"data":    logs,
	})
	return
}

func SearchUserLogs(c *gin.Context) {
	keyword := c.Query("keyword")
	userId := c.GetInt(ctxkey.Id)
	logs, err := model.SearchUserLogs(userId, keyword)
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
		"data":    logs,
	})
	return
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	quotaNum := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel)
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum,
			//"token": tokenNum,
		},
	})
	return
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString(ctxkey.Username)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	quotaNum := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel)
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum,
			//"token": tokenNum,
		},
	})
	return
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(targetTimestamp)
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
		"data":    count,
	})
	return
}

func GetUserUsage(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	startTime, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	userIdStr := c.Query("user_id")
	if userIdStr != "" {
		role := c.GetInt(ctxkey.Role)
		if role >= model.RoleAdminUser {
			userId, _ = strconv.Atoi(userIdStr)
		}
	}
	usages, err := model.GetUsage(userId, modelName, tokenName, int(startTime), int(endTime))
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
		"data":    usages,
	})
	return
}

func FlushUserUsage(c *gin.Context) {
	startId64, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	startId := int(startId64)

	month64, _ := strconv.ParseInt(c.Query("month"), 10, 64)
	day64, _ := strconv.ParseInt(c.Query("day"), 10, 64)
	location, _ := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	endTime := time.Date(2024, time.Month(int(month64)), int(day64), 0, 0, 0, 0, location)

	for startId > 0 {
		logs, err := model.PaginateLogs(startId, 1000)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		usages := make(map[string]*model.Usage, 0)
		for _, log := range logs {
			createdAt := time.Unix(log.CreatedAt, 0)
			if createdAt.Before(endTime) {
				startId = 0
				break
			}
			hourStr := createdAt.Format("2006010215")
			hour, _ := strconv.Atoi(hourStr)
			key := strconv.Itoa(log.UserId) + log.ModelName + log.TokenName + hourStr
			if _, ok := usages[key]; !ok {
				usages[key] = &model.Usage{
					UserId:       log.UserId,
					Hour:         hour,
					ModelName:    log.ModelName,
					TokenName:    log.TokenName,
					Count:        0,
					InputTokens:  0,
					OutputTokens: 0,
					Quota:        0,
				}
			}
			usage := usages[key]
			usage.Count++
			usage.InputTokens += log.PromptTokens
			usage.OutputTokens += log.CompletionTokens
			usage.Quota += log.Quota
		}
		for _, usage := range usages {
			err = model.AddUsage(usage.UserId, usage.ModelName, usage.Hour, usage.TokenName, usage.Count, usage.InputTokens, usage.OutputTokens, usage.Quota)
			if err != nil {
				logger.SysError("failed to add usage: " + err.Error())
			}
		}
		if startId != 0 {
			startId = logs[len(logs)-1].Id
		}
	}
}
