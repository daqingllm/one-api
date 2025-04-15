package gemini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/render"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/image"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/constant"
	"github.com/songquanpeng/one-api/relay/model"

	"github.com/gin-gonic/gin"
)

// https://ai.google.dev/docs/gemini_api_overview?hl=zh-cn

const (
	VisionMaxImageNum = 16
)

var mimeTypeMap = map[string]string{
	"json_object": "application/json",
	"text":        "text/plain",
}

// Setting safety to the lowest possible values since Gemini is already powerless enough
func ConvertRequest(textRequest model.GeneralOpenAIRequest) *ChatRequest {
	geminiRequest := ChatRequest{
		Contents: make([]ChatContent, 0, len(textRequest.Messages)),
		SafetySettings: []ChatSafetySettings{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: config.GeminiSafetySetting,
			},
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: config.GeminiSafetySetting,
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: config.GeminiSafetySetting,
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: config.GeminiSafetySetting,
			},
			{
				Category:  "HARM_CATEGORY_CIVIC_INTEGRITY",
				Threshold: config.GeminiSafetySetting,
			},
		},
		GenerationConfig: ChatGenerationConfig{
			Temperature:     textRequest.Temperature,
			TopP:            textRequest.TopP,
			MaxOutputTokens: textRequest.MaxTokens,
		},
	}
	if textRequest.ResponseFormat != nil {
		if mimeType, ok := mimeTypeMap[textRequest.ResponseFormat.Type]; ok {
			geminiRequest.GenerationConfig.ResponseMimeType = mimeType
		}
		if textRequest.ResponseFormat.JsonSchema != nil {
			geminiRequest.GenerationConfig.ResponseSchema = textRequest.ResponseFormat.JsonSchema.Schema
			geminiRequest.GenerationConfig.ResponseMimeType = mimeTypeMap["json_object"]
		}
	}
	if textRequest.Modalities != nil {
		geminiRequest.GenerationConfig.ResponseModalities = textRequest.Modalities

	}

	if textRequest.Tools != nil {
		functions := make([]FunctionDeclaration, 0, len(textRequest.Tools))
		for _, tool := range textRequest.Tools {
			jsonBytes, err := json.Marshal(tool.Function)
			if err != nil {
				logger.SysError("Failed to marshal tool function: " + err.Error())
				continue
			}
			var functionDeclaration = FunctionDeclaration{}
			err = json.Unmarshal(jsonBytes, &functionDeclaration)
			if err != nil {
				logger.SysError("Failed to unmarshal tool function: " + err.Error())
				continue

			}
			// if tool.Function.Parameters != nil && common.IsEmptyObject(tool.Function.Parameters) {
			// 	tool.Function.Parameters = nil
			// } else if tool.Function.Parameters != nil {
			// 	// 兼容gemini不能传空对象
			// 	if params, ok := tool.Function.Parameters.(map[string]any); ok {
			// 		if params["type"] == "object" && common.IsEmptyObject(params["properties"]) {
			// 			tool.Function.Parameters = nil
			// 		}
			// 	}

			// }
			functions = append(functions, functionDeclaration)
		}
		geminiRequest.Tools = []ChatTools{
			{
				FunctionDeclarations: functions,
			},
		}
	} else if textRequest.Functions != nil {
		jsonBytes, err := json.Marshal(textRequest.Functions)
		if err != nil {
			logger.SysError("Failed to marshal tool function: " + err.Error())
		}
		var functionDeclarations = []FunctionDeclaration{}
		err = json.Unmarshal(jsonBytes, &functionDeclarations)
		if err != nil {
			logger.SysError("Failed to unmarshal tool function: " + err.Error())
		}
		geminiRequest.Tools = []ChatTools{
			{
				FunctionDeclarations: functionDeclarations,
			},
		}
	}
	if toolChoice := textRequest.ToolChoice; toolChoice != nil {
		toolConfig := ToolConfig{
			FunctionCallingConfig: FunctionCallingConfig{
				Mode: "auto", // default mode
			},
		}
		if str, ok := toolChoice.(string); ok {
			switch str {
			case "required":
				toolConfig.FunctionCallingConfig.Mode = "any"
				toolConfig.FunctionCallingConfig.AllowedFunctionNames = getAllowedFunctionNames(&geminiRequest)
			case "auto":
			}
		} else if m, ok := toolChoice.(map[string]interface{}); ok {
			if funcMap, ok := m["function"].(map[string]interface{}); ok {
				if funcName, ok := funcMap["name"].(string); ok {
					toolConfig.FunctionCallingConfig.Mode = "any"
					toolConfig.FunctionCallingConfig.AllowedFunctionNames = []string{funcName}
				}
			}
		}
		geminiRequest.ToolConfig = toolConfig
	}
	shouldAddDummyModelMessage := false
	toolCallIdMap := make(map[string]string)
	for _, message := range textRequest.Messages {
		content := ChatContent{}
		switch message.GetMessageType() {

		case model.ToolMessage:
			content.Role = "user"
			responseMap := make(map[string]interface{}, 1)
			responseMap["content"] = message.Content
			content.Parts = append(content.Parts, Part{
				Text: "",
				FunctionResponse: &FunctionResponse{
					Id:       message.ToolCallId,
					Name:     toolCallIdMap[message.ToolCallId],
					Response: responseMap,
				},
			})
			geminiRequest.Contents = append(geminiRequest.Contents, content)
		case model.ToolCallMessage:
			content.Role = "model"
			for _, toolCall := range message.ToolCalls {
				toolCallIdMap[toolCall.Id] = toolCall.Function.Name
				var argumentsMap map[string]interface{}

				// 与ToolMessage保持一致的解析逻辑
				if argsMap, ok := toolCall.Function.Arguments.(map[string]any); ok {
					argumentsMap = argsMap
				} else {
					argsStr, ok := toolCall.Function.Arguments.(string)
					if !ok {
						logger.SysError("toolCall arguments is not string type")
						argsStr = "{}"
					}
					err := json.Unmarshal([]byte(argsStr), &argumentsMap)
					if err != nil {
						logger.SysError("Failed to unmarshal toolCall arguments: " + err.Error())
						argumentsMap = make(map[string]interface{})
					}
				}
				content.Parts = append(content.Parts, Part{
					Text: "",
					FunctionCall: &FunctionCall{
						Id:           toolCall.Id,
						FunctionName: toolCall.Function.Name,
						Arguments:    argumentsMap,
					},
				})
			}
			geminiRequest.Contents = append(geminiRequest.Contents, content)
		case model.ContentMessage:
			content.Role = "user"
			content.Parts = append(content.Parts, Part{
				Text: message.StringContent(),
			})
			openaiContent := message.ParseContent()
			var parts []Part
			imageNum := 0
			for _, part := range openaiContent {
				if part.Type == model.ContentTypeText {
					parts = append(parts, Part{
						Text: part.Text,
					})
				} else if part.Type == model.ContentTypeImageURL {
					imageNum += 1
					if imageNum > VisionMaxImageNum {
						continue
					}
					mimeType, data, _ := image.GetImageFromUrl(part.ImageURL.Url)
					parts = append(parts, Part{
						InlineData: &InlineData{
							MimeType: mimeType,
							Data:     data,
						},
					})
				}
			}
			content.Parts = parts

			// there's no assistant role in gemini and API shall vomit if Role is not user or model
			if content.Role == "assistant" {
				content.Role = "model"
			}
			// Converting system prompt to prompt from user for the same reason
			if content.Role == "system" {
				content.Role = "user"
				shouldAddDummyModelMessage = true
			}
			geminiRequest.Contents = append(geminiRequest.Contents, content)

			// If a system message is the last message, we need to add a dummy model message to make gemini happy
			if shouldAddDummyModelMessage {
				geminiRequest.Contents = append(geminiRequest.Contents, ChatContent{
					Role: "model",
					Parts: []Part{
						{
							Text: "Okay",
						},
					},
				})
				shouldAddDummyModelMessage = false
			}
		}

	}

	return &geminiRequest
}

func getAllowedFunctionNames(geminiRequest *ChatRequest) []string {
	seen := make(map[string]bool)
	var uniqueNames []string

	for _, tool := range geminiRequest.Tools {
		for _, fnDecl := range tool.FunctionDeclarations {
			if !seen[fnDecl.Name] {
				seen[fnDecl.Name] = true
				uniqueNames = append(uniqueNames, fnDecl.Name)
			}
		}
	}
	return uniqueNames
}

func ConvertEmbeddingRequest(request model.GeneralOpenAIRequest) *BatchEmbeddingRequest {
	inputs := request.ParseInput()
	requests := make([]EmbeddingRequest, len(inputs))
	model := fmt.Sprintf("models/%s", request.Model)

	for i, input := range inputs {
		requests[i] = EmbeddingRequest{
			Model: model,
			Content: ChatContent{
				Parts: []Part{
					{
						Text: input,
					},
				},
			},
		}
	}

	return &BatchEmbeddingRequest{
		Requests: requests,
	}
}

type ChatResponse struct {
	Candidates     []ChatCandidate    `json:"candidates"`
	PromptFeedback ChatPromptFeedback `json:"promptFeedback"`
}

func (g *ChatResponse) GetResponseText() string {
	if g == nil {
		return ""
	}
	if len(g.Candidates) > 0 && len(g.Candidates[0].Content.Parts) > 0 && (g.Candidates[0].Content.Parts[0].Thought == nil || !*g.Candidates[0].Content.Parts[0].Thought) {
		return g.Candidates[0].Content.Parts[0].Text
	}
	return ""
}
func (g *ChatResponse) GetResponseThoughtText() string {
	if g == nil {
		return ""
	}
	if len(g.Candidates) > 0 && len(g.Candidates[0].Content.Parts) > 0 && (g.Candidates[0].Content.Parts[0].Thought != nil && *g.Candidates[0].Content.Parts[0].Thought) {
		return g.Candidates[0].Content.Parts[0].Text
	}
	return ""
}

type ChatCandidate struct {
	Content       ChatContent        `json:"content"`
	FinishReason  string             `json:"finishReason"`
	Index         int64              `json:"index"`
	SafetyRatings []ChatSafetyRating `json:"safetyRatings"`
}

type ChatSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type ChatPromptFeedback struct {
	SafetyRatings []ChatSafetyRating `json:"safetyRatings"`
}

func getToolCalls(candidate *ChatCandidate) []model.Tool {
	var toolCalls []model.Tool

	item := candidate.Content.Parts[0]
	if item.FunctionCall == nil {
		return toolCalls
	}
	argsBytes, err := json.Marshal(item.FunctionCall.Arguments)
	if err != nil {
		logger.FatalLog("getToolCalls failed: " + err.Error())
		return toolCalls
	}
	toolCall := model.Tool{
		Id:   fmt.Sprintf("call_%s", random.GetUUID()),
		Type: "function",
		Function: model.Function{
			Arguments: string(argsBytes),
			Name:      item.FunctionCall.FunctionName,
		},
	}
	toolCalls = append(toolCalls, toolCall)
	return toolCalls
}

func responseGeminiChat2OpenAI(response *ChatResponse) *openai.TextResponse {
	fullTextResponse := openai.TextResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", random.GetUUID()),
		Object:  "chat.completion",
		Created: helper.GetTimestamp(),
		Choices: make([]openai.TextResponseChoice, 0, len(response.Candidates)),
	}
	for i, candidate := range response.Candidates {
		choice := openai.TextResponseChoice{
			Index: i,
			Message: model.Message{
				Role: "assistant",
			},
			FinishReason: constant.StopFinishReason,
		}
		if len(candidate.Content.Parts) == 1 && candidate.Content.Parts[0].Text != "" {
			//如果是纯文本，则直接返回
			choice.Message.Content = candidate.Content.Parts[0].Text
		} else if len(candidate.Content.Parts) > 0 {
			if candidate.Content.Parts[0].FunctionCall != nil {
				choice.Message.ToolCalls = getToolCalls(&candidate)
				choice.FinishReason = constant.ToolCallsFinishReason

			} else {
				var builder strings.Builder
				var thoughtBuilder strings.Builder
				var multiModContents = make([]model.MultiModContent, 0)
				var existMultiContents bool = false
				for _, part := range candidate.Content.Parts {
					if part.Thought != nil && *part.Thought {
						thoughtBuilder.WriteString(part.Text + "\n")
					} else if part.Text != "" {
						multiModContents = append(multiModContents, model.MultiModContent{
							Text: part.Text,
						})
						builder.WriteString(part.Text + "\n")
					} else if part.InlineData != nil {
						multiModContents = append(multiModContents, model.MultiModContent{
							InlineData: model.InlineData{
								Data:     part.InlineData.Data,
								MimeType: part.InlineData.MimeType,
							},
						})
						existMultiContents = true
					}
				}
				if existMultiContents {
					choice.Message.MultiModContents = multiModContents
				}
				// 提取多模态中的文本进行填充
				choice.Message.Content = strings.TrimSpace(builder.String())
				thoughtContent := strings.TrimSpace(thoughtBuilder.String())
				if thoughtContent != "" {
					choice.Message.ReasoningContent = thoughtContent
				}
			}
		} else {
			choice.Message.Content = ""
			choice.FinishReason = candidate.FinishReason
		}
		fullTextResponse.Choices = append(fullTextResponse.Choices, choice)
	}
	return &fullTextResponse
}

func streamResponseGeminiChat2OpenAI(geminiResponse *ChatResponse) *openai.ChatCompletionsStreamResponse {
	var choice openai.ChatCompletionsStreamResponseChoice
	// choice.Delta.Content = geminiResponse.GetResponseText()
	thoughtText := geminiResponse.GetResponseThoughtText()
	if len(geminiResponse.Candidates) == 0 {
		return nil
	}
	firstCandidate := geminiResponse.Candidates[0]
	multiModContents, content, _ := getMultiModOrPlainContents(&firstCandidate)
	if thoughtText != "" {
		choice.Delta.ReasoningContent = thoughtText
	}
	if len(multiModContents) > 0 {
		choice.Delta.MultiModContents = multiModContents
	}
	choice.Delta.Content = content
	var firstPart *Part
	if len(firstCandidate.Content.Parts) > 0 {
		firstPart = &firstCandidate.Content.Parts[0]
	}
	if firstPart != nil && firstPart.FunctionCall != nil {
		choice.Delta.ToolCalls = getToolCalls(&geminiResponse.Candidates[0])
		choice.FinishReason = &constant.ToolCallsFinishReason
	}
	if geminiResponse.Candidates[0].FinishReason == "stop" {
		choice.FinishReason = &constant.StopFinishReason
	}
	//choice.FinishReason = &constant.StopFinishReason
	var response openai.ChatCompletionsStreamResponse
	response.Id = fmt.Sprintf("chatcmpl-%s", random.GetUUID())
	response.Created = helper.GetTimestamp()
	response.Object = "chat.completion.chunk"
	response.Model = "gemini"
	response.Choices = []openai.ChatCompletionsStreamResponseChoice{choice}
	return &response
}

func getMultiModOrPlainContents(candidate *ChatCandidate) ([]model.MultiModContent, string, error) {
	var contentBuilder = strings.Builder{}
	var multiModContents = make([]model.MultiModContent, 0)
	var existMultiContents bool = false
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			contentBuilder.WriteString(part.Text)
			multiModContents = append(multiModContents, model.MultiModContent{
				Text: part.Text,
			})
		} else if part.InlineData != nil {
			existMultiContents = true
			multiModContents = append(multiModContents, model.MultiModContent{
				InlineData: model.InlineData{
					Data:     part.InlineData.Data,
					MimeType: part.InlineData.MimeType,
				},
			})
		}
	}
	if existMultiContents {
		return multiModContents, contentBuilder.String(), nil
	}
	return multiModContents[:0], contentBuilder.String(), nil
}
func embeddingResponseGemini2OpenAI(response *EmbeddingResponse) *openai.EmbeddingResponse {
	openAIEmbeddingResponse := openai.EmbeddingResponse{
		Object: "list",
		Data:   make([]openai.EmbeddingResponseItem, 0, len(response.Embeddings)),
		Model:  "gemini-embedding",
		Usage:  model.Usage{TotalTokens: 0},
	}
	for _, item := range response.Embeddings {
		openAIEmbeddingResponse.Data = append(openAIEmbeddingResponse.Data, openai.EmbeddingResponseItem{
			Object:    `embedding`,
			Index:     0,
			Embedding: item.Values,
		})
	}
	return &openAIEmbeddingResponse
}

func StreamHandler(c *gin.Context, resp *http.Response, modelName string) (*model.ErrorWithStatusCode, string) {
	responseText := ""
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)

	common.SetEventStreamHeaders(c)

	for scanner.Scan() {
		data := scanner.Text()
		data = strings.TrimSpace(data)
		if !strings.HasPrefix(data, "data: ") {
			continue
		}
		data = strings.TrimPrefix(data, "data: ")
		data = strings.TrimSuffix(data, "\"")

		var geminiResponse ChatResponse
		err := json.Unmarshal([]byte(data), &geminiResponse)
		if err != nil {
			logger.SysError("error unmarshalling stream response: " + err.Error())
			continue
		}

		response := streamResponseGeminiChat2OpenAI(&geminiResponse)
		if response == nil || len(response.Choices) == 0 {
			continue
		}
		if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
			responseText, _ := json.Marshal(response)
			logger.DebugForcef(c.Request.Context(), "gemini Stream Response: %s userId: %d", string(responseText), c.GetInt(ctxkey.Id))
		}
		responseText += response.Choices[0].Delta.StringContent() + response.Choices[0].Delta.StringReasoningContent()

		err = render.ObjectData(c, response)
		if err != nil {
			logger.SysError(err.Error())
		}
	}

	if err := scanner.Err(); err != nil {
		logger.SysError("error reading stream: " + err.Error())
	}

	render.Done(c)

	err := resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), ""
	}

	return nil, responseText
}

func Handler(c *gin.Context, resp *http.Response, promptTokens int, modelName string) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	var geminiResponse ChatResponse
	err = json.Unmarshal(responseBody, &geminiResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	if len(geminiResponse.Candidates) == 0 {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: "No candidates returned",
				Type:    "server_error",
				Param:   "",
				Code:    500,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}
	fullTextResponse := responseGeminiChat2OpenAI(&geminiResponse)
	fullTextResponse.Model = modelName
	completionTokens := openai.CountTokenText(geminiResponse.GetResponseText()+geminiResponse.GetResponseThoughtText(), modelName)
	usage := model.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
	fullTextResponse.Usage = usage
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	uid := c.GetInt(ctxkey.Id)
	if config.DebugUserIds[uid] {
		logger.DebugForcef(c, "gemini response: %s userId: %d", string(jsonResponse), uid)

	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "write_response_body_failed", http.StatusRequestTimeout), nil
	}
	return nil, &usage
}

func EmbeddingHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	var geminiEmbeddingResponse EmbeddingResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	err = json.Unmarshal(responseBody, &geminiEmbeddingResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	if geminiEmbeddingResponse.Error != nil {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: geminiEmbeddingResponse.Error.Message,
				Type:    "gemini_error",
				Param:   "",
				Code:    geminiEmbeddingResponse.Error.Code,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}
	fullTextResponse := embeddingResponseGemini2OpenAI(&geminiEmbeddingResponse)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "write_response_body_failed", http.StatusRequestTimeout), nil
	}
	return nil, &fullTextResponse.Usage
}
