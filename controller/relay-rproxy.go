package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

// https://platform.openai.com/docs/api-reference/chat

func RelayRProxy(weaverFactory rproxy.WeaverFactory) gin.HandlerFunc {
	return func(c *gin.Context) {
		weaverFactory.GetWeaver(c).Weave()
	}
}
