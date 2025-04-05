package common

import (
	"net/http"
	"strconv"

	"github.com/songquanpeng/one-api/model"
	dbmodel "github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type DefaultChannelSelector struct {
}

func NewDefaultChannelSelector() *DefaultChannelSelector {
	return &DefaultChannelSelector{}
}
func (c DefaultChannelSelector) SelectChannel(context *rproxy.RproxyContext) (orderedChannels []*model.Channel, err *relaymodel.ErrorWithStatusCode) {
	context.GetSpecialChannelId()
	channelId := context.GetSpecialChannelId()
	if channelId != "" {
		id, e := strconv.Atoi(channelId)
		if e != nil {
			return nil, relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "special_channel_invalid", "无效的渠道 Id")
		}
		channel, e := model.CacheGetChannelById(id)
		if e != nil {
			return nil, relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "special_channel_invalid", "无效的渠道 Id")
		}
		return []*model.Channel{channel}, nil
	} else {
		orderedChannels = make([]*model.Channel, 0)
		recentChannelId := model.CacheGetRecentChannel(context.SrcContext.Request.Context(), context.GetUserId(), context.GetRequestModel())
		if recentChannelId > 0 {
			channel, err := model.CacheGetChannelById(recentChannelId)
			if err == nil {
				orderedChannels = append(orderedChannels, channel)
			}
		}
		//按照优先级获取渠道
		channels, err := dbmodel.CacheGetRandomSatisfiedChannels(context.GetGroup(), context.GetOriginalModel(), append(make([]int, 1), recentChannelId))
		if err != nil && len(orderedChannels) <= 0 {
			return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "no_valid_channel_error", "no_valid_channel_error")
		}
		orderedChannels = append(orderedChannels, channels...)
	}

	return orderedChannels, nil

}
