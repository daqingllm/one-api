package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type ChannelSelector interface {
	SelectChannel(context *RproxyContext) (orderedChannels []*model.Channel, err *relaymodel.ErrorWithStatusCode)
}
