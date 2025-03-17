package model

type InlineData struct {
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type MultiModContent struct {
	Text       string     `json:"text,omitempty"`
	InlineData InlineData `json:"inlineData,omitempty"`
}
