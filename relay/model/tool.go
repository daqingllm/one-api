package model

type Tool struct {
	Id       string   `json:"id,omitempty"`
	Type     string   `json:"type,omitempty"` // when splicing claude tools stream messages, it is empty
	Function Function `json:"function"`
	Index    *int     `json:"index,omitempty"`
}

type Function struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`       // when splicing claude tools stream messages, it is empty
	Parameters  any    `json:"parameters,omitempty"` // request
	Arguments   any    `json:"arguments,omitempty"`  // response
}

type Annotation struct {
	Type        string      `json:"type"`
	UrlCitation UrlCitation `json:"url_citation"`
}

type UrlCitation struct {
	Url        string `json:"url"`
	Title      string `json:"title"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}
