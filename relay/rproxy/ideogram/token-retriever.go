package ideogram

import (
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type IdeoGramTokenRetriever struct {
}

func (r *IdeoGramTokenRetriever) Retrieve(context *rproxy.RproxyContext) (token *model.Token, err *relaymodel.ErrorWithStatusCode) {
	key := context.SrcContext.Request.Header.Get("Api-Key")
	key = strings.TrimPrefix(key, "sk-")
	parts := strings.Split(key, "-")
	key = parts[0]
	token, e := model.CacheGetTokenByKey(context.SrcContext, key)
	if e != nil {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusUnauthorized, "retrieve_token_failed", "获取Token失败")
	}
	// 如果请求中携带了channelId，则使用指定的channelId
	if len(parts) > 1 {
		context.SrcContext.Set(ctxkey.SpecificChannelId, parts[1])
	}
	return token, nil
}
