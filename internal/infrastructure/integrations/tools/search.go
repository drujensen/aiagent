package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"aiagent/internal/domain/interfaces"
)

type SearchTool struct {
	configuration map[string]string
}

func NewSearchTool(configuration map[string]string) *SearchTool {
	return &SearchTool{configuration: configuration}
}

func (t *SearchTool) Name() string {
	return "Search"
}

func (t *SearchTool) Description() string {
	// Note: The description seems incorrect for a search tool. It should be updated.
	return "A tool to search for information using DuckDuckGo API"
}

func (t *SearchTool) Configuration() []string {
	return []string{}
}

func (t *SearchTool) Parameters() []interfaces.Parameter {
	return []interfaces.Parameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

func (t *SearchTool) Execute(arguments string) (string, error) {
	var args struct {
		query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("error parsing tool arguments: %v", err)
	}

	query := args.query
	if query == "" {
		return "", fmt.Errorf("search query cannot be empty")
	}

	apiURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json"

	// Create a new HTTP client that follows redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Create a new request with Content-Type header
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request to DuckDuckGo API: %w", err)
	}
	defer resp.Body.Close()

	// Check for redirects
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		// Handle redirect if needed
		location, err := resp.Location()
		if err != nil {
			return "", fmt.Errorf("failed to get redirect location: %w", err)
		}
		return fmt.Sprintf("Redirect detected to: %s", location), nil
	}

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
