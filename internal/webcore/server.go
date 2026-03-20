package webcore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

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

	Ready      bool
	ReadyMutex sync.Mutex

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

	server := &Server{
		Router:         router,
		Port:           port,
		PluginRegistry: pluginRegistry,
		Ready:          false,
	}
	server.setupRoutes()
	go server.initPlugins(progPath, fuzzIntroPrefix, srcPathMatch)
	return server
}

func (s *Server) initPlugins(progPath string, fuzzIntroPrefix string, srcPathMatch string) {
	ctx := context.Background()

	log.Println("Initializing plugins...")
	plugins := s.PluginRegistry.List()

	progPathAbs, err := filepath.Abs(progPath)
	if err != nil {
		log.Printf("Error getting absolute path of program: %v\n", err)
		os.Exit(1)
	}

	fuzzIntroPrefixAbs, err := filepath.Abs(fuzzIntroPrefix)
	if err != nil {
		log.Printf("Error getting absolute path of fuzz intro prefix: %v\n", err)
		os.Exit(1)
	}

	for _, p := range plugins {
		if err := p.Init(ctx, plugin.PluginConfig{
			Executable:      progPathAbs,
			FuzzIntroPrefix: fuzzIntroPrefixAbs,
			SrcPathMatch:    srcPathMatch,
		}); err != nil {
			log.Printf("Error initializing plugin %s: %v\n", p.Name(), err)
			os.Exit(1)
		}
	}

	log.Println("Plugins initialized successfully")

	s.ReadyMutex.Lock()
	defer s.ReadyMutex.Unlock()
	s.Ready = true
}
func (s *Server) Start() error {
	return s.Router.Run(fmt.Sprintf(":%d", s.Port))
}
