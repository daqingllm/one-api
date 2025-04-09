package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

// https://platform.openai.com/docs/api-reference/chat

func RelayRProxy(weaverFactoryFunc func() rproxy.WeaverFactory) gin.HandlerFunc {
	return func(c *gin.Context) {
		weaverFactory := weaverFactoryFunc()
		err := weaverFactory.GetWeaver(c).Weave()
		if err != nil {
			c.JSON(err.StatusCode, gin.H{
				"type": "error",
				"error": gin.H{
					"message": helper.MessageWithRequestId(err.Message, c.GetString(helper.RequestIdKey)),
					"type":    "Aihubmix_api_error",
				},
			})
			c.Abort()
			logger.Error(c.Request.Context(), err.Message)
		}
	}
}
