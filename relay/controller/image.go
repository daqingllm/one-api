package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func getImageRequest(c *gin.Context, _ int) (*relaymodel.ImageRequest, error) {
	imageRequest := &relaymodel.ImageRequest{}
	err := common.UnmarshalBodyReusable(c, imageRequest)
	if err != nil {
		return nil, err
	}
	if imageRequest.N == 0 {
		imageRequest.N = 1
	}
	if imageRequest.Size == "" {
		imageRequest.Size = "1024x1024"
	}
	if imageRequest.Model == "" {
		imageRequest.Model = "gpt-image-1"
	}
	if imageRequest.Quality == "" {
		if strings.HasPrefix(imageRequest.Model, "dall-e") {
			imageRequest.Quality = "standard"
		} else if strings.HasPrefix(imageRequest.Model, "gpt-image") {
			imageRequest.Quality = "auto"
		}
	}
	return imageRequest, nil
}

func isValidImageSize(model string, size string) bool {
	if model == "cogview-3" || billingratio.ImageSizeRatios[model] == nil {
		return true
	}
	_, ok := billingratio.ImageSizeRatios[model][size]
	return ok
}

func isValidImagePromptLength(model string, promptLength int) bool {
	maxPromptLength, ok := billingratio.ImagePromptLengthLimitations[model]
	return !ok || promptLength <= maxPromptLength
}

func isWithinRange(element string, value int) bool {
	amounts, ok := billingratio.ImageGenerationAmounts[element]
	return !ok || (value >= amounts[0] && value <= amounts[1])
}

func getImageSizeRatio(model string, size string) float64 {
	if ratio, ok := billingratio.ImageSizeRatios[model][size]; ok {
		return ratio
	}
	return 1
}

func getImageQualityRatio(model string, quality string) float64 {
	if ratio, ok := billingratio.ImageQualityRatios[model][quality]; ok {
		return ratio
	}
	return 1
}

func validateImageRequest(imageRequest *relaymodel.ImageRequest, _ *meta.Meta, relayMode int) *relaymodel.ErrorWithStatusCode {
	// check prompt length
	if imageRequest.Prompt == "" && (relayMode == relaymode.ImagesEdits || relayMode == relaymode.ImagesGenerations) {
		return openai.ErrorWrapper(errors.New("prompt is required"), "prompt_missing", http.StatusBadRequest)
	}

	// model validation
	if !isValidImageSize(imageRequest.Model, imageRequest.Size) {
		return openai.ErrorWrapper(errors.New("size not supported for this image model"), "size_not_supported", http.StatusBadRequest)
	}

	if !isValidImagePromptLength(imageRequest.Model, len(imageRequest.Prompt)) {
		return openai.ErrorWrapper(errors.New("prompt is too long"), "prompt_too_long", http.StatusBadRequest)
	}

	// Number of generated images validation
	if !isWithinRange(imageRequest.Model, imageRequest.N) {
		return openai.ErrorWrapper(errors.New("invalid value of n"), "n_not_within_range", http.StatusBadRequest)
	}
	return nil
}

func getImageCostRatio(model string, size string, quality string) float64 {
	imageSizeCostRatio := getImageSizeRatio(model, size)
	imageQualityRatio := getImageQualityRatio(model, quality)
	if quality == "hd" && model == "dall-e-3" {
		if size == "1024x1024" {
			imageQualityRatio = 2
		} else {
			imageQualityRatio = 1.5
		}
	}
	return imageSizeCostRatio * imageQualityRatio
}

func RelayImageHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	imageRequest, err := getImageRequest(c, meta.Mode)
	if err != nil {
		logger.Errorf(ctx, "getImageRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_image_request", http.StatusBadRequest)
	}

	// map model name
	var isModelMapped bool
	meta.OriginModelName = imageRequest.Model
	imageRequest.Model, isModelMapped = getMappedModelName(imageRequest.Model, meta.ModelMapping)
	meta.ActualModelName = imageRequest.Model

	// model validation
	bizErr := validateImageRequest(imageRequest, meta, relayMode)
	if bizErr != nil {
		return bizErr
	}

	imageCostRatio := getImageCostRatio(imageRequest.Model, imageRequest.Size, imageRequest.Quality)

	imageModel := imageRequest.Model
	modelRatio := billingratio.GetModelRatio(imageModel, meta.ChannelType)
	groupRatio := billingratio.GetGroupRatio(meta.Group)
	ratio := modelRatio * groupRatio
	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)

	//var quota int64
	//switch meta.ChannelType {
	//case channeltype.Replicate:
	//	// replicate always return 1 image
	//	quota = int64(ratio * imageCostRatio * 1000)
	//default:
	//	quota = int64(ratio*imageCostRatio*1000) * int64(imageRequest.N)
	//}
	quota := int64(ratio*imageCostRatio*1000) * int64(imageRequest.N)

	if userQuota-quota < 0 {
		return openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}

	// Convert the original image model
	imageRequest.Model, _ = getMappedModelName(imageRequest.Model, billingratio.ImageOriginModelName)
	c.Set("response_format", imageRequest.ResponseFormat)

	var requestBody io.Reader
	if strings.ToLower(c.GetString(ctxkey.ContentType)) == "application/json" &&
		(isModelMapped || meta.ChannelType == channeltype.Azure) { // make Azure channel request body
		jsonStr, err := json.Marshal(imageRequest)
		if err != nil {
			return openai.ErrorWrapper(err, "marshal_image_request_failed", http.StatusInternalServerError)
		}
		requestBody = bytes.NewBuffer(jsonStr)
		if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
			logger.DebugForcef(ctx, "Azure channel: channel id %d, user id %d, request body: %s", c.GetInt(ctxkey.ChannelId), c.GetInt(ctxkey.Id), string(jsonStr))
		}
	} else {
		requestBody = c.Request.Body
		if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
			requestBody, _ := common.GetRequestBody(c)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			logger.DebugForcef(ctx, "Openai channel: channel id %d, user id %d, request body: %s", c.GetInt(ctxkey.ChannelId), c.GetInt(ctxkey.Id), string(requestBody))
		}
	}

	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// these adaptors need to convert the request
	switch meta.ChannelType {
	case channeltype.Zhipu,
		channeltype.Ali,
		//channeltype.Replicate,
		channeltype.Baidu:
		finalRequest, err := adaptor.ConvertImageRequest(imageRequest)
		if err != nil {
			return openai.ErrorWrapper(err, "convert_image_request_failed", http.StatusInternalServerError)
		}
		jsonStr, err := json.Marshal(finalRequest)
		if err != nil {
			return openai.ErrorWrapper(err, "marshal_image_request_failed", http.StatusInternalServerError)
		}
		requestBody = bytes.NewBuffer(jsonStr)
	}

	// do request
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ChannelErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(meta, resp) {
		return RelayErrorHandler(resp)
	}

	// do response
	_, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		return respErr
	}

	defer func(ctx context.Context) {
		if resp != nil &&
			resp.StatusCode != http.StatusCreated && // replicate returns 201
			resp.StatusCode != http.StatusOK {
			return
		}
		usage := openai.GetImageUsageIfPossible(resp)
		if usage != nil {
			//gpt-image price
			completionRatio := billingratio.GetCompletionRatio(imageModel, meta.ChannelType)
			quota = int64(math.Ceil(ratio*
				float64(usage.InputTokensDetails.TextTokens) + // text input
				float64(usage.InputTokensDetails.ImageTokens*2) + // image input
				float64(usage.OutputTokens)*completionRatio)) // image output
		}

		err := model.PostConsumeTokenQuota(meta.TokenId, quota)
		if err != nil {
			logger.SysError("error consuming token remain quota: " + err.Error())
		}
		err = model.CacheUpdateUserQuota(ctx, meta.UserId)
		if err != nil {
			logger.SysError("error update user quota cache: " + err.Error())
		}
		if quota != 0 {
			tokenName := c.GetString(ctxkey.TokenName)
			logContent := fmt.Sprintf("模型倍率 %.3f，分组倍率 %.3f", modelRatio, groupRatio)
			if usage != nil {
				logContent += fmt.Sprintf("，图片生成倍率 8.0，图片生成消耗 %d 个 token", usage.InputTokensDetails.TextTokens+usage.InputTokensDetails.ImageTokens*2+usage.OutputTokens*8)
				model.RecordConsumeLog(ctx, meta.UserId, meta.ChannelId, usage.InputTokensDetails.TextTokens+usage.InputTokensDetails.ImageTokens*2, 0, usage.OutputTokens, imageRequest.Model, tokenName, quota, logContent)
			}
			model.RecordConsumeLog(ctx, meta.UserId, meta.ChannelId, 0, 0, 0, imageRequest.Model, tokenName, quota, logContent)
			model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
			channelId := c.GetInt(ctxkey.ChannelId)
			model.UpdateChannelUsedQuota(channelId, quota)
		}
	}(c.Request.Context())

	return nil
}
