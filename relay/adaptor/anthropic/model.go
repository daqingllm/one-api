package anthropic

// https://docs.anthropic.com/claude/reference/messages_post

type Metadata struct {
	UserId string `json:"user_id"`
}

type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ContentReq struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *ImageSource `json:"source,omitempty"`
	// tool_calls
	Id        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	Content   string `json:"content,omitempty"`
	ToolUseId string `json:"tool_use_id,omitempty"`
}

type Content struct {
	Type   string       `json:"type"`
	Text   string       `json:"text"`
	Source *ImageSource `json:"source,omitempty"`
	//citation
	Citations []Citation `json:"citations,omitempty"`
	// thinking
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	// redacted thinking
	Data string `json:"data,omitempty"`
	// tool_calls
	Id    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

type Citation struct {
	Type          string `json:"type,omitempty"`
	CitedText     string `json:"cited_text,omitempty"`
	DocumentIndex int    `json:"document_index,omitempty"`
	DocumentTitle string `json:"document_title,omitempty"`
	// char
	EndCharIndex   int `json:"end_char_index,omitempty"`
	StartCharIndex int `json:"start_char_index,omitempty"`
	//page
	EndePageNumber  int `json:"ende_page_number,omitempty"`
	StartPageNumber int `json:"start_page_number,omitempty"`
	//contentblock
	EndBlockIndex   int `json:"end_block_index,omitempty"`
	StartBlockIndex int `json:"start_block_index,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type Thinking struct {
	BudgetTokens int    `json:"budget_tokens,omitempty"`
	Type         string `json:"type"`
}

type Tool struct {
	Name         string        `json:"name"`
	Type         string        `json:"type,omitempty"`
	Description  string        `json:"description,omitempty"`
	InputSchema  *InputSchema  `json:"input_schema,omitempty"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`

	//computerUseTool
	DisplayHeightPx int `json:"display_height_px,omitempty"`
	DisplayWidthPx  int `json:"display_width_px,omitempty"`
	DisplayNumber   int `json:"display_number,omitempty"`
}

type InputSchema struct {
	Type       string `json:"type"`
	Properties any    `json:"properties,omitempty"`
}

type CacheControl struct {
	Type string `json:"type"`
}

type Request struct {
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	Metadata      *Metadata `json:"metadata,omitempty"`
	System        any       `json:"system,omitempty"`
	MaxTokens     int       `json:"max_tokens,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
	Temperature   *float64  `json:"temperature,omitempty"`
	Thinking      *Thinking `json:"thinking,omitempty"`
	TopP          *float64  `json:"top_p,omitempty"`
	TopK          int       `json:"top_k,omitempty"`
	Tools         []Tool    `json:"tools,omitempty"`
	ToolChoice    any       `json:"tool_choice,omitempty"`
	//Metadata    `json:"metadata,omitempty"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Response struct {
	Id           string    `json:"id"`
	Type         string    `json:"type"`
	Role         string    `json:"role,omitempty"`
	Content      []Content `json:"content,omitempty"`
	Model        string    `json:"model,omitempty"`
	StopReason   *string   `json:"stop_reason,omitempty"`
	StopSequence *string   `json:"stop_sequence,omitempty"`
	Usage        *Usage    `json:"usage,omitempty"`
	Error        *Error    `json:"error,omitempty"`
}

type Delta struct {
	Type         string  `json:"type"`
	Text         string  `json:"text,omitempty"`
	Thinking     string  `json:"thinking,omitempty"`
	Signature    string  `json:"signature,omitempty"`
	PartialJson  string  `json:"partial_json,omitempty"`
	StopReason   *string `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

type StreamResponse struct {
	//todo type = "error" https://docs.anthropic.com/en/api/messages-streaming#error-events
	Type         string    `json:"type"`
	Message      *Response `json:"message,omitempty"`
	Index        *int      `json:"index,omitempty"`
	ContentBlock *Content  `json:"content_block,omitempty"`
	Delta        *Delta    `json:"delta,omitempty"`
	Usage        *Usage    `json:"usage,omitempty"`
}
