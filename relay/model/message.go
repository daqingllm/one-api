package model

import "strings"

type Message struct {
	Role             string            `json:"role,omitempty"`
	Content          any               `json:"content,omitempty"`
	ReasoningContent any               `json:"reasoning_content,omitempty"`
	MultiModContents []MultiModContent `json:"multi_mod_content,omitempty"`
	Refusal          *string           `json:"refusal,omitempty"`
	Name             *string           `json:"name,omitempty"`
	FunctionCall     *Function         `json:"function_call,omitempty"`
	ToolCalls        []Tool            `json:"tool_calls,omitempty"`
	ToolCallId       string            `json:"tool_call_id,omitempty"`
	Audio            any               `json:"audio,omitempty"`
	Annotations      []Annotation      `json:"annotations,omitempty"`
}

type MessageType int

const (
	ContentMessage MessageType = iota
	ToolMessage
	ToolCallMessage
	ReasoningContentMessage
)

func (m Message) GetMessageType() MessageType {
	if m.IsToolCallMessage() {
		return ToolCallMessage
	}
	if m.IsToolMessage() {
		return ToolMessage
	}
	if m.IsContentMessage() {
		return ContentMessage
	}
	if m.IsReasoningContentMessage() {
		return ReasoningContentMessage
	}
	return ContentMessage
}

func (m Message) IsReasoningContentMessage() bool {
	reasoning_content, ok := m.ReasoningContent.(string)
	if ok && reasoning_content != "" {
		return true
	}
	reasoningList, ok := m.ReasoningContent.([]any)
	if ok && len(reasoningList) > 0 {
		return true
	}
	return false
}
func (m Message) IsStringContent() bool {
	_, ok := m.Content.(string)
	return ok
}

func (m Message) IsToolCallMessage() bool {
	return len(m.ToolCalls) > 0
}

func (m Message) IsToolMessage() bool {
	return strings.ToLower(m.Role) == "tool"
}

func (m Message) IsContentMessage() bool {
	if m.IsToolMessage() {
		return false
	}
	content, ok := m.Content.(string)
	if ok && content != "" {
		return true
	}
	list, ok := m.Content.([]any)
	if ok && len(list) > 0 {
		return true
	}
	return false
}
func (m Message) StringContent() string {
	content, ok := m.Content.(string)
	if ok {
		return content
	}
	contentList, ok := m.Content.([]any)
	if ok {
		var contentStr string
		for _, contentItem := range contentList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == ContentTypeText {
				if subStr, ok := contentMap["text"].(string); ok {
					contentStr += subStr
				}
			}
		}
		return contentStr
	}
	return ""
}

func (m Message) StringReasoningContent() string {
	content, ok := m.ReasoningContent.(string)
	if ok {
		return content
	}
	contentList, ok := m.ReasoningContent.([]any)
	if ok {
		var contentStr string
		for _, contentItem := range contentList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == ContentTypeText {
				if subStr, ok := contentMap["text"].(string); ok {
					contentStr += subStr
				}
			}
		}
		return contentStr
	}
	return ""
}

func (m Message) ParseContent() []MessageContent {
	var contentList []MessageContent
	content, ok := m.Content.(string)
	if ok {
		contentList = append(contentList, MessageContent{
			Type: ContentTypeText,
			Text: content,
		})
		return contentList
	}
	anyList, ok := m.Content.([]any)
	if ok {
		for _, contentItem := range anyList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			switch contentMap["type"] {
			case ContentTypeText:
				if subStr, ok := contentMap["text"].(string); ok {
					contentList = append(contentList, MessageContent{
						Type: ContentTypeText,
						Text: subStr,
					})
				}
			case ContentTypeImageURL:
				if subObj, ok := contentMap["image_url"].(map[string]any); ok {
					contentList = append(contentList, MessageContent{
						Type: ContentTypeImageURL,
						ImageURL: &ImageURL{
							Url: subObj["url"].(string),
						},
					})
				}
			}
		}
		return contentList
	}
	return nil
}

type ImageURL struct {
	Url    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type MessageContent struct {
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}
