package common

import (
	"net/http"

	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type ChannelValidator struct {
	ctx *rproxy.RproxyContext
}

func (c *ChannelValidator) Validate() *relaymodel.ErrorWithStatusCode {
	channel := c.ctx.GetSpecialChannelId()
	if channel == "" {
		return nil
	}
	if model.IsAdmin(c.ctx.Token.UserId) {
		return relaymodel.NewErrorWithStatusCode(http.StatusForbidden, nil, "您没有权限访问该通道")
	}
	return nil
}
