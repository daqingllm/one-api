package common

import (
	"net/http"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/util"
)

type GenerationRespHandler struct {
}

func (r *GenerationRespHandler) Handle(context *rproxy.RproxyContext, resp rproxy.Response) (err *relaymodel.ErrorWithStatusCode) {
	httpResp, ok := resp.(*http.Response)
	if !ok {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "invalid_response", "invalid_response")
	}
	if context.Meta.IsStream {
		_, err = util.StreamResponseHandler(context.SrcContext, httpResp)

	} else {
		_, err = util.ResponseHandler(context.SrcContext, httpResp)
	}
	if err != nil {
		return err
	}
	return nil
}
