package common

import (
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type GenerateRespHandler struct {
}

func (r *GenerateRespHandler) Handle(context *rproxy.RproxyContext, resp rproxy.Response) (err *relaymodel.ErrorWithStatusCode) {
	// httpResp, ok := resp.(*http.Response)
	// if !ok {
	// 	return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "invalid_response", "invalid_response")
	// }
	panic("not implemented")

}
