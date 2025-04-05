package common

import (
	"net/http"

	"github.com/songquanpeng/one-api/common/blacklist"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type BlacklistValidator struct {
	ctx *rproxy.RproxyContext
}

func (b *BlacklistValidator) Validate() *relaymodel.ErrorWithStatusCode {
	userEnabled, err := model.CacheIsUserEnabled(b.ctx.Token.UserId)
	if err != nil {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, err.Error(), err.Error())
	}
	if !userEnabled || blacklist.IsUserBanned(b.ctx.Token.UserId) {
		return relaymodel.NewErrorWithStatusCode(http.StatusForbidden, nil, "用户已被封禁")
	}
	return nil
}
