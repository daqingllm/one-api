package jina

import "github.com/songquanpeng/one-api/relay/model"

func ConvertEmbeddingRequest(request model.GeneralOpenAIRequest) *JinaEmbeddingRequest {
	return &JinaEmbeddingRequest{
		Model:         request.Model,
		Dimensions:    request.Dimensions,
		EmbeddingType: request.EncodingFormat,
		Task:          "",
		Input:         request.Input,
	}
}

func ConvertChatRequest(request model.GeneralOpenAIRequest) *JinaDeepSearchRequest {
	return &JinaDeepSearchRequest{
		Model:           request.Model,
		Messages:        request.Messages,
		ReasoningEffort: "medium",
		NoDirectAnswer:  false,
		Stream:          request.Stream,
	}
}
