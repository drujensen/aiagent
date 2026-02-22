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
	"sync"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/drujensen/aiagent/internal/impl/config"
	"github.com/drujensen/aiagent/internal/impl/tools"

	uiapicontrollers "github.com/drujensen/aiagent/internal/ui/controllers"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yuin/goldmark"
	gfmext "github.com/yuin/goldmark/extension"

	"go.uber.org/zap"
)

//go:embed static/* templates/*
var embeddedFiles embed.FS

type UI struct {
	chatService         services.ChatService
	agentService        services.AgentService
	modelService        services.ModelService
	toolService         services.ToolService
	providerService     services.ProviderService
	modelRefreshService services.ModelRefreshService
	modelFilterService  *services.ModelFilterService
	globalConfig        *config.GlobalConfig
	logger              *zap.Logger
	wsUpgrader          websocket.Upgrader
	wsClients           map[*websocket.Conn]bool
	wsClientsMutex      sync.RWMutex
}

func NewUI(chatService services.ChatService, agentService services.AgentService, modelService services.ModelService, toolService services.ToolService, providerService services.ProviderService, modelRefreshService services.ModelRefreshService, modelFilterService *services.ModelFilterService, globalConfig *config.GlobalConfig, logger *zap.Logger) *UI {
	ui := &UI{
		chatService:         chatService,
		agentService:        agentService,
		modelService:        modelService,
		toolService:         toolService,
		providerService:     providerService,
		modelRefreshService: modelRefreshService,
		modelFilterService:  modelFilterService,
		globalConfig:        globalConfig,
		logger:              logger,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow connections from any origin for development
			},
		},
		wsClients: make(map[*websocket.Conn]bool),
	}

	return ui
}

// handleWebSocket handles WebSocket connections
func (u *UI) handleWebSocket(c echo.Context) error {
	ws, err := u.wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		u.logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return err
	}
	defer ws.Close()

	// Add client to the clients map
	u.wsClientsMutex.Lock()
	u.wsClients[ws] = true
	u.wsClientsMutex.Unlock()

	u.logger.Info("WebSocket client connected")

	// Clean up when client disconnects
	defer func() {
		u.wsClientsMutex.Lock()
		delete(u.wsClients, ws)
		u.wsClientsMutex.Unlock()
		u.logger.Info("WebSocket client disconnected")
	}()

	// Keep connection alive and handle any incoming messages
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	return nil
}

func (u *UI) Run() error {
	funcMap := template.FuncMap{
		"renderMarkdown":   renderMarkdown,
		"formatToolResult": formatToolResult,
		"formatToolName":   formatToolName,
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

	homeController := uiapicontrollers.NewHomeController(u.logger, tmpl, u.chatService, u.agentService, u.modelService, u.modelFilterService, u.toolService)
	agentController := uiapicontrollers.NewAgentController(u.logger, tmpl, u.agentService, u.toolService, u.providerService)
	chatController := uiapicontrollers.NewChatController(u.logger, tmpl, u.chatService, u.agentService, u.modelService, u.providerService, u.modelFilterService, u.globalConfig)
	toolFactory, err := tools.NewToolFactory()
	if err != nil {
		u.logger.Fatal("Failed to initialize tool factory", zap.Error(err))
	}
	toolController := uiapicontrollers.NewToolController(u.logger, tmpl, u.toolService, toolFactory)
	providerController := uiapicontrollers.NewProviderController(u.logger, tmpl, u.providerService, u.modelRefreshService)

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

	// WebSocket endpoint for real-time updates
	e.GET("/ws", u.handleWebSocket)

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
