package tool

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"strings"
)

type SearchResult struct {
	Id        int    `json:"id"`
	Content   string `json:"content"`
	SourceUrl string `json:"sourceUrl"`
}

func EnhanceSearchPrompt(c *gin.Context, textRequest *relaymodel.GeneralOpenAIRequest) error {
	// Get last user message
	if len(textRequest.Messages) == 0 {
		return nil
	}
	lastUserMessage := textRequest.Messages[len(textRequest.Messages)-1]
	if lastUserMessage.Role != "user" {
		return nil
	}
	// Check if the last user message is a string and not empty
	if _, ok := lastUserMessage.Content.(string); !ok {
		return nil
	} else if lastUserMessage.Content == "" {
		return nil
	}
	if c.GetString(ctxkey.SurfingContext) != "" {
		lastUserMessage.Content = c.GetString(ctxkey.SurfingContext)
		return nil
	}

	query := lastUserMessage.Content.(string)
	resp, err := SearchByTavily(query)
	if err != nil {
		return err
	}
	if len(resp.Results) == 0 {
		logger.SysError("Tavily no search results")
		return nil
	}
	// contruct SearchResult
	var searchResults []SearchResult
	for idx, result := range resp.Results {
		searchResult := SearchResult{
			Id:        idx + 1,
			Content:   result.Content,
			SourceUrl: result.Url,
		}
		searchResults = append(searchResults, searchResult)
	}
	// convert to json
	searchResJson, _ := json.Marshal(searchResults)

	// write a prompt template and translate to English: "请根据参考资料回答问题\n\n## 标注规则：\n- 请在适当的情况下在句子末尾引用上下文。\n- 请按照引用编号[number]的格式在答案中对应部分引用上下文。\n- 如果一句话源自多个上下文，请列出所有相关的引用编号，例如[1][2]，切记不要将引用集中在最后返回引用编号，而是在答案对应部分列出。\n\n## 我的问题是：\n\n{query}\n\n## 参考资料：\n\n```json\n[\n  {\n    \"id\": {id},\n    \"content\": \"{content}\",\n    \"sourceUrl\": \"{source_url}\",\n    \"type\": \"url\"\n  }\n]\n```\n\n请使用同用户问题相同的语言进行回答。\n"
	promptTemplate := "Please answer the question based on the reference materials\n\n## Annotation Rules:\n- Please quote the context at the end of the sentence when appropriate.\n- Please quote the context in the answer in the format of citation number [number].\n- If a sentence comes from multiple contexts, please list all relevant citation numbers, such as [1][2], and remember not to concentrate the citations at the end of the answer, but list them in the corresponding part of the answer.\n\n## My question is:\n\n{query}\n\n## Reference Materials:\n\n```json\n{json}\n```\n\nPlease answer in the same language as the user question.\n"
	// replace {query} with the query, {json} with the json
	prompt := strings.ReplaceAll(promptTemplate, "{query}", query)
	prompt = strings.ReplaceAll(prompt, "{json}", string(searchResJson))
	lastUserMessage.Content = prompt
	// set the prompt to the context
	c.Set(ctxkey.SurfingContext, prompt)
	return nil
}
