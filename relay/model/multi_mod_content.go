package model

type InlineData struct {
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type MultiModContent struct {
	Text       string     `json:"text,omitempty"`
	InlineData InlineData `json:"inline_data,omitempty"`
}
