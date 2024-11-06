package claude_adaptor

import (
	"bytes"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/render"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	claude "github.com/songquanpeng/one-api/relay/adaptor/aws/claude"
	"github.com/songquanpeng/one-api/relay/adaptor/aws/utils"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
)

var _ Adaptor = new(Aws)

type Aws struct{}

func (a *Aws) DoRequest(c *gin.Context, request *anthropic.Request, meta *meta.Meta) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	awsClient := bedrockruntime.New(bedrockruntime.Options{
		Region:      meta.Config.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(meta.Config.AK, meta.Config.SK, "")),
	})
	if meta.IsStream {
		return AwsStreamHandler(c, request, awsClient)
	} else {
		return AwsHandler(c, request, awsClient)
	}
}

func AwsHandler(c *gin.Context, request *anthropic.Request, client *bedrockruntime.Client) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	awsModelId, err := claude.AwsModelID(request.Model)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "awsModelID"))
	}

	awsReq := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(awsModelId),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}

	awsClaudeReq := &claude.Request{
		AnthropicVersion: "bedrock-2023-05-31",
	}
	if err = copier.Copy(awsClaudeReq, request); err != nil {
		logger.Errorf(ctx, "copy request error: %v", err)
		return nil, utils.WrapErr(errors.Wrap(err, "copy request"))
	}

	awsReq.Body, err = json.Marshal(awsClaudeReq)
	if err != nil {
		logger.Errorf(ctx, "marshal request error: %v", err)
		return nil, utils.WrapErr(errors.Wrap(err, "marshal request"))
	}

	awsResp, err := client.InvokeModel(c.Request.Context(), awsReq)
	if err != nil {
		logger.Errorf(ctx, "invoke model error: %v", err)
		return nil, controller.RelayErrorHandler(nil)
	}

	claudeResponse := new(anthropic.Response)
	err = json.Unmarshal(awsResp.Body, claudeResponse)
	if err != nil {
		logger.Errorf(ctx, "unmarshal response error: %v", err)
		return nil, utils.WrapErr(errors.Wrap(err, "unmarshal response"))
	}
	if claudeResponse.Error != nil && claudeResponse.Error.Type != "" {
		return nil, &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: claudeResponse.Error.Message,
				Type:    claudeResponse.Error.Type,
				Param:   "",
				Code:    claudeResponse.Error.Type,
			},
			StatusCode: 400,
		}
	}
	c.JSON(http.StatusOK, claudeResponse)
	return claudeResponse.Usage, nil
}

func AwsStreamHandler(c *gin.Context, request *anthropic.Request, client *bedrockruntime.Client) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	awsModelId, err := claude.AwsModelID(request.Model)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "awsModelID"))
	}

	awsReq := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(awsModelId),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}

	awsClaudeReq := &claude.Request{
		AnthropicVersion: "bedrock-2023-05-31",
	}
	if err = copier.Copy(awsClaudeReq, request); err != nil {
		logger.Errorf(ctx, "copy request error: %v", err)
		return nil, utils.WrapErr(errors.Wrap(err, "copy request"))
	}
	awsReq.Body, err = json.Marshal(awsClaudeReq)
	if err != nil {
		logger.Errorf(ctx, "marshal request error: %v", err)
		return nil, utils.WrapErr(errors.Wrap(err, "marshal request"))
	}

	awsResp, err := client.InvokeModelWithResponseStream(c.Request.Context(), awsReq)
	if err != nil {
		logger.Errorf(ctx, "invoke model error: %v", err)
		return nil, controller.RelayErrorHandler(nil)
	}
	stream := awsResp.GetStream()
	defer func(stream *bedrockruntime.InvokeModelWithResponseStreamEventStream) {
		err := stream.Close()
		if err != nil {
			logger.Errorf(ctx, "close stream error: %v", err)
		}
	}(stream)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	var inputTokens int
	var outputTokens int

	c.Stream(func(w io.Writer) bool {
		event, ok := <-stream.Events()
		if !ok {
			return false
		}

		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			claudeResp := new(anthropic.StreamResponse)
			err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(claudeResp)
			if err != nil {
				logger.SysError("error unmarshalling stream response: " + err.Error())
				return false
			}
			jsonStr, err := json.Marshal(claudeResp)
			if err != nil {
				logger.SysError("error marshalling stream response: " + err.Error())
				return true
			}
			eventType := claudeResp.Type
			render.RawData(c, "event: "+eventType)
			render.RawData(c, "data: "+string(jsonStr))
			render.RawData(c, "")

			if claudeResp.Message != nil {
				inputTokens += claudeResp.Message.Usage.InputTokens
				outputTokens += claudeResp.Message.Usage.OutputTokens
			}
			if claudeResp.Usage != nil {
				inputTokens += claudeResp.Usage.InputTokens
				outputTokens += claudeResp.Usage.OutputTokens
			}
			return true
		case *types.UnknownUnionMember:
			logger.Errorf(ctx, "unknown tag:"+v.Tag)
			return false
		default:
			logger.Errorf(ctx, "union is nil or unknown type")
			return false
		}
	})

	return &anthropic.Usage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}
