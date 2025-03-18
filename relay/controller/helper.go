package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/songquanpeng/one-api/relay/constant/role"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller/validator"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func getAndValidateTextRequest(c *gin.Context, relayMode int) (*relaymodel.GeneralOpenAIRequest, error) {
	textRequest := &relaymodel.GeneralOpenAIRequest{}
	err := common.UnmarshalBodyReusable(c, textRequest)
	if err != nil {
		return nil, err
	}
	if relayMode == relaymode.Moderations && textRequest.Model == "" {
		textRequest.Model = "text-moderation-latest"
	}
	if relayMode == relaymode.Embeddings && textRequest.Model == "" {
		textRequest.Model = c.Param("model")
	}
	err = validator.ValidateTextRequest(textRequest, relayMode)
	if err != nil {
		return nil, err
	}
	return textRequest, nil
}

func getPromptTokens(textRequest *relaymodel.GeneralOpenAIRequest, relayMode int) int {
	switch relayMode {
	case relaymode.ChatCompletions:
		return openai.CountTokenMessages(textRequest.Messages, textRequest.Model)
	case relaymode.Completions:
		return openai.CountTokenInput(textRequest.Prompt, textRequest.Model)
	case relaymode.Moderations:
		return openai.CountTokenInput(textRequest.Input, textRequest.Model)
	}
	return 0
}

func getPreConsumedQuota(textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64) int64 {
	preConsumedTokens := config.PreConsumedQuota + int64(promptTokens)
	if textRequest.MaxTokens != 0 {
		preConsumedTokens += int64(textRequest.MaxTokens)
	}
	return int64(float64(preConsumedTokens) * ratio)
}

func preConsumeQuota(ctx context.Context, textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64, meta *meta.Meta) (int64, *relaymodel.ErrorWithStatusCode) {
	preConsumedQuota := getPreConsumedQuota(textRequest, promptTokens, ratio)

	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-preConsumedQuota < 0 {
		return preConsumedQuota, openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	err = model.CacheDecreaseUserQuota(meta.UserId, preConsumedQuota)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}

	if preConsumedQuota > 0 {
		logger.Debug(ctx, fmt.Sprintf("user %d has quota %d, use %d, need to pre-consume", meta.UserId, userQuota, preConsumedQuota))
		err := model.PreConsumeTokenQuota(meta.TokenId, preConsumedQuota)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
	}
	return preConsumedQuota, nil
}

func postConsumeQuota(ctx context.Context, usage *relaymodel.Usage, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, ratio float64, preConsumedQuota int64, modelRatio float64, groupRatio float64, systemPromptReset bool) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(ctx, fmt.Sprintf("panic in postConsumeQuota: %v", r))
		}
	}()
	if usage == nil {
		logger.Error(ctx, "usage is nil, which is unexpected")
		return
	}
	var quota int64
	completionRatio := billingratio.GetCompletionRatio(textRequest.Model, meta.ChannelType)
	cacheRatio := billingratio.GetCacheRatio(textRequest.Model, meta.ChannelType)
	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens
	cachedTokens := 0
	audioPromptTokens := 0
	audioCompletionTokens := 0
	if usage.PromptTokensDetails != nil {
		if usage.PromptTokensDetails.CachedTokens != nil {
			cachedTokens = *usage.PromptTokensDetails.CachedTokens
		}
		if usage.PromptTokensDetails.AudioTokens != nil {
			audioPromptTokens = *usage.PromptTokensDetails.AudioTokens
		}
	}
	if usage.CompletionTokensDetails != nil {
		if usage.CompletionTokensDetails.AudioTokens != nil {
			audioCompletionTokens = *usage.CompletionTokensDetails.AudioTokens
		}
	}
	audioInputRatio, audioOutputRatio := 1.0, 1.0
	if audioPromptTokens > 0 || audioCompletionTokens > 0 {
		audioInputRatio, audioOutputRatio = billingratio.GetAudioRatios(textRequest.Model)
	}

	//if cacheRatio > 0 && cachedTokens > 0 {
	//	quota = int64(math.Ceil((float64(promptTokens-cachedTokens) + float64(cachedTokens)*cacheRatio + float64(completionTokens)*completionRatio) * ratio))
	//} else {
	//	quota = int64(math.Ceil((float64(promptTokens) + float64(completionTokens)*completionRatio) * ratio))
	//}
	quota = int64(math.Ceil(ratio *
		(float64(promptTokens-cachedTokens-audioPromptTokens) + // non-cached text prompt tokens
			float64(cachedTokens)*cacheRatio + // cached text prompt tokens
			float64(audioPromptTokens)*audioInputRatio + // audio prompt tokens
			float64(audioCompletionTokens)*audioOutputRatio + // audio completion tokens
			float64(completionTokens-audioCompletionTokens)*completionRatio))) // text completion tokens

	if ratio != 0 && quota <= 0 {
		quota = 1
	}
	var extraLog string

	//tools cost
	if meta.Extra["web_search"] == "true" {
		var searchQuota int64
		switch meta.ActualModelName {
		case "gpt-4o-search-preview":
			fallthrough
		case "gpt-4o":
			switch meta.Extra["search_context_size"] {
			case "low":
				searchQuota = 15000 // $30.00 1k calls
			case "medium":
				searchQuota = 17500 // $35.00 1k calls
			case "high":
				searchQuota = 25000 // $50.00 1k calls
			default:
				searchQuota = 17500 // medium
			}
		case "gpt-4o-mini":
			fallthrough
		case "gpt-4o-mini-search-preview":
			switch meta.Extra["search_context_size"] {
			case "low":
				searchQuota = 12500 // $25.00 1k calls
			case "medium":
				searchQuota = 13750 // $27.50 1k calls
			case "high":
				searchQuota = 15000 // $30.00 1k calls
			default:
				searchQuota = 13750 // medium
			}
		}
		if strings.HasPrefix(meta.ActualModelName, "gemini") {
			//$3.5 / 1K
			searchQuota = 1750
		}
		searchCost := float64(searchQuota) / 1000 * 0.002
		extraLog += fmt.Sprintf("Websearch费用$%.4f。", searchCost)
		quota += searchQuota
	}

	totalTokens := promptTokens + completionTokens
	if totalTokens == 0 {
		// in this case, must be some error happened
		// we cannot just return, because we may have to return the pre-consumed quota
		quota = 0
	}
	quotaDelta := quota - preConsumedQuota
	err := model.PostConsumeTokenQuota(meta.TokenId, quotaDelta)
	if err != nil {
		logger.Error(ctx, "error consuming token remain quota: "+err.Error())
	}
	err = model.CacheUpdateUserQuota(ctx, meta.UserId)
	if err != nil {
		logger.Error(ctx, "error update user quota cache: "+err.Error())
	}
	if systemPromptReset {
		extraLog += "注意系统提示词已被重置。"
	}
	var logContent string
	if extraLog != "" {
		logContent = fmt.Sprintf("模型倍率 %.3f，分组倍率 %.3f，补全倍率 %.3f(%s)", modelRatio, groupRatio, completionRatio, extraLog)
	} else {
		logContent = fmt.Sprintf("模型倍率 %.3f，分组倍率 %.3f，补全倍率 %.3f", modelRatio, groupRatio, completionRatio)

	}
	model.RecordConsumeLog(ctx, meta.UserId, meta.ChannelId, promptTokens, cachedTokens, completionTokens, textRequest.Model, meta.TokenName, quota, logContent)
	model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
	model.UpdateChannelUsedQuota(meta.ChannelId, quota)
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

func isErrorHappened(meta *meta.Meta, resp *http.Response) bool {
	if resp == nil {
		if meta.ChannelType == channeltype.AwsClaude {
			return false
		}
		return true
	}
	if resp.StatusCode != http.StatusOK &&
		// replicate return 201 to create a task
		resp.StatusCode != http.StatusCreated {
		return true
	}
	if meta.ChannelType == channeltype.DeepL {
		// skip stream check for deepl
		return false
	}

	//if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") &&
	//	// Even if stream mode is enabled, replicate will first return a task info in JSON format,
	//	// requiring the client to request the stream endpoint in the task info
	//	meta.ChannelType != channeltype.Replicate {
	if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return true
	}
	return false
}

func setSystemPrompt(ctx context.Context, request *relaymodel.GeneralOpenAIRequest, prompt string) (reset bool) {
	if prompt == "" {
		return false
	}
	if len(request.Messages) == 0 {
		return false
	}
	if request.Messages[0].Role == role.System {
		request.Messages[0].Content = prompt
		logger.Infof(ctx, "rewrite system prompt")
		return true
	}
	request.Messages = append([]relaymodel.Message{{
		Role:    role.System,
		Content: prompt,
	}}, request.Messages...)
	logger.Infof(ctx, "add system prompt")
	return true
}
