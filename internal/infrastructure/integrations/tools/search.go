package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SearchTool struct{}

func NewSearchTool() *SearchTool {
	return &SearchTool{}
}

func (t *SearchTool) Name() string {
	return "Search"
}

func (t *SearchTool) Execute(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("search query cannot be empty")
	}

	apiURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json"

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request to DuckDuckGo API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		AbstractText  string `json:"AbstractText"`
		RelatedTopics []struct {
			Text string `json:"Text"`
		} `json:"RelatedTopics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if result.AbstractText != "" {
		return result.AbstractText, nil
	}

	if len(result.RelatedTopics) > 0 {
		var topics []string
		for _, topic := range result.RelatedTopics {
			topics = append(topics, topic.Text)
		}
		return "Related topics: " + fmt.Sprint(topics), nil
	}

	return "No results found", nil
}
