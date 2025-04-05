package rproxy

import (
	"net/http"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type FailOverTolerancer struct {
	channels []*model.Channel
	selector ChannelSelector
	handler  Handler
}

func NewFailOverTolerancer(selector ChannelSelector, handler Handler) *FailOverTolerancer {
	return &FailOverTolerancer{
		channels: nil,
		selector: selector,
		handler:  handler,
	}
}

func (f *FailOverTolerancer) FaultTolerance(context *RproxyContext) (err *relaymodel.ErrorWithStatusCode) {
	orderedChannels, e := f.selector.SelectChannel(context)
	logger.Infof(context.SrcContext, "ordered channels len %d : %v", len(orderedChannels), orderedChannels)
	if e != nil {
		return e
	}
	if len(orderedChannels) == 0 {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "no_channel_available", "通道访问失败")

	}
	f.channels = orderedChannels
	for _, channel := range f.channels {
		if channel.Status != 1 {
			continue
		}
		e := f.handler.Handle(channel, context)
		if e != nil {
			return e
		}
	}
	return nil
}

func (f *FailOverTolerancer) GetHandler() Handler {
	return f.handler
}
