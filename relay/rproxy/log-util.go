package rproxy

import (
	"encoding/json"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	dbmodel "github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func LogRespError(ctx *RproxyContext, channels []*model.Channel, err *relaymodel.ErrorWithStatusCode) {
	userId := ctx.GetUserId()
	originalModel := ctx.GetOriginalModel()
	channelIds := make([]int, 0, len(channels))
	for _, channel := range channels {
		channelIds = append(channelIds, channel.Id)
	}
	logger.Errorf(ctx.SrcContext, "relay error (user id: %d, model: %s, channels: %v): %s", userId, originalModel, channelIds, err.Message)
	if config.IsZiai {
		return
	}
	channelsData, _ := json.Marshal(channelIds)
	respData, _ := json.Marshal(err)
	var requestBody string
	if resolvedReq := ctx.ResolvedRequest; resolvedReq != nil {
		if bodyBytes, ok := resolvedReq.([]byte); ok {
			requestBody = string(bodyBytes)
		} else {
			requestBody = ""
		}
	}

	dbmodel.RecordFailedLog(ctx.SrcContext, userId, originalModel, string(channelsData), err.StatusCode, string(respData), requestBody, ctx.SrcContext.GetString(helper.RequestIdKey), ctx.SrcContext.Request.URL.Path)
}
