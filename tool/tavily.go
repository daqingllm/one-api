package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"io"
	"net/http"
	"time"
)

// Define request payload structure
type TavilyRequest struct {
	Query                    string   `json:"query"`
	Topic                    string   `json:"topic"`
	SearchDepth              string   `json:"search_depth"`
	ChunksPerSource          int      `json:"chunks_per_source"`
	MaxResults               int      `json:"max_results"`
	TimeRange                *string  `json:"time_range"`
	Days                     int      `json:"days"`
	IncludeAnswer            bool     `json:"include_answer"`
	IncludeRawContent        bool     `json:"include_raw_content"`
	IncludeImages            bool     `json:"include_images"`
	IncludeImageDescriptions bool     `json:"include_image_descriptions"`
	IncludeDomains           []string `json:"include_domains"`
	ExcludeDomains           []string `json:"exclude_domains"`
}

type TavilyResponse struct {
	Query        string                 `json:"query"`
	Answer       string                 `json:"answer"`
	Images       []*TavilyImageResponse `json:"images"`
	Results      []*TavilySearchResult  `json:"results"`
	ResponseTime float64                `json:"response_time"`
}

type TavilyImageResponse struct {
	Url         string `json:"url"`
	Description string `json:"description"`
}

type TavilySearchResult struct {
	Title      string  `json:"title,omitempty"`
	Url        string  `json:"url,omitempty"`
	Content    string  `json:"content,omitempty"`
	Score      float64 `json:"score,omitempty"`
	RawContent string  `json:"raw_content,omitempty"`
}

func SearchByTavily(query string) (*TavilyResponse, error) {
	if len(config.TavilyKeys) == 0 {
		return nil, fmt.Errorf("no Tavily keys configured")
	}
	// Create request payload
	reqPayload := TavilyRequest{
		Query:                    query,
		Topic:                    "general",
		SearchDepth:              "basic",
		ChunksPerSource:          3,
		MaxResults:               3,
		TimeRange:                nil,
		Days:                     3,
		IncludeAnswer:            true,
		IncludeRawContent:        false,
		IncludeImages:            false,
		IncludeImageDescriptions: false,
		IncludeDomains:           []string{},
		ExcludeDomains:           []string{},
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	var resp *http.Response
	for _, key := range config.TavilyKeys {
		req.Header.Set("Authorization", key) // Send request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err = client.Do(req)
		if err != nil {
			logger.SysErrorf("Tavily request failed: %s", err.Error())
			continue
		}
		// Handle error response
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			logger.SysErrorf("Tavily request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			continue
		}
	}
	if resp == nil {
		return nil, fmt.Errorf("all Tavily keys failed")
	}
	defer resp.Body.Close()

	// Handle error response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily API error (status %d)", resp.StatusCode)
	}

	// Parse response
	var tavilyResp TavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tavilyResp, nil
}
