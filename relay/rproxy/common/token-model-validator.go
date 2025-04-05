package common

import (
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type TokenModelValidator struct {
	ctx *rproxy.RproxyContext
}

func (this *TokenModelValidator) Validate() *relaymodel.ErrorWithStatusCode {
	if this.ctx.Token.Models != nil && *this.ctx.Token.Models != "" {
		if this.ctx.GetOriginalModel() != "" && !isModelInList(this.ctx.GetOriginalModel(), *this.ctx.Token.Models) {
			return relaymodel.NewErrorWithStatusCode(http.StatusForbidden, nil, "该令牌不支持该模型")
		}
	}
	return nil
}

func isModelInList(modelName string, models string) bool {
	modelList := strings.Split(models, ",")
	for _, model := range modelList {
		if modelName == model {
			return true
		}
	}
	return false
}

