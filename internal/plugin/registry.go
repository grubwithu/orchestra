package plugin

import (
	"context"
	"log"
	"sort"
	"sync"
)

// Registry manages the registration and execution of plugins
type Registry struct {
	mu       sync.RWMutex
	plugins  []Plugin
	enabled  map[string]bool
	priority map[string]int
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins:  []Plugin{},
		enabled:  make(map[string]bool),
		priority: make(map[string]int),
	}
}

// Register registers a new plugin
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, ok := r.enabled[name]; ok {
		return PluginAlreadyRegisteredError(name)
	}

	r.plugins = append(r.plugins, p)
	r.enabled[name] = true

	if withPriority, ok := p.(PluginWithPriority); ok {
		r.priority[name] = withPriority.Priority()
	}

	// Sort plugins by priority
	sort.Slice(r.plugins, func(i, j int) bool {
		return r.getPriority(r.plugins[i].Name()) > r.getPriority(r.plugins[j].Name())
	})

	log.Printf("Plugin '%s' registered\n", name)
	return nil
}

// Unregister unregisters a plugin by name
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.enabled[name] {
		return PluginNotFoundError(name)
	}

	for i, p := range r.plugins {
		if p.Name() == name {
			r.plugins = append(r.plugins[:i], r.plugins[i+1:]...)
			break
		}
	}

	delete(r.enabled, name)
	delete(r.priority, name)
	return nil
}

// Get retrieves a plugin by name
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}

// Process processes data through the plugin pipeline
// Each plugin receives the PluginData with results from previous plugins
func (r *Registry) Process(ctx context.Context, data *PluginData) error {
	r.mu.RLock()
	plugins := make([]Plugin, len(r.plugins))
	copy(plugins, r.plugins)
	r.mu.RUnlock()

	for _, p := range plugins {
		if !r.enabled[p.Name()] {
			continue
		}

		// Check if plugin requires this data
		if !p.Require(data) {
			log.Printf("Plugin '%s' skipped (not required)\n", p.Name())
			continue
		}

		// Validate if plugin supports validation
		if withValidation, ok := p.(PluginWithValidation); ok {
			if err := withValidation.Validate(ctx, data); err != nil {
				log.Printf("Validation failed for plugin '%s': %v\n", p.Name(), err)
				return err
			}
		}

		if err := p.Process(ctx, data); err != nil {
			log.Printf("Error processing with plugin '%s': %v\n", p.Name(), err)
			return PluginProcessingFailedError(p.Name(), err)
		}
	}

	return nil
}

// ProcessAll processes data through the entire plugin pipeline
func (r *Registry) ProcessAll(ctx context.Context, data *PluginData) error {
	r.mu.RLock()
	plugins := make([]Plugin, len(r.plugins))
	copy(plugins, r.plugins)
	r.mu.RUnlock()

	if data.Data == nil {
		data.Data = make(map[string]any)
	}

	for _, p := range plugins {
		if !r.enabled[p.Name()] {
			continue
		}

		// Check if plugin requires this data
		if !p.Require(data) {
			log.Printf("Plugin '%s' skipped (not required)\n", p.Name())
			continue
		}

		// Validate if plugin supports validation
		if withValidation, ok := p.(PluginWithValidation); ok {
			if err := withValidation.Validate(ctx, data); err != nil {
				log.Printf("Validation failed for plugin '%s': %v\n", p.Name(), err)
				continue
			}
		}

		if err := p.Process(ctx, data); err != nil {
			log.Printf("Error processing with plugin '%s': %v\n", p.Name(), err)
			continue
		}
	}

	return nil
}

// Results retrieves results from all plugins without processing new data
// This is used for peeking at current plugin states (e.g., handlePeekResult)
func (r *Registry) Results(ctx context.Context) (map[string]any, error) {
	r.mu.RLock()
	plugins := make([]Plugin, len(r.plugins))
	copy(plugins, r.plugins)
	r.mu.RUnlock()

	results := make(map[string]any)

	for _, p := range plugins {
		if !r.enabled[p.Name()] {
			continue
		}

		// Call Result to get current state
		result, err := p.Result(ctx)
		if err != nil {
			log.Printf("Error getting results from plugin '%s': %v\n", p.Name(), err)
			continue
		}
		results[p.Name()] = result
	}

	if len(results) == 0 {
		return nil, NoPluginsEnabledError()
	}

	return results, nil
}

// Enable enables a plugin by name
func (r *Registry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.enabled[name] {
		return PluginNotFoundError(name)
	}
	r.enabled[name] = true
	return nil
}

// Disable disables a plugin by name
func (r *Registry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.enabled[name] {
		return PluginNotFoundError(name)
	}
	r.enabled[name] = false
	return nil
}

// List lists all registered plugins
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Plugin, len(r.plugins))
	copy(result, r.plugins)
	return result
}

// Cleanup calls Cleanup on all registered plugins
func (r *Registry) Cleanup(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if err := p.Cleanup(ctx); err != nil {
			log.Printf("Error cleaning up plugin '%s': %v\n", p.Name(), err)
		}
	}
}

func (r *Registry) getPriority(name string) int {
	if p, ok := r.priority[name]; ok {
		return p
	}
	return 0
}
