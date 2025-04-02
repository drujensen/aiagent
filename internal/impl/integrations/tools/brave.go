package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type BraveTool struct {
	configuration map[string]string
	logger        *zap.Logger
	rateLimit     struct {
		sync.Mutex
		perSecond int
		perMonth  int
		second    int
		month     int
		lastReset time.Time
	}
}

// NewBraveTool creates a new instance of BraveTool.
func NewBraveTool(configuration map[string]string, logger *zap.Logger) *BraveTool {
	return &BraveTool{
		configuration: configuration,
		logger:        logger,
		rateLimit: struct {
			sync.Mutex
			perSecond int
			perMonth  int
			second    int
			month     int
			lastReset time.Time
		}{
			perSecond: 1,
			perMonth:  15000,
			lastReset: time.Now(),
		},
	}
}

// Name returns the name of the tool.
func (t *BraveTool) Name() string {
	return "Brave Search"
}

// Description returns a description of the tool.
func (t *BraveTool) Description() string {
	return "Tools for web and local search using Brave Search API. Available tools: brave_web_search, brave_local_search"
}

// Configuration returns the required configuration keys.
func (t *BraveTool) Configuration() []string {
	return []string{"brave_api_key"}
}

// Parameters returns the parameters required by the tool.
func (t *BraveTool) Parameters() []interfaces.Parameter {
	return []interfaces.Parameter{
		{
			Name:        "tool",
			Type:        "string",
			Description: "The specific tool to use: 'brave_web_search' or 'brave_local_search'",
			Required:    true,
		},
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query (max 400 chars for web, location-based for local)",
			Required:    true,
		},
		{
			Name:        "count",
			Type:        "integer",
			Description: "Number of results (1-20, default 10 for web, 5 for local)",
			Required:    false,
		},
		{
			Name:        "offset",
			Type:        "integer",
			Description: "Pagination offset for web search (0-9, default 0)",
			Required:    false,
		},
	}
}

// Rate limiting
func (t *BraveTool) checkRateLimit() error {
	t.rateLimit.Lock()
	defer t.rateLimit.Unlock()

	now := time.Now()
	if now.Sub(t.rateLimit.lastReset) > time.Second {
		t.rateLimit.second = 0
		t.rateLimit.lastReset = now
	}

	if t.rateLimit.second >= t.rateLimit.perSecond || t.rateLimit.month >= t.rateLimit.perMonth {
		return fmt.Errorf("rate limit exceeded")
	}

	t.rateLimit.second++
	t.rateLimit.month++
	return nil
}

// Search arguments structure
type searchArgs struct {
	Tool   string `json:"tool"`
	Query  string `json:"query"`
	Count  int    `json:"count,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// Brave API response structures
type webResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type locationResult struct {
	ID string `json:"id"`
}

type braveWebResponse struct {
	Web       struct{ Results []webResult }      `json:"web"`
	Locations struct{ Results []locationResult } `json:"locations"`
}

type bravePoi struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address struct {
		StreetAddress   string `json:"streetAddress"`
		AddressLocality string `json:"addressLocality"`
		AddressRegion   string `json:"addressRegion"`
		PostalCode      string `json:"postalCode"`
	} `json:"address"`
	Coordinates struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"coordinates"`
	Phone  string `json:"phone"`
	Rating struct {
		RatingValue float64 `json:"ratingValue"`
		RatingCount int     `json:"ratingCount"`
	} `json:"rating"`
	OpeningHours []string `json:"openingHours"`
	PriceRange   string   `json:"priceRange"`
}

type bravePoiResponse struct {
	Results []bravePoi `json:"results"`
}

type braveDescriptionResponse struct {
	Descriptions map[string]string `json:"descriptions"`
}

// Execute performs the search based on the specified tool
func (t *BraveTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing search", zap.String("arguments", arguments))

	// Parse arguments
	var args searchArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	if args.Query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Get API key
	apiKey, ok := t.configuration["brave_api_key"]
	if !ok {
		t.logger.Error("BRAVE_API_KEY not found in configuration")
		return "", fmt.Errorf("brave_api_key not found in configuration")
	}

	switch args.Tool {
	case "brave_web_search":
		return t.performWebSearch(apiKey, args.Query, args.Count, args.Offset)
	case "brave_local_search":
		return t.performLocalSearch(apiKey, args.Query, args.Count)
	default:
		return "", fmt.Errorf("unknown tool: %s", args.Tool)
	}
}

func (t *BraveTool) performWebSearch(apiKey, query string, count, offset int) (string, error) {
	if err := t.checkRateLimit(); err != nil {
		return "", err
	}

	if count <= 0 || count > 20 {
		count = 10
	}
	if offset < 0 || offset > 9 {
		offset = 0
	}

	apiURL := "https://api.search.brave.com/res/v1/web/search"
	u, _ := url.Parse(apiURL)
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", count))
	q.Set("offset", fmt.Sprintf("%d", offset))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %d %s\n%s", resp.StatusCode, resp.Status, string(body))
	}

	var result braveWebResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	var output strings.Builder
	for i, r := range result.Web.Results {
		if i > 0 {
			output.WriteString("\n\n")
		}
		output.WriteString(fmt.Sprintf("Title: %s\nDescription: %s\nURL: %s", r.Title, r.Description, r.URL))
	}
	if output.Len() == 0 {
		return "No results found", nil
	}
	return output.String(), nil
}

func (t *BraveTool) performLocalSearch(apiKey, query string, count int) (string, error) {
	if err := t.checkRateLimit(); err != nil {
		return "", err
	}

	if count <= 0 || count > 20 {
		count = 5
	}

	// First get location IDs
	apiURL := "https://api.search.brave.com/res/v1/web/search"
	u, _ := url.Parse(apiURL)
	q := u.Query()
	q.Set("q", query)
	q.Set("search_lang", "en")
	q.Set("result_filter", "locations")
	q.Set("count", fmt.Sprintf("%d", count))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %d %s\n%s", resp.StatusCode, resp.Status, string(body))
	}

	var webResp braveWebResponse
	if err := json.NewDecoder(resp.Body).Decode(&webResp); err != nil {
		return "", err
	}

	locationIDs := make([]string, 0, len(webResp.Locations.Results))
	for _, loc := range webResp.Locations.Results {
		locationIDs = append(locationIDs, loc.ID)
	}

	if len(locationIDs) == 0 {
		return t.performWebSearch(apiKey, query, count, 0)
	}

	// Get POI and descriptions in parallel
	var poisResp bravePoiResponse
	var descResp braveDescriptionResponse
	var wg sync.WaitGroup
	var poiErr, descErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		poisResp, poiErr = t.getPois(apiKey, locationIDs)
	}()
	go func() {
		defer wg.Done()
		descResp, descErr = t.getDescriptions(apiKey, locationIDs)
	}()
	wg.Wait()

	if poiErr != nil {
		return "", poiErr
	}
	if descErr != nil {
		return "", descErr
	}

	return t.formatLocalResults(poisResp, descResp), nil
}

func (t *BraveTool) getPois(apiKey string, ids []string) (bravePoiResponse, error) {
	if err := t.checkRateLimit(); err != nil {
		return bravePoiResponse{}, err
	}

	u, _ := url.Parse("https://api.search.brave.com/res/v1/local/pois")
	q := u.Query()
	for _, id := range ids {
		q.Add("ids", id)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return bravePoiResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return bravePoiResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return bravePoiResponse{}, fmt.Errorf("API error: %d %s\n%s", resp.StatusCode, resp.Status, string(body))
	}

	var result bravePoiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return bravePoiResponse{}, err
	}
	return result, nil
}

func (t *BraveTool) getDescriptions(apiKey string, ids []string) (braveDescriptionResponse, error) {
	if err := t.checkRateLimit(); err != nil {
		return braveDescriptionResponse{}, err
	}

	u, _ := url.Parse("https://api.search.brave.com/res/v1/local/descriptions")
	q := u.Query()
	for _, id := range ids {
		q.Add("ids", id)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return braveDescriptionResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return braveDescriptionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return braveDescriptionResponse{}, fmt.Errorf("API error: %d %s\n%s", resp.StatusCode, resp.Status, string(body))
	}

	var result braveDescriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return braveDescriptionResponse{}, err
	}
	return result, nil
}

func (t *BraveTool) formatLocalResults(pois bravePoiResponse, desc braveDescriptionResponse) string {
	var output strings.Builder
	for i, poi := range pois.Results {
		if i > 0 {
			output.WriteString("\n---\n")
		}
		address := []string{
			poi.Address.StreetAddress,
			poi.Address.AddressLocality,
			poi.Address.AddressRegion,
			poi.Address.PostalCode,
		}
		addrStr := strings.Join(filterEmpty(address), ", ")
		if addrStr == "" {
			addrStr = "N/A"
		}

		hours := strings.Join(poi.OpeningHours, ", ")
		if hours == "" {
			hours = "N/A"
		}

		rating := "N/A"
		if poi.Rating.RatingValue > 0 {
			rating = fmt.Sprintf("%.1f (%d reviews)", poi.Rating.RatingValue, poi.Rating.RatingCount)
		}

		output.WriteString(fmt.Sprintf(`Name: %s
Address: %s
Phone: %s
Rating: %s
Price Range: %s
Hours: %s
Description: %s`,
			poi.Name,
			addrStr,
			nullString(poi.Phone, "N/A"),
			rating,
			nullString(poi.PriceRange, "N/A"),
			hours,
			nullString(desc.Descriptions[poi.ID], "No description available")))
	}
	if output.Len() == 0 {
		return "No local results found"
	}
	return output.String()
}

func filterEmpty(slice []string) []string {
	var result []string
	for _, s := range slice {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func nullString(s, defaultStr string) string {
	if s == "" {
		return defaultStr
	}
	return s
}
