package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SearchTool implements the Tool interface for performing searches using DuckDuckGo's Instant Answer API.
// It enables AI agents to retrieve concise web information based on a query string.
//
// Key features:
// - API Integration: Uses DuckDuckGo's free Instant Answer API for search results.
// - Result Parsing: Extracts "AbstractText" or "RelatedTopics" from the JSON response.
// - Error Handling: Manages network errors, invalid responses, and no-result scenarios.
//
// Dependencies:
// - net/http: For making HTTP requests to the DuckDuckGo API.
// - net/url: For escaping query parameters in the URL.
// - encoding/json: For parsing the API's JSON response.
//
// Notes:
// - The Instant Answer API is used (https://api.duckduckgo.com/), which requires no API key.
// - Results prioritize "AbstractText" for direct answers; falls back to "RelatedTopics" if empty.
// - Edge case: Empty queries return an error to prompt agent correction.
// - Limitation: Does not handle rate limiting explicitly; assumes low usage within free tier.
type SearchTool struct{}

// NewSearchTool creates a new SearchTool instance to perform searches using DuckDuckGo's Instant Answer API.
// It ensures the tool is initialized with a valid operation context.
//
// Returns:
// - *FileTool: A new instance of FileTool.
func NewSearchTool() *SearchTool {
	return &SearchTool{}
}

// Name returns the identifier for this tool, used by agents to invoke it.
// It adheres to the Tool interface requirement.
//
// Returns:
// - string: The tool's name, "DuckDuckGoSearch".
func (t *SearchTool) Name() string {
	return "Search"
}

// Execute performs a search using the DuckDuckGo Instant Answer API with the provided query.
// It constructs the API request, retrieves the response, and parses it for relevant content.
//
// Parameters:
// - query: The search string provided by the agent.
//
// Returns:
// - string: The search result (abstract or related topics), or an error message.
// - error: Nil on success, or an error if the request or parsing fails.
//
// Behavior:
// - Escapes the query to handle special characters safely.
// - Returns "No results found" if no abstract or topics are available.
// - Includes detailed error messages for debugging (wrapped with fmt.Errorf).
func (t *SearchTool) Execute(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("search query cannot be empty")
	}

	// Construct the API URL with escaped query
	apiURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json"

	// Perform HTTP GET request
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request to DuckDuckGo API: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Define struct for JSON response parsing
	var result struct {
		AbstractText  string `json:"AbstractText"`
		RelatedTopics []struct {
			Text string `json:"Text"`
		} `json:"RelatedTopics"`
	}

	// Decode JSON response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Return AbstractText if available
	if result.AbstractText != "" {
		return result.AbstractText, nil
	}

	// Fallback to RelatedTopics if AbstractText is empty
	if len(result.RelatedTopics) > 0 {
		var topics []string
		for _, topic := range result.RelatedTopics {
			topics = append(topics, topic.Text)
		}
		return "Related topics: " + fmt.Sprint(topics), nil
	}

	// No results found
	return "No results found", nil
}
