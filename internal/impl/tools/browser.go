package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"aiagent/internal/domain/entities"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"go.uber.org/zap"
)

type BrowserTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	browser       *rod.Browser // Persistent browser instance
}

func NewBrowserTool(name, description string, configuration map[string]string, logger *zap.Logger) *BrowserTool {
	bt := &BrowserTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
	if err := bt.initializeBrowser(); err != nil {
		logger.Error("Failed to initialize browser in NewBrowserTool", zap.Error(err))
		// Browser initialization failed; proceed with nil browser, handle in Execute
	}
	return bt
}

func (b *BrowserTool) Name() string {
	return b.name
}

func (b *BrowserTool) Description() string {
	return b.description
}

func (b *BrowserTool) FullDescription() string {
	var builder strings.Builder
	builder.WriteString(b.Description())
	builder.WriteString("\n\nConfiguration for this tool:\n")
	builder.WriteString("| Key           | Value         |\n")
	builder.WriteString("|---------------|---------------|\n")
	for key, value := range b.Configuration() {
		builder.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return builder.String()
}

func (b *BrowserTool) Configuration() map[string]string {
	return b.configuration
}

func (b *BrowserTool) UpdateConfiguration(config map[string]string) {
	b.configuration = config
	// Optionally reinitialize if config changes affect browser (e.g., headless mode)
	if err := b.initializeBrowser(); err != nil {
		b.logger.Error("Failed to reinitialize browser after configuration update", zap.Error(err))
	}
}

func (b *BrowserTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"navigate", "getTitle", "click"},
			Description: "The browser operation to perform",
			Required:    true,
		},
		{
			Name:        "url",
			Type:        "string",
			Description: "The URL for navigation (required for 'navigate')",
			Required:    false,
		},
		{
			Name:        "selector",
			Type:        "string",
			Description: "CSS selector for operations like 'click'",
			Required:    false,
		},
	}
}

func (b *BrowserTool) Execute(arguments string) (string, error) {
	b.logger.Debug("Executing browser operation", zap.String("arguments", arguments))
	if b.browser == nil {
		return "", fmt.Errorf("browser not initialized; check logs for errors")
	}

	var args struct {
		Operation string `json:"operation"`
		Url       string `json:"url"`
		Selector  string `json:"selector"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Operation == "" {
		return "", fmt.Errorf("operation is required")
	}

	switch args.Operation {
	case "navigate":
		if args.Url == "" {
			return "", fmt.Errorf("url is required for navigate")
		}
		page := b.browser.MustPage(args.Url)
		defer page.Close() // Close page, not browser
		return fmt.Sprintf("Navigated to %s successfully", args.Url), nil
	case "getTitle":
		if args.Url == "" {
			return "", fmt.Errorf("url is required for getTitle")
		}
		page := b.browser.MustPage(args.Url)
		defer page.Close()
		title := page.MustInfo().Title
		return title, nil
	case "click":
		if args.Url == "" || args.Selector == "" {
			return "", fmt.Errorf("url and selector are required for click")
		}
		page := b.browser.MustPage(args.Url)
		defer page.Close()
		element := page.MustElement(args.Selector)
		element.MustClick()
		return fmt.Sprintf("Clicked element with selector %s", args.Selector), nil
	default:
		return "", fmt.Errorf("unsupported operation: %s", args.Operation)
	}
}

// Helper method to initialize browser
func (b *BrowserTool) initializeBrowser() error {
	if b.browser != nil {
		return nil // Already initialized
	}
	headless := b.configuration["headless"] == "true"
	launcher := launcher.New().Headless(headless)
	controlURL, err := launcher.Launch()
	if err != nil {
		return err
	}
	b.browser = rod.New().ControlURL(controlURL).MustConnect()
	return nil
}

var _ entities.Tool = (*BrowserTool)(nil) // Confirms interface implementation
