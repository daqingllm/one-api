package jina

import "github.com/songquanpeng/one-api/relay/model"

// https://jina.ai/embeddings/
type JinaEmbeddingRequest struct {
	Model         string `json:"model"`
	Dimensions    int    `json:"dimensions,omitempty"`
	EmbeddingType string `json:"embedding_type,omitempty"`
	Task          string `json:"task,omitempty"`
	Input         any    `json:"input"`
}

// https://jina.ai/deepsearch
type JinaDeepSearchRequest struct {
	Model           string          `json:"model"`
	Messages        []model.Message `json:"messages"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	NoDirectAnswer  bool            `json:"no_direct_answer,omitempty"`
	Stream          bool            `json:"stream,omitempty"`
}

type JinaRerankRequest struct {
	Documents       []any  `json:"documents"`
	Query           any    `json:"query"`
	Model           string `json:"model"`
	TopN            int    `json:"top_n,omitempty"`
	ReturnDocuments bool   `json:"return_documents"`
}
