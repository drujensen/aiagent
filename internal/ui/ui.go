package ui

import (
	"bytes"
	"embed"
	"html/template"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"slices"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/drujensen/aiagent/internal/impl/tools"

	_ "github.com/drujensen/aiagent/internal/api"
	apicontrollers "github.com/drujensen/aiagent/internal/api/controllers"
	uiapicontrollers "github.com/drujensen/aiagent/internal/ui/controllers"

	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/yuin/goldmark"
	gfmext "github.com/yuin/goldmark/extension"

	"go.uber.org/zap"
)

//go:embed static/* templates/*
var embeddedFiles embed.FS

type UI struct {
	chatService     services.ChatService
	agentService    services.AgentService
	toolService     services.ToolService
	providerService services.ProviderService
	logger          *zap.Logger
}

func NewUI(chatService services.ChatService, agentService services.AgentService, toolService services.ToolService, providerService services.ProviderService, logger *zap.Logger) *UI {
	return &UI{
		chatService:     chatService,
		agentService:    agentService,
		toolService:     toolService,
		providerService: providerService,
		logger:          logger,
	}
}

func (u *UI) Run() error {
	funcMap := template.FuncMap{
		"renderMarkdown": renderMarkdown,
		"inArray": func(value string, array []string) bool {
			return slices.Contains(array, value)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"formatNumber": func(num int) string {
			return humanize.Comma(int64(num))
		},
		"collectModelNames": func(models []entities.ModelPricing) []string {
			names := make([]string, 0, len(models))
			for _, model := range models {
				names = append(names, model.Name)
			}
			return names
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(embeddedFiles, "templates/*.html")
	if err != nil {
		u.logger.Fatal("Failed to parse templates", zap.Error(err))
	}

	homeController := uiapicontrollers.NewHomeController(u.logger, tmpl, u.chatService, u.agentService, u.toolService)
	agentController := uiapicontrollers.NewAgentController(u.logger, tmpl, u.agentService, u.toolService, u.providerService)
	chatController := uiapicontrollers.NewChatController(u.logger, tmpl, u.chatService, u.agentService)
	toolFactory, err := tools.NewToolFactory()
	if err != nil {
		u.logger.Fatal("Failed to initialize tool factory", zap.Error(err))
	}
	toolController := uiapicontrollers.NewToolController(u.logger, tmpl, u.toolService, toolFactory)
	providerController := uiapicontrollers.NewProviderController(u.logger, tmpl, u.providerService)

	apiAgentController := apicontrollers.NewAgentController(u.logger, u.agentService)
	apiChatController := apicontrollers.NewChatController(u.logger, u.chatService)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("logger", u.logger)
			return next(c)
		}
	})

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Content-Language", "en")
			return next(c)
		}
	})

	// serve static files from embedded
	e.GET("/static/*", func(c echo.Context) error {
		path := c.Param("*")
		filePath := "static/" + path
		file, err := embeddedFiles.Open(filePath)
		if err != nil {
			u.logger.Error("Failed to open static file", zap.String("path", filePath), zap.Error(err))
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}
		defer file.Close()

		// Determine MIME type based on file extension
		ext := filepath.Ext(path)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream" // Fallback MIME type
		}

		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			u.logger.Error("Failed to read static file", zap.String("path", filePath), zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read file")
		}

		return c.Blob(http.StatusOK, mimeType, content)
	})

	homeController.RegisterRoutes(e)
	agentController.RegisterRoutes(e)
	chatController.RegisterRoutes(e)
	toolController.RegisterRoutes(e)
	providerController.RegisterRoutes(e)

	api := e.Group("/api")
	apiAgentController.RegisterRoutes(api)
	apiChatController.RegisterRoutes(api)

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	u.logger.Info("Starting HTTP server on :8080")
	if err := e.Start(":8080"); err != nil {
		u.logger.Fatal("Failed to start server", zap.Error(err))
	}
	return nil
}

func renderMarkdown(markdown string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(gfmext.GFM)).Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
