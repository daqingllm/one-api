package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type HandlerFunc func(channel *model.Channel, context *RproxyContext) *relaymodel.Error

type FailOverTolerancer struct {
	channels []*model.Channel
	selector ChannelSelector
	handler  HandlerFunc
}

func NewFailOverTolerancer(selector ChannelSelector, handler HandlerFunc) *FailOverTolerancer {
	return &FailOverTolerancer{
		channels: nil,
		selector: selector,
		handler:  handler,
	}
}

func (f *FailOverTolerancer) FaultTolerance(context *RproxyContext) (err *relaymodel.ErrorWithStatusCode) {
	orderedChannels, e := f.selector.SelectChannel(context)
	if e != nil {
		panic("")
	}
	f.channels = orderedChannels
	for _, channel := range f.channels {
		if channel.Status != 1 {
			continue
		}
		err := f.handler(channel, context)
		if err == nil {
			break
		}

	}
	return nil
}
