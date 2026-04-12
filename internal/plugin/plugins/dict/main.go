package dict

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/grubwithu/orchestra/internal/plugin"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/prerun"
	"github.com/grubwithu/orchestra/internal/plugin/plugins/seed"
	sitter "github.com/smacker/go-tree-sitter"
)

const PLUGIN_NAME = "dict"

type DictItem struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type FunctionDict struct {
	FunctionName string     `json:"function_name"`
	DictItems    []DictItem `json:"dict_items"`
}

type DictResult struct {
	Content string `json:"content,omitempty"`
}

type Plugin struct {
	config     plugin.PluginConfig
	ast        map[string]*sitter.Tree
	sourceCode map[string][]byte
	funcDicts  map[string]FunctionDict
	mutex      sync.RWMutex
}

func NewPlugin() *Plugin {
	return &Plugin{
		funcDicts: make(map[string]FunctionDict),
	}
}

func (p *Plugin) Name() string {
	return PLUGIN_NAME
}

func (p *Plugin) Require(data *plugin.PluginData) bool {
	_, ok := data.Data[prerun.PLUGIN_NAME].(*prerun.PrerunData)
	if !ok {
		return false
	}
	_, ok = data.Data[seed.PLUGIN_NAME].(*seed.SeedData)
	if !ok {
		return false
	}
	return data.Period != "begin"
}

func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	p.Log(ctx, "Initialized\n")
	return nil
}

func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	prerunData, ok := data.Data[prerun.PLUGIN_NAME].(*prerun.PrerunData)
	if !ok {
		return fmt.Errorf("prerun data not found for dict: %s", data.Fuzzer)
	}

	p.ast = prerunData.AST
	p.sourceCode = prerunData.SourceCode

	seedData, ok := data.Data[seed.PLUGIN_NAME].(*seed.SeedData)
	if !ok {
		return nil
	}

	constraintGroups := seedData.ConstraintGroups
	if len(constraintGroups) == 0 {
		return nil
	}

	funcNamesToProcess := make(map[string]bool)
	for _, group := range constraintGroups {
		for _, profile := range group.PathDetail {
			if profile != nil {
				funcNamesToProcess[profile.FunctionName] = true
			}
		}
	}

	for funcName := range funcNamesToProcess {
		if _, exists := p.funcDicts[funcName]; exists {
			continue
		}

		funcProfile := p.findFunctionProfile(prerunData.CallTree, funcName)
		if funcProfile == nil {
			continue
		}

		dictItems := p.computeDictForFunction(funcProfile)
		p.funcDicts[funcName] = FunctionDict{
			FunctionName: funcName,
			DictItems:    dictItems,
		}
	}

	return nil
}

func (p *Plugin) Result(ctx context.Context, previousResults map[string]any) (any, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var result DictResult

	seedResult, ok := previousResults[seed.PLUGIN_NAME].(*seed.SeedResult)
	if !ok {
		return result, nil
	}

	constraintGroup := seedResult.ConstraintGroup
	if constraintGroup == nil {
		return nil, nil
	}

	var lines []string
	itemIndex := 0
	seenValues := make(map[string]bool)

	for _, profile := range constraintGroup.PathDetail {
		if profile == nil {
			continue
		}
		funcName := profile.FunctionName
		if funcDict, exists := p.funcDicts[funcName]; exists {
			for _, item := range funcDict.DictItems {
				if seenValues[item.Value] {
					continue
				}
				seenValues[item.Value] = true
				escapedValue := escapeForDict(item.Value)
				lines = append(lines, fmt.Sprintf("keyword%d=\"%s\"", itemIndex, escapedValue))
				itemIndex++
			}
		}
	}

	result.Content = strings.Join(lines, "\n")
	return &result, nil
}

func (p *Plugin) Cleanup(ctx context.Context) error {
	log.Println("DictPlugin cleanup")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.funcDicts = make(map[string]FunctionDict)
	return nil
}

func (p *Plugin) Priority() int {
	return 400
}

func (p *Plugin) Log(ctx context.Context, format string, args ...any) {
	log.Printf("[DICT] "+format, args...)
}
