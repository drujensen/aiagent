package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"go.uber.org/zap"
)

type BrowserTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	browser       *rod.Browser
	page          *rod.Page
}

func NewBrowserTool(name, description string, configuration map[string]string, logger *zap.Logger) *BrowserTool {
	bt := &BrowserTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
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
}

func (b *BrowserTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"open", "click", "screenshot", "close", "getTitle", "getPageSource", "getElementText", "getElementAttribute", "setInputValue"},
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
		{
			Name:        "value",
			Type:        "string",
			Description: "The value to set for input fields (required for 'setInputValue')",
			Required:    false,
		},
		{
			Name:        "filename",
			Type:        "string",
			Description: "The name of the file to save the screenshot (required for 'screenshot')",
			Required:    false,
		},
	}
}

func (b *BrowserTool) Execute(arguments string) (string, error) {
	b.logger.Debug("Executing browser operation", zap.String("arguments", arguments))
	fmt.Println("\rExecuting browser operation", arguments)

	if err := b.initializeBrowser(); err != nil {
		b.logger.Error("Failed to initialize browser", zap.Error(err))
		return "", fmt.Errorf("browser not initialized; check logs for errors")
	}

	var args struct {
		Operation string `json:"operation"`
		Url       string `json:"url"`
		Selector  string `json:"selector"`
		Value     string `json:"value"`
		Filename  string `json:"filename"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Operation == "" {
		return "", fmt.Errorf("operation is required")
	}

	switch args.Operation {
	case "open":
		if args.Url == "" {
			return "", fmt.Errorf("url is required for navigate")
		}
		b.page = b.browser.MustPage(args.Url)
		if b.page == nil {
			return "", fmt.Errorf("page is not initialized")
		}
		return fmt.Sprintf("Navigated to %s successfully", args.Url), nil
	case "getTitle":
		if b.page == nil {
			return "", fmt.Errorf("page is not initialized")
		}
		title := b.page.MustInfo().Title
		return title, nil
	case "click":
		if b.page == nil {
			return "", fmt.Errorf("page is not initialized")
		}
		if args.Selector == "" {
			return "", fmt.Errorf("selector is required for click")
		}
		element := b.page.MustElement(args.Selector)
		element.MustClick()
		return fmt.Sprintf("Clicked element with selector %s", args.Selector), nil
	case "screenshot":
		if b.page == nil {
			return "", fmt.Errorf("page is not initialized")
		}
		workspace := b.configuration["workspace"]
		if workspace == "" {
			var err error
			workspace, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("could not get current directory: %v", err)
			}
		}
		filename := workspace + "/" + args.Filename
		screenshot, err := b.page.Screenshot(true, nil)
		if err != nil {
			return "", fmt.Errorf("failed to take screenshot: %w", err)
		}
		err = os.WriteFile(filename, screenshot, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to save screenshot: %w", err)
		}
		return "Screenshot saved successfully at " + filename, nil
	case "close":
		if b.page != nil {
			b.page.MustClose()
		}
		return "Browser closed successfully", nil
	case "getPageSource":
		source, err := b.page.HTML()
		if err != nil {
			return "", fmt.Errorf("failed to get page source: %w", err)
		}
		return source, nil
	case "getElementText":
		if args.Selector == "" {
			return "", fmt.Errorf("selector is required for getElementText")
		}
		element := b.page.MustElement(args.Selector)
		text, err := element.Text()
		if err != nil {
			return "", fmt.Errorf("failed to get element text: %w", err)
		}
		return text, nil
	case "getElementAttribute":
		if args.Selector == "" {
			return "", fmt.Errorf("selector is required for getElementAttribute")
		}
		element := b.page.MustElement(args.Selector)
		attribute, err := element.Attribute("value")
		if err != nil {
			return "", fmt.Errorf("failed to get element attribute: %w", err)
		}
		if attribute == nil {
			return "", fmt.Errorf("attribute not found")
		}
		return *attribute, nil
	case "setInputValue":
		if args.Selector == "" {
			return "", fmt.Errorf("selector is required for setInputValue")
		}
		if args.Value == "" {
			return "", fmt.Errorf("value is required for setInputValue")
		}
		element := b.page.MustElement(args.Selector)
		err := element.Input(args.Value)
		if err != nil {
			return "", fmt.Errorf("failed to set input value: %w", err)
		}
		return fmt.Sprintf("Set input value for element with selector %s", args.Selector), nil

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
