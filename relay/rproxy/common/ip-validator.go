package common

import (
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/common/network"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type IpValidator struct {
	ctx *rproxy.RproxyContext
}

func (v *IpValidator) Validate() *relaymodel.ErrorWithStatusCode {
	token := v.ctx.Token
	if token.Subnet != nil && *token.Subnet != "" {
		if !network.IsIpInSubnets(v.ctx.SrcContext, v.ctx.SrcContext.ClientIP(), *token.Subnet) {
			return relaymodel.NewErrorWithStatusCode(http.StatusForbidden, nil, fmt.Sprintf("该令牌只能在指定网段使用：%s，当前 ip：%s", *token.Subnet, v.ctx.SrcContext.ClientIP()))
		}
	}
	return nil
}
