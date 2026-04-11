package dict

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/grubwithu/orchestra/internal/analysis"
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
	Content string `json:"content"`
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
	return ok
}

func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
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

	seedData, ok := data.Data[seed.PLUGIN_NAME].(map[string]any)
	if !ok {
		return nil
	}

	constraintGroupsAny := seedData["constraint_groups"]
	constraintGroups, ok := constraintGroupsAny.([]analysis.ConstraintGroup)
	if !ok || len(constraintGroups) == 0 {
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

	seedResult, ok := previousResults[seed.PLUGIN_NAME]
	if !ok {
		return result, nil
	}

	seedResultMap, ok := seedResult.(map[string]any)
	if !ok {
		return result, nil
	}

	constraintGroup, ok := seedResultMap["constraint_group"].(*analysis.ConstraintGroup)
	if !ok || constraintGroup == nil {
		return result, nil
	}

	var lines []string
	itemIndex := 0

	for _, profile := range constraintGroup.PathDetail {
		if profile == nil {
			continue
		}
		funcName := profile.FunctionName
		if funcDict, exists := p.funcDicts[funcName]; exists {
			for _, item := range funcDict.DictItems {
				escapedValue := escapeForDict(item.Value)
				lines = append(lines, fmt.Sprintf("keyword%d=\"%s\"", itemIndex, escapedValue))
				itemIndex++
			}
		}
	}

	result.Content = strings.Join(lines, "\n")
	return result, nil
}

func escapeForDict(s string) string {
	var builder strings.Builder
	for _, ch := range s {
		if ch == '\\' {
			builder.WriteString("\\\\")
		} else if ch == '"' {
			builder.WriteString("\\\"")
		} else if ch >= 0x20 && ch <= 0x7E {
			builder.WriteRune(ch)
		} else {
			fmt.Fprintf(&builder, "\\x%02X", ch)
		}
	}
	return builder.String()
}

func (p *Plugin) findFunctionProfile(callTree analysis.CallTree, funcName string) *analysis.FunctionProfile {
	if callTree.Nodes == nil {
		return nil
	}

	for _, node := range callTree.Nodes {
		if node.FunctionProfile != nil && node.FunctionProfile.FunctionName == funcName {
			return node.FunctionProfile
		}
	}
	return nil
}

func (p *Plugin) computeDictForFunction(funcProfile *analysis.FunctionProfile) []DictItem {
	var dictItems []DictItem

	if funcProfile == nil {
		return dictItems
	}

	tree, hasAST := p.ast[funcProfile.FunctionSourceFile]
	if !hasAST {
		p.extractNumericConstants(funcProfile, &dictItems)
		return dictItems
	}

	sourceCode := p.sourceCode[funcProfile.FunctionSourceFile]
	if sourceCode == nil {
		p.extractNumericConstants(funcProfile, &dictItems)
		return dictItems
	}

	funcNode := p.findFunctionNode(tree, funcProfile.FunctionLinenumber, funcProfile.FunctionName)
	if funcNode == nil {
		p.extractNumericConstants(funcProfile, &dictItems)
		return dictItems
	}

	p.extractStringLiterals(funcNode, sourceCode, &dictItems)
	p.extractNumericConstants(funcProfile, &dictItems)

	return dictItems
}

func (p *Plugin) findFunctionNode(tree *sitter.Tree, lineNumber int, funcName string) *sitter.Node {
	rootNode := tree.RootNode()

	var findFunc func(node *sitter.Node) *sitter.Node
	findFunc = func(node *sitter.Node) *sitter.Node {
		if node.Type() == "function_definition" || node.Type() == "function_declarator" {
			startPoint := node.StartPoint()
			if int(startPoint.Row)+1 == lineNumber {
				return node
			}
		}

		child := node.NamedChild(0)
		for child != nil {
			if result := findFunc(child); result != nil {
				return result
			}
			child = child.NextSibling()
		}
		return nil
	}

	return findFunc(rootNode)
}

func (p *Plugin) extractStringLiterals(node *sitter.Node, sourceCode []byte, dictItems *[]DictItem) {
	var findStrings func(n *sitter.Node)
	findStrings = func(n *sitter.Node) {
		if n.Type() == "string_literal" {
			content := n.Content(sourceCode)
			content = strings.Trim(content, "\"")
			if len(content) > 0 {
				*dictItems = append(*dictItems, DictItem{
					Type:  "string",
					Value: content,
				})
			}
		}

		child := n.NamedChild(0)
		for child != nil {
			findStrings(child)
			child = child.NextSibling()
		}
	}

	findStrings(node)
}

func (p *Plugin) extractNumericConstants(funcProfile *analysis.FunctionProfile, dictItems *[]DictItem) {
	if funcProfile == nil {
		return
	}

	if funcProfile.BranchProfiles != nil {
		for _, branch := range funcProfile.BranchProfiles {
			if branch.ImmediateValue != 0 {
				*dictItems = append(*dictItems, DictItem{
					Type:  "int",
					Value: fmt.Sprintf("%d", branch.ImmediateValue),
				})
			}

			for _, caseVal := range branch.CaseValues {
				*dictItems = append(*dictItems, DictItem{
					Type:  "int",
					Value: fmt.Sprintf("%d", caseVal),
				})
			}
		}
	}

	if funcProfile.ConstantsTouched != nil {
		for _, constant := range funcProfile.ConstantsTouched {
			*dictItems = append(*dictItems, DictItem{
				Type:  "string",
				Value: constant,
			})
		}
	}
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
