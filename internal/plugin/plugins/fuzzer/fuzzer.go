package fuzzer

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/grubwithu/orchestra/internal/analysis"
	"github.com/grubwithu/orchestra/internal/plugin"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/prerun"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/seed"
)

// Plugin handles constraint data processing
type Plugin struct {
	config           plugin.PluginConfig
	Job2Fuzzer       map[int]string                      // JobID -> Fuzzer Name
	JobBeginCov      map[int][]analysis.FileLineCov      // JobID -> Seeds Coverage
	fuzzerScores     map[string]analysis.ConstraintScore // Fuzzer Name -> Constraint Score
	fuzzerEfficiency map[string]float64                  // Fuzzer Name -> Efficiency
	fuzzerCovSeq     map[string][]int                    // Fuzzer Name -> Coverage Sequence
	mutex            sync.RWMutex
}

// NewPlugin creates a new constraint plugin
func NewPlugin() *Plugin {
	return &Plugin{
		Job2Fuzzer:       make(map[int]string),
		JobBeginCov:      make(map[int][]analysis.FileLineCov),
		fuzzerScores:     make(map[string]analysis.ConstraintScore),
		fuzzerEfficiency: make(map[string]float64),
		fuzzerCovSeq:     make(map[string][]int),
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "fuzzer"
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
	p.Log(ctx, "Initialized\n")
	return nil
}

// Process processes constraint data
func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Get prerun results
	prerunData, ok := data.Data["prerun"].(prerun.PrerunData)
	if !ok {
		return fmt.Errorf("prerun data not found for fuzzer: %s", data.Fuzzer)
	}

	// Update fuzzer coverage
	if _, exists := p.fuzzerCovSeq[data.Fuzzer]; !exists {
		// Initialize with __init__ coverage if available
		if initCov, exists := p.fuzzerCovSeq["__init__"]; exists && len(initCov) > 0 {
			p.fuzzerCovSeq[data.Fuzzer] = []int{initCov[0], prerunData.Cov}
		} else {
			p.fuzzerCovSeq[data.Fuzzer] = []int{prerunData.Cov}
		}
	} else {
		p.fuzzerCovSeq[data.Fuzzer] = append(p.fuzzerCovSeq[data.Fuzzer], prerunData.Cov)
	}

	// Handle begin period
	if data.Period == "begin" {
		p.JobBeginCov[data.JobID] = prerunData.LineCov
		p.Job2Fuzzer[data.JobID] = data.Fuzzer
		p.Log(ctx, "Saved baseline coverage for fuzzer: %s\n", data.Fuzzer)
		return nil
	}

	// Get coverage data
	coverageData, ok := data.Data["coverage"].(seed.CoverageData)
	if !ok {
		return fmt.Errorf("coverage data not found for fuzzer: %s", data.Fuzzer)
	}

	// Get AST and SourceCode from prerun result
	ast := prerunData.AST
	sourceCode := prerunData.SourceCode

	// Calculate fuzzer scores if we have line coverage
	if ast != nil && sourceCode != nil {
		// Get previous file line coverage
		var prevFileLineCovs []analysis.FileLineCov
		if fileLineCovs, ok := p.JobBeginCov[data.JobID]; ok {
			prevFileLineCovs = fileLineCovs
			delete(p.JobBeginCov, data.JobID)
		} else {
			return fmt.Errorf("The coverage information of input seeds not found for fuzzer: %s", data.Fuzzer)
		}

		// Calculate efficiency
		seedCov := p.fuzzerCovSeq[data.Fuzzer][len(p.fuzzerCovSeq[data.Fuzzer])-2]
		efficiency := float64(seedCov-prerunData.Cov) / float64(data.Budge)
		p.fuzzerEfficiency[data.Fuzzer] = max(0.0, efficiency)

		// Get important functions
		importantFunctions := []string{}
		for _, group := range coverageData.ConstraintGroups {
			importantFunctions = append(importantFunctions, group.Path...)
		}

		// Calculate score
		score := analysis.CalculateFuzzerScore(data.Fuzzer, prerunData.LineCov, prevFileLineCovs, ast, sourceCode, importantFunctions)

		// Update fuzzer score
		if existingScore, exists := p.fuzzerScores[data.Fuzzer]; exists {
			score = analysis.UpdateFuzzerScore(score, existingScore)
		}
		// Normalize the score
		score = analysis.NormalizeScore(score)

		// find the max efficiency
		maxEfficiency := 0.0
		for _, efficiency := range p.fuzzerEfficiency {
			if efficiency > maxEfficiency {
				maxEfficiency = efficiency
			}
		}
		// Normalize the efficiency coefficient to 0.5-1.5 range
		k := 0.5 + (efficiency/maxEfficiency)*1.0

		for i := range score {
			score[i] *= k
		}
		p.fuzzerScores[data.Fuzzer] = score

		if p.config.Verbose {
			p.Log(ctx, "Process: fuzzer=%s, job=%d, efficiency=%f, score=%v\n", data.Fuzzer, data.JobID, efficiency, score)
		} else {
			p.Log(ctx, "Processed data for fuzzer=%s\n", data.Fuzzer)
		}
		return nil
	} else {
		return fmt.Errorf("AST or SourceCode is nil for fuzzer: %s", data.Fuzzer)
	}
}

// Result returns the current state/result of the plugin
func (p *Plugin) Result(ctx context.Context) (any, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return map[string]any{
		"fuzzer_scores": p.fuzzerScores,
	}, nil
}

// Cleanup cleans up the plugin resources
func (p *Plugin) Cleanup(ctx context.Context) error {
	log.Println("FuzzerPlugin cleanup")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Job2Fuzzer = make(map[int]string)
	p.JobBeginCov = make(map[int][]analysis.FileLineCov)
	p.fuzzerScores = make(map[string]analysis.ConstraintScore)
	p.fuzzerEfficiency = make(map[string]float64)
	p.fuzzerCovSeq = make(map[string][]int)
	return nil
}

// Priority returns the plugin priority
func (p *Plugin) Priority() int {
	return 100 // Lower priority than coverage
}

func (p *Plugin) Log(ctx context.Context, format string, args ...any) {
	log.Printf("[FUZZER] "+format, args...)
}
