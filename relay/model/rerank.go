package model

type RerankRequest struct {
	Documents      []any  `json:"documents"`
	Query          any    `json:"query"`
	Model          string `json:"model"`
	TopN           int    `json:"top_n"`
	MaxChunkPerDoc int    `json:"max_chunk_per_doc,omitempty"`
}

type RerankResponseDocument struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type RerankUsage struct {
	TotalTokens *int `json:"total_tokens,omitempty"`
	SearchUnits *int `json:"search_units,omitempty"`
}

type RerankResponse struct {
	Model   string                   `json:"model"`
	Results []RerankResponseDocument `json:"results"`
	Usage   RerankUsage              `json:"usage"`
}
