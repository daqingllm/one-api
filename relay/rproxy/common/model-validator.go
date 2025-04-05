package common

import (
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type ModelValidator struct {
	ctx             *rproxy.RproxyContext
	supportedModels []string
}

func (m *ModelValidator) Validate() *model.ErrorWithStatusCode {
	if m.ctx.GetOriginalModel() != "" {
		if !isModelInList(m.ctx.GetOriginalModel(), strings.Join(m.supportedModels, ",")) {
			return model.NewErrorWithStatusCode(http.StatusForbidden, nil, "该模型不支持")
		}
	}
	return nil
}
