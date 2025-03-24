// Package aws provides the AWS adaptor for the relay service.
package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/smithy-go"
	"github.com/songquanpeng/one-api/common/config"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/aws/utils"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids.html
var AwsModelIDMap = map[string]string{
	"claude-instant-1.2":         "anthropic.claude-instant-v1",
	"claude-2.0":                 "anthropic.claude-v2",
	"claude-2.1":                 "anthropic.claude-v2:1",
	"claude-3-haiku-20240307":    "anthropic.claude-3-haiku-20240307-v1:0",
	"claude-3-sonnet-20240229":   "anthropic.claude-3-sonnet-20240229-v1:0",
	"claude-3-opus-20240229":     "anthropic.claude-3-opus-20240229-v1:0",
	"claude-3-5-sonnet-20240620": "anthropic.claude-3-5-sonnet-20240620-v1:0",
	"claude-3-5-sonnet-20241022": "anthropic.claude-3-5-sonnet-20241022-v2:0",
	"claude-3-5-sonnet-latest":   "anthropic.claude-3-5-sonnet-20241022-v2:0",
	"claude-3-7-sonnet-20250219": "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
	"claude-3-5-haiku-20241022":  "anthropic.claude-3-5-haiku-20241022-v1:0",
}

func AwsModelID(requestModel string) (string, error) {
	if awsModelID, ok := AwsModelIDMap[requestModel]; ok {
		return awsModelID, nil
	}

	return "", errors.Errorf("model %s not found", requestModel)
}

func Handler(c *gin.Context, awsCli *bedrockruntime.Client, modelName string) (*relaymodel.ErrorWithStatusCode, *relaymodel.Usage) {
	awsModelId, err := AwsModelID(c.GetString(ctxkey.RequestModel))
	if err != nil {
		return utils.WrapErr(errors.Wrap(err, "awsModelID")), nil
	}

	awsReq := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(awsModelId),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}

	claudeReq_, ok := c.Get(ctxkey.ConvertedRequest)
	if !ok {
		return utils.WrapErr(errors.New("request not found")), nil
	}
	claudeReq := claudeReq_.(*anthropic.Request)
	awsClaudeReq := &Request{
		AnthropicVersion: "bedrock-2023-05-31",
	}
	if err = copier.Copy(awsClaudeReq, claudeReq); err != nil {
		return utils.WrapErr(errors.Wrap(err, "copy request")), nil
	}

	awsReq.Body, err = json.Marshal(awsClaudeReq)
	if err != nil {
		return utils.WrapErr(errors.Wrap(err, "marshal request")), nil
	}

	if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
		logger.DebugForcef(c.Request.Context(), "Aws Request: %s", string(awsReq.Body))
	}
	awsResp, err := awsCli.InvokeModel(c.Request.Context(), awsReq)
	if err != nil {
		if opErr, ok := err.(*smithy.OperationError); ok {
			if httpErr, ok := opErr.Err.(*awshttp.ResponseError); ok {
				return &relaymodel.ErrorWithStatusCode{
					IsChannelResponseError: true,
					StatusCode:             httpErr.HTTPStatusCode(),
					Error: relaymodel.Error{
						Message: httpErr.Error(),
					},
				}, nil
			}
		}
		return utils.WrapErr(errors.Wrap(err, "InvokeModel")), nil
	}

	claudeResponse := new(anthropic.Response)
	err = json.Unmarshal(awsResp.Body, claudeResponse)
	if err != nil {
		return utils.WrapErr(errors.Wrap(err, "unmarshal response")), nil
	}

	openaiResp := anthropic.ResponseClaude2OpenAI(claudeResponse)
	openaiResp.Model = modelName
	usage := relaymodel.Usage{
		PromptTokens:     claudeResponse.Usage.InputTokens,
		CompletionTokens: claudeResponse.Usage.OutputTokens,
		TotalTokens:      claudeResponse.Usage.InputTokens + claudeResponse.Usage.OutputTokens,
	}
	openaiResp.Usage = usage

	c.JSON(http.StatusOK, openaiResp)
	return nil, &usage
}

func StreamHandler(c *gin.Context, awsCli *bedrockruntime.Client) (*relaymodel.ErrorWithStatusCode, *relaymodel.Usage) {
	createdTime := helper.GetTimestamp()
	awsModelId, err := AwsModelID(c.GetString(ctxkey.RequestModel))
	if err != nil {
		return utils.WrapErr(errors.Wrap(err, "awsModelID")), nil
	}

	awsReq := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(awsModelId),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}

	claudeReq_, ok := c.Get(ctxkey.ConvertedRequest)
	if !ok {
		return utils.WrapErr(errors.New("request not found")), nil
	}
	claudeReq := claudeReq_.(*anthropic.Request)

	awsClaudeReq := &Request{
		AnthropicVersion: "bedrock-2023-05-31",
	}
	if err = copier.Copy(awsClaudeReq, claudeReq); err != nil {
		return utils.WrapErr(errors.Wrap(err, "copy request")), nil
	}
	awsReq.Body, err = json.Marshal(awsClaudeReq)

	if err != nil {
		return utils.WrapErr(errors.Wrap(err, "marshal request")), nil
	}
	if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
		logger.DebugForcef(c.Request.Context(), "Aws Stream Request: %s", string(awsReq.Body))
	}

	awsResp, err := awsCli.InvokeModelWithResponseStream(c.Request.Context(), awsReq)
	userId := c.GetInt(ctxkey.Id)
	if config.DebugUserIds[userId] {
		logger.DebugForcef(c.Request.Context(), "Aws Stream Request: %s", string(awsReq.Body))
		if awsResp != nil {
			resp, _ := json.Marshal(awsResp)
			logger.DebugForcef(c.Request.Context(), "Aws Stream Response: %s", string(resp))
		}
		if err != nil {
			logger.DebugForcef(c.Request.Context(), "Aws Stream Error: %s", err.Error())
		}
	}
	if err != nil {
		if opErr, ok := err.(*smithy.OperationError); ok {
			if httpErr, ok := opErr.Err.(*awshttp.ResponseError); ok {
				return &relaymodel.ErrorWithStatusCode{
					IsChannelResponseError: true,
					StatusCode:             httpErr.HTTPStatusCode(),
					Error: relaymodel.Error{
						Message: httpErr.Error(),
					},
				}, nil
			}
		}
		return utils.WrapErr(errors.Wrap(err, "InvokeModelWithResponseStream")), nil
	}
	stream := awsResp.GetStream()
	defer stream.Close()

	var usage relaymodel.Usage
	started := new(bool)
	var lastToolCallChoice openai.ChatCompletionsStreamResponseChoice
	toolCounter := &anthropic.ToolCounter{}
	firstEvent, ok := <-stream.Events()
	if !ok {
		logger.Errorf(c.Request.Context(), "stream ended before any response")
		return utils.WrapErr(errors.New("error ocurred in stream")), nil
	}
	c.Stream(func(w io.Writer) bool {
		if !*started {
			common.SetEventStreamHeaders(c)
			a := true
			started = &a
			return streamEventHandler(c, &firstEvent, toolCounter, &lastToolCallChoice, &usage, createdTime)
		}
		event, ok := <-stream.Events()
		if !ok {
			c.Render(-1, common.CustomEvent{Data: "data: [DONE]"})
			return false
		}
		return streamEventHandler(c, &event, toolCounter, &lastToolCallChoice, &usage, createdTime)
	})
	return nil, &usage
}

func streamEventHandler(c *gin.Context, event *types.ResponseStream, toolCounter *anthropic.ToolCounter, lastToolCallChoice *openai.ChatCompletionsStreamResponseChoice, usage *relaymodel.Usage, createdTime int64) bool {
	switch v := (*event).(type) {
	case *types.ResponseStreamMemberChunk:
		var id string
		claudeResp := new(anthropic.StreamResponse)
		err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(claudeResp)
		if err != nil {
			logger.SysError("error unmarshalling stream response: " + err.Error())
			return false
		}

		response, meta := anthropic.StreamResponseClaude2OpenAI(claudeResp, toolCounter)
		if meta != nil {
			usage.PromptTokens += meta.Usage.InputTokens
			usage.CompletionTokens += meta.Usage.OutputTokens
			if len(meta.Id) > 0 { // only message_start has an id, otherwise it's a finish_reason event.
				id = fmt.Sprintf("chatcmpl-%s", meta.Id)
				return true
			} else { // finish_reason case
				if len(lastToolCallChoice.Delta.ToolCalls) > 0 {
					lastArgs := &lastToolCallChoice.Delta.ToolCalls[len(lastToolCallChoice.Delta.ToolCalls)-1].Function
					if len(lastArgs.Arguments.(string)) == 0 { // compatible with OpenAI sending an empty object `{}` when no arguments.
						lastArgs.Arguments = "{}"
						response.Choices[len(response.Choices)-1].Delta.Content = nil
						response.Choices[len(response.Choices)-1].Delta.ToolCalls = lastToolCallChoice.Delta.ToolCalls
					}
				}
			}
		}
		if response == nil {
			return true
		}
		response.Id = id
		response.Model = c.GetString(ctxkey.OriginalModel)
		response.Created = createdTime

		for _, choice := range response.Choices {
			if len(choice.Delta.ToolCalls) > 0 {
				lastToolCallChoice = &choice
			}
		}
		jsonStr, err := json.Marshal(response)
		if err != nil {
			logger.SysError("error marshalling stream response: " + err.Error())
			return true
		}

		c.Render(-1, common.CustomEvent{Data: "data: " + string(jsonStr)})
		c.Writer.Flush()
		return true
	case *types.UnknownUnionMember:
		logger.Errorf(c.Request.Context(), "unknown tag: %s", v.Tag)
		return false
	default:
		logger.Errorf(c.Request.Context(), "union is nil or unknown type")
		return false
	}
}
