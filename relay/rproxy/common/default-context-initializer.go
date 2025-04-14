package common

import (
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/tidwall/gjson"
)

type DefaultContextInitializer struct {
	tokenRetrierver rproxy.TokenRetriever
	modelRetrierver rproxy.ModelRetriever
}

func (c *DefaultContextInitializer) Initialize(context *rproxy.RproxyContext) (err *relaymodel.ErrorWithStatusCode) {
	//设置token
	token, err := c.tokenRetrierver.Retrieve(context)
	if err != nil {
		return err
	}
	context.Token = token

	//设置模型
	modelName, err := c.modelRetrierver.Retrieve(context)
	if err != nil {
		return err
	}

	//设置用户组信息
	userGroup, _ := model.CacheGetUserGroup(context.SrcContext.Request.Context(), token.UserId)
	context.UserGroup = userGroup

	//兼容老逻辑,先设置gin.context
	context.SrcContext.Set(ctxkey.RequestModel, modelName)
	context.SrcContext.Set(ctxkey.Group, userGroup)

	context.SrcContext.Set(ctxkey.Id, token.UserId)
	context.SrcContext.Set(ctxkey.TokenId, token.Id)
	context.SrcContext.Set(ctxkey.TokenName, token.Name)
	if channelId := context.SrcContext.Param("channelid"); channelId != "" {
		context.SrcContext.Set(ctxkey.SpecificChannelId, channelId)
	}
	context.Meta = meta.GetByContext(context.SrcContext)
	if context.ResolvedRequest != nil {
		if err := context.ResolvedRequest.([]byte); err != nil {
			context.Meta.IsStream = gjson.GetBytes(context.ResolvedRequest.([]byte), "stream").Bool()
		}
	}
	return nil
}

func (c *DefaultContextInitializer) GetTokenRetriever() rproxy.TokenRetriever {
	return c.tokenRetrierver
}

func (c *DefaultContextInitializer) GetModelRetriever() rproxy.ModelRetriever {
	return c.modelRetrierver
}
