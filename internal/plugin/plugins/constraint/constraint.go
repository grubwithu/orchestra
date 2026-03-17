package constraint

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"

	"github.com/grubwithu/hfc/internal/analysis"
	"github.com/grubwithu/hfc/internal/plugin"
	"github.com/grubwithu/hfc/internal/plugin/plugins/prerun"
)

// Plugin handles constraint data processing
type Plugin struct {
	config           plugin.PluginConfig
	fuzzerBeginCov   map[string][]analysis.FileLineCov
	constraintGroups []analysis.ConstraintGroup
	fuzzerScores     map[string]analysis.ConstraintScore
	mutex            sync.RWMutex
}

// NewPlugin creates a new constraint plugin
func NewPlugin() *Plugin {
	return &Plugin{
		fuzzerBeginCov:   make(map[string][]analysis.FileLineCov),
		constraintGroups: []analysis.ConstraintGroup{},
		fuzzerScores:     make(map[string]analysis.ConstraintScore),
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "constraint"
}

// Require checks if the plugin should process the given data
// Constraint plugin requires prerun data
func (p *Plugin) Require(data *plugin.PluginData) bool {
	_, ok := data.Data["prerun"].(prerun.PrerunData)
	return ok
}

// Init initializes the plugin
func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	log.Printf("ConstraintPlugin initialized\n")
	return nil
}

// Process processes constraint data
func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Get prerun results
	prerunResult, ok := data.Data["prerun"].(prerun.PrerunData)
	if !ok {
		return errors.New("prerun data not found")
	}

	// Handle begin period
	if data.Period == "begin" {
		p.fuzzerBeginCov[data.Fuzzer] = prerunResult.LineCov
		log.Printf("Saved baseline coverage for fuzzer: %s\n", data.Fuzzer)
		return nil
	}

	// Get AST and SourceCode from prerun result
	ast := prerunResult.AST
	sourceCode := prerunResult.SourceCode

	// Calculate constraint groups if we have all necessary data
	if ast != nil && sourceCode != nil && p.config.CallTree != nil {
		// Identify important constraints
		constraints := analysis.IdentifyImportantConstraints(p.config.CallTree, &prerunResult.ProgCov)
		if len(constraints) > 0 {
			// Group constraints by function
			groups := analysis.GroupConstraintsByFunction(
				constraints,
				&prerunResult.ProgCov,
				ast,
				sourceCode,
				p.config.CallTree.ProgramProfile.AllFunctions.Elements,
			)

			// Sort groups by importance
			sort.Slice(groups, func(i, j int) bool {
				return groups[i].TotalImportance > groups[j].TotalImportance
			})

			// Update constraint groups
			p.constraintGroups = groups
		}
	}

	// Calculate fuzzer scores if we have line coverage
	if ast != nil && sourceCode != nil {
		// Get previous file line coverage
		var prevFileLineCovs []analysis.FileLineCov
		if fileLineCovs, ok := p.fuzzerBeginCov[data.Fuzzer]; ok {
			prevFileLineCovs = fileLineCovs
		}

		// Get important functions
		importantFunctions := []string{}
		for _, group := range p.constraintGroups {
			importantFunctions = append(importantFunctions, group.MainFunction)
		}

		// Calculate score
		score := analysis.CalculateFuzzerScore(data.Fuzzer, prerunResult.LineCov, prevFileLineCovs, ast, sourceCode, importantFunctions)

		// Update fuzzer score
		if existingScore, exists := p.fuzzerScores[data.Fuzzer]; exists {
			p.fuzzerScores[data.Fuzzer] = analysis.UpdateFuzzerScore(score, existingScore)
		} else {
			p.fuzzerScores[data.Fuzzer] = score
		}
	}

	log.Printf("Processed constraint data for fuzzer: %s\n", data.Fuzzer)
	return nil
}

// Result returns the current state/result of the plugin
func (p *Plugin) Result(ctx context.Context) (any, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return map[string]any{
		"constraint_groups": p.constraintGroups,
		"fuzzer_scores":     p.fuzzerScores,
	}, nil
}

// Cleanup cleans up the plugin resources
func (p *Plugin) Cleanup(ctx context.Context) error {
	log.Println("ConstraintPlugin cleanup")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.fuzzerBeginCov = make(map[string][]analysis.FileLineCov)
	p.constraintGroups = []analysis.ConstraintGroup{}
	p.fuzzerScores = make(map[string]analysis.ConstraintScore)
	return nil
}

// Priority returns the plugin priority
func (p *Plugin) Priority() int {
	return 100 // Lower priority than coverage
}
