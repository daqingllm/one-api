package common

import (
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type AuthorizationTokenRetriever struct {
}

func (r *AuthorizationTokenRetriever) Retrieve(context *rproxy.RproxyContext) (token *model.Token, err *relaymodel.ErrorWithStatusCode) {
	key := context.SrcContext.Request.Header.Get("Authorization")
	key = strings.TrimPrefix(key, "Bearer ")
	key = strings.TrimPrefix(key, "sk-")
	parts := strings.Split(key, "-")
	key = parts[0]
	token, e := model.CacheGetTokenByKey(context.SrcContext, key)
	if e != nil {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusUnauthorized, "retrieve_token_failed", "获取Token失败")
	}
	return token, nil
}
