package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

// https://platform.openai.com/docs/api-reference/chat

func relayHelper(c *gin.Context, relayMode int) *model.ErrorWithStatusCode {
	var err *model.ErrorWithStatusCode
	switch relayMode {
	case relaymode.ImagesGenerations, relaymode.ImagesEdits, relaymode.ImageVariations:
		err = controller.RelayImageHelper(c, relayMode)
	case relaymode.AudioSpeech:
		fallthrough
	case relaymode.AudioTranslation:
		fallthrough
	case relaymode.AudioTranscription:
		err = controller.RelayAudioHelper(c, relayMode)
	case relaymode.Rerank:
		err = controller.RelayRerankHelper(c, relayMode)
	case relaymode.Proxy:
		err = controller.RelayProxyHelper(c, relayMode)
	default:
		err = controller.RelayTextHelper(c)
	}
	return err
}

func Relay(c *gin.Context) {
	ctx := c.Request.Context()
	relayMode := relaymode.GetByPath(c.Request.URL.Path)
	if config.DebugEnabled {
		requestBody, _ := common.GetRequestBody(c)
		logger.Debugf(ctx, "request body: %s", string(requestBody))
	}
	channelId := c.GetInt(ctxkey.ChannelId)
	userId := c.GetInt(ctxkey.Id)
	bizErr := relayHelper(c, relayMode)
	if bizErr == nil {
		dbmodel.CacheSetRecentChannel(ctx, userId, c.GetString(ctxkey.RequestModel), channelId)
		monitor.Emit(channelId, true)
		return
	}
	channelName := c.GetString(ctxkey.ChannelName)
	group := c.GetString(ctxkey.Group)
	originalModel := c.GetString(ctxkey.OriginalModel)
	go processChannelRelayError(ctx, userId, channelId, channelName, bizErr)
	requestId := c.GetString(helper.RequestIdKey)
	retry := true
	if !shouldRetry(c, bizErr) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		retry = false
	}
	excludedChannels := make([]int, 0)
	excludedChannels = append(excludedChannels, channelId)
	for retry {
		retryChannel, err := dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, excludedChannels)
		if err != nil {
			logger.Errorf(ctx, "CacheGetRandomSatisfiedChannel failed: %+v", err)
			break
		}
		if retryChannel == nil {
			logger.Errorf(ctx, "All channels have been tried, no more channel to try")
			break
		}
		logger.Infof(ctx, "using channel #%d to retry (remain times %d)", retryChannel.Id, len(excludedChannels))
		middleware.SetupContextForSelectedChannel(c, retryChannel, originalModel)
		requestBody, err := common.GetRequestBody(c)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		bizErr = relayHelper(c, relayMode)
		if bizErr == nil {
			return
		}
		excludedChannels = append(excludedChannels, retryChannel.Id)
		go processChannelRelayError(ctx, userId, retryChannel.Id, retryChannel.Name, bizErr)
	}

	requestBody, _ := common.GetRequestBody(c)
	responseError := model.Error{
		Message: bizErr.Error.Message,
		Type:    bizErr.Error.Type,
		Param:   bizErr.Error.Param,
		Code:    bizErr.Error.Code,
	}
	if bizErr.StatusCode == http.StatusTooManyRequests {
		responseError.Message = "当前分组上游负载已饱和，联系客服或请稍后重试"
	}
	responseError.Message = helper.MessageWithRequestId(responseError.Message, requestId)
	c.JSON(bizErr.StatusCode, gin.H{
		"error": responseError,
	})

	go logRespError(ctx, userId, originalModel, excludedChannels, bizErr.StatusCode, responseError, string(requestBody))
}

func logRespError(ctx context.Context, userId int, originalModel string, channels []int, statusCode int, responseError model.Error, requestBody string) {
	logger.Errorf(ctx, "relay error (user id: %d, model: %s, channels: %v): %s", userId, originalModel, channels, responseError.Message)
	channelsData, _ := json.Marshal(channels)
	respData, _ := json.Marshal(responseError)
	dbmodel.RecordFailedLog(userId, originalModel, string(channelsData), statusCode, string(respData), requestBody)
}

func shouldRetry(c *gin.Context, bizError *model.ErrorWithStatusCode) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	if !bizError.IsChannelResponseError {
		return false
	}
	if bizError.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if bizError.StatusCode/100 == 5 {
		return true
	}
	if bizError.StatusCode == http.StatusBadRequest {
		return false
	}
	if bizError.StatusCode/100 == 2 {
		return false
	}
	return true
}

func processChannelRelayError(ctx context.Context, userId int, channelId int, channelName string, err *model.ErrorWithStatusCode) {
	logger.Errorf(ctx, "relay error (channel id %d, user id: %d): %s", channelId, userId, err.Message)
	// https://platform.openai.com/docs/guides/error-codes/api-errors
	if monitor.ShouldDisableChannel(err, err.StatusCode) {
		monitor.DisableChannel(channelId, channelName, err.Message)
	} else {
		monitor.Emit(channelId, false)
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := model.Error{
		Message: "API not implemented",
		Type:    "Aihubmix_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := model.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
