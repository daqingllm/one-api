package rproxy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/meta"
)

type Ratio struct {
	ModelRatio int
	GroupRatio int
}

type RproxyContext struct {
	Meta             *meta.Meta
	SrcContext       *gin.Context
	UserGroup        string
	Token            *model.Token
	Ratio            *Ratio
	ResolvedRequest  any
	ResolvedResponse any
}

func (c *RproxyContext) GetGroup() string {
	if c.UserGroup == "" {
		userId := c.SrcContext.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(c.SrcContext.Request.Context(), userId)
		c.UserGroup = userGroup
	}
	return c.UserGroup
}

func (c *RproxyContext) GetRatio() *Ratio {
	if c.Ratio == nil {
		c.Ratio = &Ratio{
			ModelRatio: 1,
			GroupRatio: 1,
		}
	}
	return c.Ratio
}

func (c *RproxyContext) GetMeta() *meta.Meta {
	return c.Meta
}
func (c *RproxyContext) GetSpecialChannelId() string {
	return c.SrcContext.GetString(ctxkey.SpecificChannelId)
}
func (c *RproxyContext) GetToken() *model.Token {

	return c.Token
}
func (c *RproxyContext) GetUserId() int {
	return c.SrcContext.GetInt(ctxkey.Id)
}

func (c *RproxyContext) GetOriginalModel() string {
	return c.Meta.OriginModelName
}

func (c *RproxyContext) GetRequest() *http.Request {
	return c.SrcContext.Request
}

func (c *RproxyContext) IsStream() bool {
	return c.Meta.IsStream
}

func (c *RproxyContext) GetRequestModel() string {
	return c.Meta.OriginModelName
}
