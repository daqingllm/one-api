package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func RelayAudioHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	audioModel := "whisper-1"

	tokenId := c.GetInt(ctxkey.TokenId)
	channelType := c.GetInt(ctxkey.Channel)
	channelId := c.GetInt(ctxkey.ChannelId)
	userId := c.GetInt(ctxkey.Id)
	group := c.GetString(ctxkey.Group)
	tokenName := c.GetString(ctxkey.TokenName)

	var ttsRequest openai.TextToSpeechRequest
	if relayMode == relaymode.AudioSpeech {
		// Read JSON
		err := common.UnmarshalBodyReusable(c, &ttsRequest)
		// Check if JSON is valid
		if err != nil {
			return openai.ErrorWrapper(err, "invalid_json", http.StatusBadRequest)
		}
		audioModel = ttsRequest.Model
		// Check if text is too long 4096
		if len(ttsRequest.Input) > 4096 {
			return openai.ErrorWrapper(errors.New("input is too long (over 4096 characters)"), "text_too_long", http.StatusBadRequest)
		}
	}

	modelRatio := billingratio.GetModelRatio(audioModel, channelType)
	groupRatio := billingratio.GetGroupRatio(group)
	ratio := modelRatio * groupRatio
	var quota int64
	var preConsumedQuota int64
	switch relayMode {
	case relaymode.AudioSpeech:
		preConsumedQuota = int64(float64(len(ttsRequest.Input)) * ratio)
		quota = preConsumedQuota
	default:
		preConsumedQuota = int64(float64(config.PreConsumedQuota) * ratio)
	}
	userQuota, err := model.CacheGetUserQuota(ctx, userId)
	if err != nil {
		return openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}

	// Check if user quota is enough
	if userQuota-preConsumedQuota < 0 {
		return openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	err = model.CacheDecreaseUserQuota(userId, preConsumedQuota)
	if err != nil {
		return openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota > 100*preConsumedQuota {
		// in this case, we do not pre-consume quota
		// because the user has enough quota
		preConsumedQuota = 0
	}
	if preConsumedQuota > 0 {
		err := model.PreConsumeTokenQuota(tokenId, preConsumedQuota)
		if err != nil {
			return openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
	}
	succeed := false
	defer func() {
		if succeed {
			return
		}
		if preConsumedQuota > 0 {
			// we need to roll back the pre-consumed quota
			defer func(ctx context.Context) {
				go func() {
					// negative means add quota back for token & user
					err := model.PostConsumeTokenQuota(tokenId, -preConsumedQuota)
					if err != nil {
						logger.Error(ctx, fmt.Sprintf("error rollback pre-consumed quota: %s", err.Error()))
					}
				}()
			}(c.Request.Context())
		}
	}()

	// map model name
	modelMapping := c.GetString(ctxkey.ModelMapping)
	if modelMapping != "" {
		modelMap := make(map[string]string)
		err := json.Unmarshal([]byte(modelMapping), &modelMap)
		if err != nil {
			return openai.ErrorWrapper(err, "unmarshal_model_mapping_failed", http.StatusInternalServerError)
		}
		if modelMap[audioModel] != "" {
			audioModel = modelMap[audioModel]
		}
	}

	baseURL := channeltype.ChannelBaseURLs[channelType]
	requestURL := c.Request.URL.String()
	if c.GetString(ctxkey.BaseURL) != "" {
		baseURL = c.GetString(ctxkey.BaseURL)
	}

	fullRequestURL := openai.GetFullRequestURL(baseURL, requestURL, channelType)
	if channelType == channeltype.Azure {
		apiVersion := meta.Config.APIVersion
		if relayMode == relaymode.AudioTranscription {
			// https://learn.microsoft.com/en-us/azure/ai-services/openai/whisper-quickstart?tabs=command-line#rest-api
			fullRequestURL = fmt.Sprintf("%s/openai/deployments/%s/audio/transcriptions?api-version=%s", baseURL, audioModel, apiVersion)
		} else if relayMode == relaymode.AudioSpeech {
			// https://learn.microsoft.com/en-us/azure/ai-services/openai/text-to-speech-quickstart?tabs=command-line#rest-api
			fullRequestURL = fmt.Sprintf("%s/openai/deployments/%s/audio/speech?api-version=%s", baseURL, audioModel, apiVersion)
		}
	}

	requestBody := &bytes.Buffer{}
	var responseFormat string
	var rewriteFormat string
	writer := multipart.NewWriter(requestBody)

	if relayMode == relaymode.AudioSpeech {
		_, err = io.Copy(requestBody, c.Request.Body)
		if err != nil {
			return openai.ErrorWrapper(err, "new_request_body_failed", http.StatusInternalServerError)
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody.Bytes()))
	} else {
		responseFormat = c.DefaultPostForm("response_format", "json")
		rewriteFormat = transFormat(responseFormat)
		// 遍历所有表单字段
		var hasFormat bool
		for key, values := range c.Request.MultipartForm.Value {
			if key == "response_format" {
				hasFormat = true
				_ = writer.WriteField(key, rewriteFormat)
				continue
			}
			for _, value := range values {
				_ = writer.WriteField(key, value)
			}
		}
		if !hasFormat {
			_ = writer.WriteField("response_format", rewriteFormat)
		}
		// 遍历所有文件字段
		for key, files := range c.Request.MultipartForm.File {
			for _, fileHeader := range files {
				// 打开文件
				file, err := fileHeader.Open()
				if err != nil {
					return openai.ErrorWrapper(err, "open_file_failed", http.StatusInternalServerError)
				}
				defer file.Close()

				// 将文件内容写入到新的 multipart 请求
				fileWriter, err := writer.CreateFormFile(key, fileHeader.Filename)
				if err != nil {
					return openai.ErrorWrapper(err, "create_form_file_failed", http.StatusInternalServerError)
				}
				if _, err := io.Copy(fileWriter, file); err != nil {
					return openai.ErrorWrapper(err, "copy_file_failed", http.StatusInternalServerError)
				}
			}
		}

		// 关闭 writer，写入结束边界
		if err := writer.Close(); err != nil {
			return openai.ErrorWrapper(err, "close_writer_failed", http.StatusInternalServerError)
		}
	}

	req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}

	if (relayMode == relaymode.AudioTranscription || relayMode == relaymode.AudioSpeech) && channelType == channeltype.Azure {
		// https://learn.microsoft.com/en-us/azure/ai-services/openai/whisper-quickstart?tabs=command-line#rest-api
		apiKey := c.Request.Header.Get("Authorization")
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		req.Header.Set("api-key", apiKey)
		req.ContentLength = c.Request.ContentLength
	} else {
		req.Header.Set("Authorization", c.Request.Header.Get("Authorization"))
	}
	if relayMode == relaymode.AudioSpeech {
		req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	} else {
		req.Header.Set("Content-Type", writer.FormDataContentType())
	}
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}

	err = req.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_request_body_failed", http.StatusInternalServerError)
	}
	err = c.Request.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_request_body_failed", http.StatusInternalServerError)
	}

	if relayMode != relaymode.AudioSpeech {
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		}
		err = resp.Body.Close()
		if err != nil {
			return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
		}

		var openAIErr openai.SlimTextResponse
		if err = json.Unmarshal(responseBody, &openAIErr); err == nil {
			if openAIErr.Error.Message != "" {
				return openai.ErrorWrapper(fmt.Errorf("type %s, code %v, message %s", openAIErr.Error.Type, openAIErr.Error.Code, openAIErr.Error.Message), "request_error", http.StatusInternalServerError)
			}
		}
		var seconds int64
		var finalResp []byte
		switch rewriteFormat {
		case "srt":
			seconds, finalResp = getSecFromSRT(responseBody)
		case "vtt":
			seconds, finalResp = getSecFromVTT(responseBody)
		case "verbose_json":
			seconds, finalResp, err = getSecFromVerboseJson(responseBody, responseFormat)
			if err != nil {
				return openai.ErrorWrapper(err, "get_sec_from_verbose_json_failed", http.StatusInternalServerError)
			}
		default:
			return openai.ErrorWrapper(errors.New("unexpected_response_format"), "unexpected_response_format", http.StatusInternalServerError)
		}
		quota = int64(float64(seconds) * ratio)
		resp.Body = io.NopCloser(bytes.NewBuffer(finalResp))
	}
	if resp.StatusCode != http.StatusOK {
		return RelayErrorHandler(resp)
	}
	succeed = true
	quotaDelta := quota - preConsumedQuota
	defer func(ctx context.Context) {
		go billing.PostConsumeQuota(ctx, tokenId, quotaDelta, quota, userId, channelId, modelRatio, groupRatio, audioModel, tokenName)
	}(c.Request.Context())

	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
	}
	return nil
}

func transFormat(format string) string {
	if format == "" || format == "json" || format == "text" {
		return "verbose_json"
	} else {
		return format
	}
}

func getSecFromVerboseJson(body []byte, format string) (int64, []byte, error) {
	var whisperResponse openai.WhisperVerboseJSONResponse
	if err := json.Unmarshal(body, &whisperResponse); err != nil {
		return 0, nil, fmt.Errorf("unmarshal_response_body_failed err :%w", err)
	}
	if format == "verbose_json" {
		return int64(whisperResponse.Duration) + 1, body, nil
	}
	if format == "json" {
		strResp := fmt.Sprintf(
			`{
  "text":"%s"
}`, whisperResponse.Text)
		return int64(whisperResponse.Duration) + 1, []byte(strResp), nil
	}
	if format == "text" {
		return int64(whisperResponse.Duration) + 1, []byte(whisperResponse.Text), nil
	}
	return 0, nil, errors.New("unexpected_response_format")
}

func getSecFromVTT(body []byte) (int64, []byte) {
	return calculateTotalDuration(string(body), "vtt"), body
}

func getSecFromSRT(body []byte) (int64, []byte) {
	return calculateTotalDuration(string(body), "srt"), body
}

func calculateTotalDuration(subtitle string, format string) int64 {
	// 用正则表达式匹配时间戳
	var re *regexp.Regexp
	if format == "vtt" {
		re = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d{3}) --> (\d{2}:\d{2}:\d{2}\.\d{3})`)
	} else {
		re = regexp.MustCompile(`(\d{2}:\d{2}:\d{2},\d{3}) --> (\d{2}:\d{2}:\d{2},\d{3})`)
	}
	matches := re.FindAllStringSubmatch(subtitle, -1)

	// 迭代匹配的时间戳
	var startTime time.Time
	var endTime time.Time
	for i, match := range matches {
		if i == 0 {
			startTime = parseTime(match[1])
		}
		endTime = parseTime(match[2])
	}
	duration := endTime.Sub(startTime)

	// 将总持续时间转换为秒
	return int64(duration.Seconds() + 1)
}

func parseTime(timeStr string) time.Time {
	// 解析时间字符串为 time.Time
	t, err := time.Parse("15:04:05.000", timeStr)
	if err != nil {
		panic(err)
	}
	return t
}
