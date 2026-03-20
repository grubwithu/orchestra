package webcore

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/grubwithu/hfc/internal/plugin"
	"github.com/grubwithu/hfc/internal/plugin/plugins/constraint"
	"github.com/grubwithu/hfc/internal/plugin/plugins/coverage"
	"github.com/grubwithu/hfc/internal/plugin/plugins/prerun"
)

const (
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

type Server struct {
	Router *gin.Engine
	Port   int

	// Plugin system
	PluginRegistry *plugin.Registry
}

func NewServer(port int, progPath string, fuzzIntroPrefix string, srcPathMatch string) *Server {
	router := gin.Default()

	// Initialize plugin registry
	pluginRegistry := plugin.NewRegistry()

	// Register default plugins
	prerunPlugin := prerun.NewPlugin()
	coveragePlugin := coverage.NewPlugin()
	constraintPlugin := constraint.NewPlugin()

	if err := pluginRegistry.Register(prerunPlugin); err != nil {
		log.Printf("Error registering prerun plugin: %v\n", err)
	}
	if err := pluginRegistry.Register(coveragePlugin); err != nil {
		log.Printf("Error registering coverage plugin: %v\n", err)
	}
	if err := pluginRegistry.Register(constraintPlugin); err != nil {
		log.Printf("Error registering constraint plugin: %v\n", err)
	}

	// Initialize plugins with PluginConfig
	ctx := context.Background()
	if err := prerunPlugin.Init(ctx, plugin.PluginConfig{
		Executable:      progPath,
		FuzzIntroPrefix: fuzzIntroPrefix,
		SrcPathMatch:    srcPathMatch,
	}); err != nil {
		log.Printf("Error initializing prerun plugin: %v\n", err)
	}

	if err := coveragePlugin.Init(ctx, plugin.PluginConfig{
		Executable: progPath,
	}); err != nil {
		log.Printf("Error initializing coverage plugin: %v\n", err)
	}
	if err := constraintPlugin.Init(ctx, plugin.PluginConfig{
		Executable: progPath,
	}); err != nil {
		log.Printf("Error initializing constraint plugin: %v\n", err)
	}

	server := &Server{
		Router:         router,
		Port:           port,
		PluginRegistry: pluginRegistry,
	}
	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	s.Router.POST("/reportCorpus", s.handleReportCorpus)
	s.Router.GET("/peekResult", s.handlePeekResult)
	s.Router.POST("/log", s.handleLog)
}

func (s *Server) Start() error {
	return s.Router.Run(fmt.Sprintf(":%d", s.Port))
}
