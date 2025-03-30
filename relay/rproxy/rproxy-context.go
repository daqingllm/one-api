package rproxy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/meta"
)

type RproxyContext struct {
	Meta       *meta.Meta
	SrcContext *gin.Context
	group      string
}

func (c *RproxyContext) GetGroup() string {
	if c.group == "" {
		userId := c.SrcContext.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(c.SrcContext.Request.Context(), userId)
		c.group = userGroup
	}
	return c.group
}

func (c *RproxyContext) GetUserId() int {
	return c.SrcContext.GetInt(ctxkey.Id)
}

func (c *RproxyContext) GetOriginalModel() string {
	return c.SrcContext.GetString(ctxkey.Group)
}

func (c *RproxyContext) GetRequest() *http.Request {
	return c.SrcContext.Request
}

func (c *RproxyContext) IsStream() bool {
	return c.Meta.IsStream
}
