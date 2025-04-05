package oai

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type OAIAuthChecker struct {
	ctx *gin.Context
}

// check implements rproxy.AuthChecker.
func (authChecker *OAIAuthChecker) Check() (result bool, err *relaymodel.ErrorWithStatusCode) {
	key := authChecker.ctx.Request.Header.Get("Authorization")
	key = strings.TrimPrefix(key, "Bearer ")
	key = strings.TrimPrefix(key, "sk-")
	parts := strings.Split(key, "-")
	key = parts[0]
	_, e := model.ValidateUserToken(authChecker.ctx.Request.Context(), key)

	if e != nil {
		logger.Error(authChecker.ctx, e.Error())
		return false, nil
	}
	return true, nil
}
