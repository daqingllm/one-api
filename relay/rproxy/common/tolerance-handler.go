package common

import (
	"net/http"
	"strconv"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type ToleranceHandler struct {
}

func NewToleranceHandler() *ToleranceHandler {
	return &ToleranceHandler{}
}
func (t *ToleranceHandler) Handle(channel *model.Channel, context *rproxy.RproxyContext) (err *relaymodel.ErrorWithStatusCode) {
	logger.SysLogf("tolerance handler channel %v", channel)
	adaptor := rproxy.GetChannelAdaptorRegistry().GetAdaptor(context.SrcContext.Request.URL.Path, context.SrcContext.Request.Method, strconv.Itoa(channel.Type))
	logger.SysLogf("tolerance handler adaptor %v", adaptor)
	if adaptor == nil {
		logger.Errorf(context.SrcContext, "get_adaptor_failed channel %v", channel)
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "get_adaptor_failed", "get_adaptor_failed")
	}
	adaptor.SetChannel(channel)
	_, err = adaptor.DoRequest(context)
	return
}
