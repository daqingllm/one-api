package relaymode

const (
	Unknown = iota
	ChatCompletions
	Completions
	Embeddings
	Moderations
	ImagesGenerations
	Edits
	AudioSpeech
	AudioTranscription
	AudioTranslation
	ImagesEdits
	ImageVariations
	Rerank
	// Proxy is a special relay mode for proxying requests to custom upstream
	Proxy
)
