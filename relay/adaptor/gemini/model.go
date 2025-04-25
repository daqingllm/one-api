package gemini

type ChatRequest struct {
	Contents         []ChatContent        `json:"contents"`
	SafetySettings   []ChatSafetySettings `json:"safety_settings,omitempty"`
	GenerationConfig ChatGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []ChatTools          `json:"tools,omitempty"`
	ToolConfig       ToolConfig           `json:"toolConfig,omitempty"`
}

type ToolConfig struct {
	FunctionCallingConfig FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type FunctionCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

type EmbeddingRequest struct {
	Model                string      `json:"model"`
	Content              ChatContent `json:"content"`
	TaskType             string      `json:"taskType,omitempty"`
	Title                string      `json:"title,omitempty"`
	OutputDimensionality int         `json:"outputDimensionality,omitempty"`
}

type BatchEmbeddingRequest struct {
	Requests []EmbeddingRequest `json:"requests"`
}

type EmbeddingData struct {
	Values []float64 `json:"values"`
}

type EmbeddingResponse struct {
	Embeddings []EmbeddingData `json:"embeddings"`
	Error      *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type FunctionCall struct {
	Id           string `json:"id,omitempty"`
	FunctionName string `json:"name"`
	Arguments    any    `json:"args,omitempty"`
}

type FileData struct {
	MimeType string `json:"mimeType,omitempty"`
	FileUri  string `json:"fileUri,omitempty"`
}

type FunctionResponse struct {
	Id       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Response any    `json:"response,omitempty"`
}

type Part struct {
	Text             string            `json:"text,omitempty"`
	Thought          *bool             `json:"thought,omitempty"`
	InlineData       *InlineData       `json:"inlineData,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

type ChatContent struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

type ChatSafetySettings struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type Empty struct{}

type Schema struct {
	Type             string             `json:"type"`
	Format           string             `json:"format,omitempty"`
	Title            string             `json:"title,omitempty"`
	Description      string             `json:"description,omitempty"`
	Nullable         bool               `json:"nullable,omitempty"`
	Enum             []string           `json:"enum,omitempty"`
	MaxItems         string             `json:"maxItems,omitempty"`
	MinItems         string             `json:"minItems,omitempty"`
	Properties       map[string]*Schema `json:"properties"` // map[string]any
	Required         []string           `json:"required,omitempty"`
	Items            *Schema            `json:"items,omitempty"` // map[string]any
	Mininum          string             `json:"minimum,omitempty"`
	Maxinum          string             `json:"maximum,omitempty"`
	Anyof            []Schema           `json:"anyOf,omitempty"`
	PropertyOrdering []string           `json:"propertyOrdering,omitempty"`
}
type FunctionDeclaration struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Parameters  *Schema `json:"parameters,omitempty"`
	Response    *Schema `json:"response,omitempty"`
}

type ChatTools struct {
	FunctionDeclarations []FunctionDeclaration `json:"function_declarations,omitempty"`
	GoogleSearch         *Empty                `json:"google_search,omitempty"`
}

type ThinkingConfig struct {
	IncludeThoughts bool `json:"include_thoughts"`
	ThinkingBudget  int  `json:"thinking_budget"`
}

type ChatGenerationConfig struct {
	ResponseMimeType   string          `json:"responseMimeType,omitempty"`
	ResponseSchema     any             `json:"responseSchema,omitempty"`
	Temperature        *float64        `json:"temperature,omitempty"`
	TopP               *float64        `json:"topP,omitempty"`
	TopK               float64         `json:"topK,omitempty"`
	MaxOutputTokens    int             `json:"maxOutputTokens,omitempty"`
	CandidateCount     int             `json:"candidateCount,omitempty"`
	StopSequences      []string        `json:"stopSequences,omitempty"`
	ResponseModalities []string        `json:"response_modalities,omitempty"`
	ThinkingConfig     *ThinkingConfig `json:"thinking_config,omitempty"`
}
