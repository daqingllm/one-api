package mistral

import (
	"fmt"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.Mode {
	case relaymode.ChatCompletions:
		return fmt.Sprintf("%s/v1/fim/completions", meta.BaseURL), nil
	default:
	}
	return "", fmt.Errorf("unsupported relay mode %d for mistral", meta.Mode)
}
