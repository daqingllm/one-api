package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	claude_adaptor "github.com/songquanpeng/one-api/relay/claudeadaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	relay_model "github.com/songquanpeng/one-api/relay/model"
	"io"
	"math"
	"net/http"
)

func ClaudeMessages(c *gin.Context) {
	ctx := c.Request.Context()
	channelId := c.GetInt(ctxkey.ChannelId)
	userId := c.GetInt(ctxkey.Id)
	bizErr := relayTextHelper(c)
	if bizErr == nil {
		model.CacheSetRecentChannel(ctx, userId, c.GetString(ctxkey.RequestModel), channelId)
		monitor.Emit(channelId, true)
		return
	}
	model.CacheSetRecentChannel(ctx, userId, c.GetString(ctxkey.RequestModel), 0)
	channelName := c.GetString(ctxkey.ChannelName)
	group := c.GetString(ctxkey.Group)
	originalModel := c.GetString(ctxkey.OriginalModel)

	go processChannelRelayError(ctx, userId, channelId, channelName, bizErr)
	requestId := c.GetString(helper.RequestIdKey)
	retry := true
	if !shouldRetry(c, bizErr) {
		logger.Errorf(ctx, "relay error happen, won't retry in this case. biz: %+v", bizErr)
		retry = false
	}
	excludedChannels := make([]int, 0)
	excludedChannels = append(excludedChannels, channelId)
	for retry {
		retryChannel, err := model.CacheGetRandomSatisfiedChannel(group, originalModel, excludedChannels)
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
		bizErr = relayTextHelper(c)
		if bizErr == nil {
			return
		}
		excludedChannels = append(excludedChannels, retryChannel.Id)
		go processChannelRelayError(ctx, userId, retryChannel.Id, retryChannel.Name, bizErr)
	}

	requestBody, _ := common.GetRequestBody(c)
	copyError := relay_model.Error{
		Message: bizErr.Error.Message,
		Type:    bizErr.Error.Type,
		Param:   bizErr.Error.Param,
		Code:    bizErr.Error.Code,
	}
	responseError := anthropic.Error{
		Type:    bizErr.Error.Type,
		Message: bizErr.Error.Message,
	}
	if bizErr.StatusCode == http.StatusTooManyRequests {
		responseError.Message = "该模型遇到官方限速，联系客服增加并发或请稍后重试"
	}
	responseError.Message = helper.MessageWithRequestId(responseError.Message, requestId)
	c.JSON(bizErr.StatusCode, gin.H{
		"type":  "error",
		"error": responseError,
	})

	if bizErr.Code != "insufficient_user_quota" {
		go logRespError(ctx, userId, originalModel, excludedChannels, bizErr.StatusCode, copyError, string(requestBody), requestId, c.GetString(ctxkey.BaseURL))
	}
}

func relayTextHelper(c *gin.Context) *relay_model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	request, err := getAndValidateRequest(c, meta.Mode)
	if err != nil {
		logger.Errorf(ctx, "getAndValidateTextRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	meta.IsStream = request.Stream

	// map model name
	meta.OriginModelName = request.Model
	request.Model, _ = getMappedModelName(request.Model, meta.ModelMapping)
	meta.ActualModelName = request.Model
	// get model ratio & group ratio
	modelRatio := billingratio.GetModelRatio(request.Model, meta.ChannelType)
	groupRatio := billingratio.GetGroupRatio(meta.Group)
	ratio := modelRatio * groupRatio
	// pre-consume quota
	promptTokens := getPromptTokens(request)
	meta.PromptTokens = promptTokens
	bizErr := validQuota(ctx, request, promptTokens, ratio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "validQuota failed: %+v", *bizErr)
		return bizErr
	}

	adaptor := getAdaptor(meta.APIType)
	if adaptor == nil {
		logger.Errorf(ctx, "getAdaptor failed: %d", meta.APIType)
		return openai.ErrorWrapper(errors.New("model is not supported for claude api"), "in", http.StatusBadRequest)
	}
	usage, bizError := adaptor.DoRequest(c, request, meta)
	if bizError != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", bizError)
		return bizError
	}

	go postConsumeQuota(ctx, usage, meta, request, ratio, modelRatio, groupRatio)
	return nil
}

func getPromptTokens(request *anthropic.Request) int {
	//todo calc input tokens
	return 1
}

func validQuota(ctx context.Context, request *anthropic.Request, promptTokens int, ratio float64, meta *meta.Meta) *relay_model.ErrorWithStatusCode {
	preConsumedQuota := getPreConsumedQuota(request, promptTokens, ratio)

	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
	if err != nil {
		return openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-preConsumedQuota < 0 {
		return openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	return nil
}

func getPreConsumedQuota(request *anthropic.Request, promptTokens int, ratio float64) int64 {
	preConsumedTokens := config.PreConsumedQuota + int64(promptTokens)
	if request.MaxTokens != 0 {
		preConsumedTokens += int64(request.MaxTokens)
	}
	return int64(float64(preConsumedTokens) * ratio)
}

func getAndValidateRequest(c *gin.Context, mode int) (*anthropic.Request, error) {
	request := &anthropic.Request{}
	err := common.UnmarshalBodyReusable(c, request)
	if err != nil {
		return nil, err
	}
	if request.Model == "" {
		return nil, errors.New("model is required")
	}

	return request, nil
}

func getMappedModelName(modelName string, mapping map[string]string) (string, bool) {
	if mapping == nil {
		return modelName, false
	}
	mappedModelName := mapping[modelName]
	if mappedModelName != "" {
		return mappedModelName, true
	}
	return modelName, false
}

func getAdaptor(apiType int) claude_adaptor.Adaptor {
	switch apiType {
	case apitype.Anthropic:
		return &claude_adaptor.Anthropic{}
	case apitype.AwsClaude:
		return &claude_adaptor.Aws{}
	case apitype.VertexAI:
		return &claude_adaptor.Vertextai{}
	default:
		return nil
	}
}

func postConsumeQuota(ctx context.Context, usage *anthropic.Usage, meta *meta.Meta, textRequest *anthropic.Request, ratio float64, modelRatio float64, groupRatio float64) {
	if usage == nil {
		logger.Error(ctx, "usage is nil, which is unexpected")
		return
	}
	var quota int64
	completionRatio := billingratio.GetCompletionRatio(textRequest.Model, meta.ChannelType)

	quota = int64(math.Ceil((float64(usage.InputTokens) + float64(usage.CacheCreationInputTokens)*1.25 +
		float64(usage.CacheReadInputTokens)*0.1 + float64(usage.OutputTokens)*completionRatio) * ratio))
	if ratio != 0 && quota <= 0 {
		quota = 1
	}
	totalTokens := usage.InputTokens + usage.OutputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
	if totalTokens == 0 {
		// in this case, must be some error happened
		// we cannot just return, because we may have to return the pre-consumed quota
		quota = 0
	}
	err := model.PostConsumeTokenQuota(meta.TokenId, quota)
	if err != nil {
		logger.Error(ctx, "error consuming token remain quota: "+err.Error())
	}
	err = model.CacheUpdateUserQuota(ctx, meta.UserId)
	if err != nil {
		logger.Error(ctx, "error update user quota cache: "+err.Error())
	}
	logContent := fmt.Sprintf("模型倍率 %.3f，分组倍率 %.3f，补全倍率 %.3f", modelRatio, groupRatio, completionRatio)
	model.RecordConsumeLog(ctx, meta.UserId, meta.ChannelId, usage.InputTokens+usage.CacheCreationInputTokens+usage.CacheReadInputTokens, usage.CacheReadInputTokens, usage.OutputTokens, textRequest.Model, meta.TokenName, quota, logContent)
	model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
	model.UpdateChannelUsedQuota(meta.ChannelId, quota)
}
