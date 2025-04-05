package common

import (
	"net/http"

	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type TokenValidator struct {
	ctx *rproxy.RproxyContext
}

func (t *TokenValidator) Validate() *relaymodel.ErrorWithStatusCode {
	err := model.ValidateToken(t.ctx.Token)
	if err != nil {
		return relaymodel.NewErrorWithStatusCode(http.StatusUnauthorized, err.Error(), err.Error())
	}
	return nil
}
