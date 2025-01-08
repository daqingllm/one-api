package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/helper"
	"time"
)

func RelayTime() func(c *gin.Context) {
	return func(c *gin.Context) {
		st := time.Now().UnixMilli()
		c.Set(helper.StartTimeKey, st)
		ctx := context.WithValue(c.Request.Context(), helper.StartTimeKey, st)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
