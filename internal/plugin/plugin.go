package plugin

import (
	"context"
)

type PluginConfig struct {
	FuzzIntroPrefix string
	Executable      string
	SrcPathMatch    string
	Output          string
	Verbose         bool
}

type PluginData struct {
	// From the request body
	Fuzzer string
	Corpus string
	Period string
	Budget int
	JobID  int

	// Pass data among plugins, temporary storage
	Data map[string]any
}

// Plugin represents the interface that all plugins must implement
type Plugin interface {
	// Name returns the unique name of the plugin
	Name() string

	// Init initializes the plugin with configuration
	Init(ctx context.Context, config PluginConfig) error

	// Require checks if the plugin should process the given data
	// Returns true if the plugin should process this data, false to skip
	Require(data *PluginData) bool

	// Process processes the incoming data
	// The data parameter contains PluginData with shared data from previous plugins
	Process(ctx context.Context, data *PluginData) error

	// Result returns the current state/result of the plugin
	// The previousResults parameter contains results from plugins that were processed before this one
	Result(ctx context.Context, previousResults map[string]any) (any, error)

	// Cleanup releases resources used by the plugin
	Cleanup(ctx context.Context) error

	// Log wrapper for log.Printf
	Log(ctx context.Context, format string, args ...any)
}

type DataExportLog interface {
	GetLog() any
}

// PluginWithPriority represents a plugin that can be prioritized
type PluginWithPriority interface {
	Plugin
	// Priority returns the priority of the plugin (higher = processed first)
	Priority() int
}

// PluginWithValidation represents a plugin that can validate input data
type PluginWithValidation interface {
	Plugin
	// Validate validates the input data before processing
	Validate(ctx context.Context, data *PluginData) error
}

// PluginWithResultMerge represents a plugin that can merge results
type PluginWithResultMerge interface {
	Plugin
	// Merge merges multiple results into one
	Merge(ctx context.Context, results []any) (any, error)
}
