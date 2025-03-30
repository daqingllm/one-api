package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	dbmodel "github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type DefaultChannelSelector struct {
}

func (c DefaultChannelSelector) SelectChannel(context *RproxyContext) (orderedChannels []*model.Channel, err *relaymodel.Error) {
	orderedChannels, error := dbmodel.CacheGetRandomSatisfiedChannels(context.GetGroup(), context.GetOriginalModel())
	if error != nil {
		//todo Error Handling
		return nil, &relaymodel.Error{}
	}
	return orderedChannels, nil

}
