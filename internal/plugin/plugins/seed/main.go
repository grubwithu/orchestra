package seed

/**
 * Seed plugin handles statistics for seed selection
 * That means:
 * 1. Maintain a global coverage data.
 * 2. Use the global coverage data and static program information to
 *    calculate the most valuable constraint groups.
 * 3. When pfuzzer request for info, return the most valuable constraint groups.
 **/

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/grubwithu/orchestra/internal/analysis"
	"github.com/grubwithu/orchestra/internal/plugin"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/prerun"
)

const PLUGIN_NAME = "seed"

type SeedData struct {
	GlobalCov        *analysis.ProgCovData      `json:"-"`
	ConstraintGroups []analysis.ConstraintGroup `json:"constraint_groups"`
}

type SeedResult struct {
	ConstraintGroup *analysis.ConstraintGroup `json:"constraint_group,omitempty"`
}

// Plugin handles coverage data processing
type Plugin struct {
	config           plugin.PluginConfig
	globalCov        *analysis.ProgCovData
	constraintGroups []analysis.ConstraintGroup
	mutex            sync.RWMutex
}

// NewPlugin creates a new seed plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return PLUGIN_NAME
}

// Require checks if the plugin should process the given data
// Seed plugin requires prerun data
func (p *Plugin) Require(data *plugin.PluginData) bool {
	_, ok := data.Data[prerun.PLUGIN_NAME].(*prerun.PrerunData)
	return ok && data.Period == "end"
}

// Init initializes the plugin
func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	p.Log(ctx, "Initialized\n")
	return nil
}

// Process processes coverage data
func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Get prerun results
	prerunData, ok := data.Data[prerun.PLUGIN_NAME].(*prerun.PrerunData)
	if !ok {
		return fmt.Errorf("prerun data not found for fuzzer: %s", data.Fuzzer)
	}

	// Update global coverage
	if p.globalCov == nil {
		p.globalCov = &prerunData.ProgCov
	} else {
		// Create a map for quick lookup of existing functions
		funcMap := make(map[string]int)
		for i, funcCov := range p.globalCov.Functions {
			funcMap[funcCov.Name] = i
		}

		// Merge function coverage
		for _, incomeFunc := range prerunData.ProgCov.Functions {
			if idx, exists := funcMap[incomeFunc.Name]; exists {
				innerFunc := &p.globalCov.Functions[idx]
				innerFunc.Count += incomeFunc.Count
				for i := range innerFunc.Regions {
					innerFunc.Regions[i][analysis.REGION_EXEC_CNT] += incomeFunc.Regions[i][analysis.REGION_EXEC_CNT]
				}
			} else {
				p.globalCov.Functions = append(p.globalCov.Functions, incomeFunc)
				funcMap[incomeFunc.Name] = len(p.globalCov.Functions) - 1
			}
		}
	}

	// Get AST and SourceCode from prerun result
	prerunData.ASTMutex.Lock()
	ast := prerunData.AST
	sourceCode := prerunData.SourceCode
	callTree := prerunData.CallTree

	// Calculate constraint groups if we have all necessary data
	if ast != nil && sourceCode != nil {
		// Generate constraint groups based on call tree leaf nodes
		groups := analysis.GetConstraintGroups(analysis.InputGetConstraintGroups{
			CallTree:         &callTree,
			ProgCovData:      &prerunData.ProgCov,
			AST:              ast,
			SourceCode:       sourceCode,
			FunctionProfiles: callTree.ProgramProfile.AllFunctions.Elements,
		})

		// Update constraint groups
		p.constraintGroups = groups
	}
	prerunData.ASTMutex.Unlock()

	seedData := &SeedData{
		GlobalCov:        p.globalCov,
		ConstraintGroups: p.constraintGroups,
	}
	data.Data[PLUGIN_NAME] = seedData

	if p.config.Verbose {
		p.Log(ctx, "Process: fuzzer=%s, funcs=%d, groups=%d\n", data.Fuzzer, len(p.globalCov.Functions), len(p.constraintGroups))
	} else {
		p.Log(ctx, "Processed data for fuzzer=%s\n", data.Fuzzer)
	}
	return nil
}

// Result returns the current state/result of the plugin
func (p *Plugin) Result(ctx context.Context, previousResults map[string]any) (any, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var result SeedResult

	// Select a constraint group with weighted random selection
	if len(p.constraintGroups) > 0 {
		result.ConstraintGroup = analysis.SelectConstraintGroup(p.constraintGroups)
	}

	return &result, nil
}

// Cleanup cleans up the plugin resources
func (p *Plugin) Cleanup(ctx context.Context) error {
	log.Println("CoveragePlugin cleanup")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.globalCov = nil
	return nil
}

// Priority returns the plugin priority
func (p *Plugin) Priority() int {
	return 500 // Medium priority
}

func (p *Plugin) Log(ctx context.Context, format string, args ...any) {
	log.Printf("[SEED] "+format, args...)
}
