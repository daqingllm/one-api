package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"net/http"
	"strconv"
)

func DistributeClaude() func(c *gin.Context) {
	return func(c *gin.Context) {
		userId := c.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(c.Request.Context(), userId)
		c.Set(ctxkey.Group, userGroup)
		var requestModel string
		var channel *model.Channel
		channelId, ok := c.Get(ctxkey.SpecificChannelId)
		if ok {
			id, err := strconv.Atoi(channelId.(string))
			if err != nil {
				abortWithMessageClaude(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			channel, err = model.CacheGetChannelById(id)
			if err != nil {
				abortWithMessageClaude(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
		} else {
			requestModel = c.GetString(ctxkey.RequestModel)
			var err error
			recentChannelId := model.CacheGetRecentChannel(c.Request.Context(), userId, requestModel)
			if recentChannelId > 0 {
				channel, err = model.CacheGetChannelById(recentChannelId)
				if err == nil {
					SetupContextForSelectedChannel(c, channel, requestModel)
					c.Next()
					return
				}
			}
			channel, err = model.CacheGetRandomSatisfiedChannel(userGroup, requestModel, nil)
			if err != nil {
				message := fmt.Sprintf("模型名字错误请求进入模型广场查看或当前分组 %s 下对于模型 %s 无使用权限", userGroup, requestModel)
				if channel != nil {
					logger.SysError(fmt.Sprintf("渠道不存在：%d", channel.Id))
					message = "数据库一致性已被破坏，请联系管理员"
				}
				abortWithMessageClaude(c, http.StatusServiceUnavailable, message)
				return
			}
		}
		SetupContextForSelectedChannel(c, channel, requestModel)
		c.Next()
	}
}
