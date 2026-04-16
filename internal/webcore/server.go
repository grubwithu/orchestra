package webcore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/grubwithu/orchestra/internal/plugin"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/dict"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/fuzzer"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/logger"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/prerun"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/seed"
)

const (
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

type Server struct {
	Router  *gin.Engine
	Port    int
	Verbose bool

	Ready      bool
	ReadyMutex sync.Mutex

	// Plugin system
	PluginRegistry *plugin.Registry
}

func NewServer(port int, progPath string, fuzzIntroPrefix string, srcPathMatch string, output string, verbose bool) *Server {
	router := gin.Default()

	// Initialize plugin registry
	pluginRegistry := plugin.NewRegistry()

	// Register default plugins
	prerunPlugin := prerun.NewPlugin()
	seedPlugin := seed.NewPlugin()
	dictPlugin := dict.NewPlugin()
	fuzzerPlugin := fuzzer.NewPlugin()
	loggerPlugin := logger.NewPlugin()

	if err := pluginRegistry.Register(prerunPlugin); err != nil {
		log.Printf("Error registering prerun plugin: %v\n", err)
	}
	if err := pluginRegistry.Register(seedPlugin); err != nil {
		log.Printf("Error registering seed plugin: %v\n", err)
	}
	if err := pluginRegistry.Register(dictPlugin); err != nil {
		log.Printf("Error registering dict plugin: %v\n", err)
	}
	if err := pluginRegistry.Register(fuzzerPlugin); err != nil {
		log.Printf("Error registering fuzzer plugin: %v\n", err)
	}
	if err := pluginRegistry.Register(loggerPlugin); err != nil {
		log.Printf("Error registering logger plugin: %v\n", err)
	}

	server := &Server{
		Router:         router,
		Port:           port,
		PluginRegistry: pluginRegistry,
		Ready:          false,
		Verbose:        verbose,
	}
	server.setupRoutes()
	go server.initPlugins(progPath, fuzzIntroPrefix, srcPathMatch, output)
	return server
}

func (s *Server) initPlugins(progPath string, fuzzIntroPrefix string, srcPathMatch string, output string) {
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
			Output:          output,
			Verbose:         s.Verbose,
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
