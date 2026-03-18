package coverage

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/grubwithu/hfc/internal/analysis"
	"github.com/grubwithu/hfc/internal/plugin"
	"github.com/grubwithu/hfc/internal/plugin/plugins/prerun"
)

// Plugin handles coverage data processing
type Plugin struct {
	config       plugin.PluginConfig
	fuzzerCovs   map[string][]int
	globalCov    *analysis.ProgCovData
	fileLineCovs []analysis.FileLineCov
	mutex        sync.RWMutex
}

// NewPlugin creates a new coverage plugin
func NewPlugin() *Plugin {
	return &Plugin{
		fuzzerCovs:   make(map[string][]int),
		fileLineCovs: []analysis.FileLineCov{},
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "coverage"
}

// Require checks if the plugin should process the given data
// Coverage plugin requires prerun data
func (p *Plugin) Require(data *plugin.PluginData) bool {
	_, ok := data.Data["prerun"].(prerun.PrerunData)
	return ok && data.Period != "begin"
}

// Init initializes the plugin
func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	log.Printf("CoveragePlugin initialized\n")
	return nil
}

// Process processes coverage data
func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Get prerun results
	prerunResult, ok := data.Data["prerun"].(prerun.PrerunData)
	if !ok {
		return errors.New("prerun data not found")
	}

	// Update fuzzer coverage
	if _, exists := p.fuzzerCovs[data.Fuzzer]; !exists {
		// Initialize with __init__ coverage if available
		if initCov, exists := p.fuzzerCovs["__init__"]; exists && len(initCov) > 0 {
			p.fuzzerCovs[data.Fuzzer] = []int{initCov[0], prerunResult.Cov}
		} else {
			p.fuzzerCovs[data.Fuzzer] = []int{prerunResult.Cov}
		}
	} else {
		p.fuzzerCovs[data.Fuzzer] = append(p.fuzzerCovs[data.Fuzzer], prerunResult.Cov)
	}

	// Update global coverage
	if p.globalCov == nil {
		p.globalCov = &prerunResult.ProgCov
	} else {
		// Merge function coverage
		for _, funcCoverage := range prerunResult.ProgCov.Functions {
			existing := false
			for i := range p.globalCov.Functions {
				if p.globalCov.Functions[i].Name == funcCoverage.Name {
					p.globalCov.Functions[i].Count += funcCoverage.Count
					existing = true
					break
				}
			}
			if !existing {
				p.globalCov.Functions = append(p.globalCov.Functions, funcCoverage)
			}
		}
	}

	// Update file line coverage
	p.fileLineCovs = prerunResult.LineCov

	log.Printf("Processed coverage data for fuzzer: %s\n", data.Fuzzer)
	return nil
}

// Result returns the current state/result of the plugin
func (p *Plugin) Result(ctx context.Context) (any, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return map[string]any{
		"fuzzer_covs":    p.fuzzerCovs,
		"global_cov":     p.globalCov,
		"file_line_covs": p.fileLineCovs,
	}, nil
}

// Cleanup cleans up the plugin resources
func (p *Plugin) Cleanup(ctx context.Context) error {
	log.Println("CoveragePlugin cleanup")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.fuzzerCovs = make(map[string][]int)
	p.globalCov = nil
	p.fileLineCovs = []analysis.FileLineCov{}
	return nil
}

// Priority returns the plugin priority
func (p *Plugin) Priority() int {
	return 500 // Medium priority
}
