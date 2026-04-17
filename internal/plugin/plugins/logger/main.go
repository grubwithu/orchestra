package logger

/**
 * Logger plugin exports plugin data to a JSONL file
 * It should run last in the plugin chain to capture all plugin results
 **/

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/grubwithu/orchestra/internal/plugin"
)

const PLUGIN_NAME = "logger"

// LogEntry represents a single log entry in the JSONL file
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Method    string         `json:"method"`
	Period    string         `json:"period"`
	Logs      map[string]any `json:"logs"`
}

// Plugin handles logging of plugin data to JSONL file
type Plugin struct {
	config     plugin.PluginConfig
	outputFile string
	file       *os.File
	mutex      sync.RWMutex
}

// NewPlugin creates a new logger plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return PLUGIN_NAME
}

// Require checks if the plugin should process the given data
// Logger plugin processes all data during "end" period
func (p *Plugin) Require(data *plugin.PluginData) bool {
	return true
}

// Init initializes the plugin
func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config

	// Get output file path from config or use default
	if config.Output != "" {
		// Use the provided output path
		p.outputFile = config.Output
	} else {
		// Default: use current working directory + output.jsonl
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		p.outputFile = filepath.Join(cwd, "output.jsonl")
	}

	// Open file in append mode, create if not exists
	var err error
	p.file, err = os.OpenFile(p.outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	p.Log(ctx, "Initialized with output file: %s\n", p.outputFile)
	return nil
}

// Process processes the data and writes to JSONL file
func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Method:    "process",
		Period:    data.Period,
		Logs:      make(map[string]any),
	}

	for key, value := range data.Data {
		data, ok := value.(plugin.DataExportLog)
		if ok {
			entry.Logs[key] = data.GetLog()
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Write to file with newline
	if _, err := p.file.Write(append(jsonData, '\n')); err != nil {
		return err
	}

	// Flush to ensure data is written
	if err := p.file.Sync(); err != nil {
		return err
	}

	return nil
}

// Result returns nil as logger doesn't provide results
func (p *Plugin) Result(ctx context.Context, previousResults map[string]any) (any, error) {
	return nil, nil
}

// Cleanup closes the file handle
func (p *Plugin) Cleanup(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.file != nil {
		p.Log(ctx, "Closing log file\n")
		return p.file.Close()
	}
	return nil
}

// Priority returns the plugin priority
// Lower priority means it runs later in the chain
// Logger should run last, so use priority 0 (lowest)
func (p *Plugin) Priority() int {
	return 0
}

// Log wrapper for log.Printf
func (p *Plugin) Log(ctx context.Context, format string, args ...any) {
	log.Printf("[LOGGER] "+format, args...)
}
