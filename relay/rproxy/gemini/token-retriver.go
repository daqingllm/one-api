package gemini

import (
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type GeminiTokenRetriever struct {
}

func (r *GeminiTokenRetriever) Retrieve(context *rproxy.RproxyContext) (token *model.Token, err *relaymodel.ErrorWithStatusCode) {
	key := context.SrcContext.Query("key")
	if key == "" {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "missing_key", "Path中缺少key参数")
	}
	key = strings.TrimPrefix(key, "sk-")
	parts := strings.Split(key, "-")
	key = parts[0]
	token, e := model.CacheGetTokenByKey(context.SrcContext, key)
	if e != nil {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusUnauthorized, "retrieve_token_failed", "获取Token失败")
	}
	// todo 这个地方解析和这个retriever语义不符合，可以放到初始化中解析，如果请求中携带了channelId，则使用指定的channelId
	if len(parts) > 1 {
		context.SrcContext.Set(ctxkey.SpecificChannelId, parts[1])
	}
	return token, nil
}
