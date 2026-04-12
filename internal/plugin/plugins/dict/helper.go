package dict

import (
	"fmt"
	"strings"

	"github.com/grubwithu/orchestra/internal/analysis"
)

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
	if callTree.ProgramProfile != nil && callTree.ProgramProfile.AllFunctions.Elements != nil {
		for _, profile := range callTree.ProgramProfile.AllFunctions.Elements {
			if profile.FunctionName == funcName {
				return profile
			}
		}
	}
	return nil
}

func (p *Plugin) computeDictForFunction(funcProfile *analysis.FunctionProfile) []DictItem {
	var dictItems []DictItem

	if funcProfile == nil {
		return dictItems
	}

	p.extractNumericConstants(funcProfile, &dictItems)

	p.extractStringLiterals(funcProfile, &dictItems)

	// tree, hasAST := p.ast[funcProfile.FunctionSourceFile]
	// if !hasAST {
	// 	return dictItems
	// }

	// sourceCode := p.sourceCode[funcProfile.FunctionSourceFile]
	// if sourceCode == nil {
	// 	return dictItems
	// }

	// literals := analysis.ExtractStringLiterals(tree, sourceCode, funcProfile.FunctionName)
	// for _, str := range literals {
	// 	dictItems = append(dictItems, DictItem{
	// 		Type:  "string",
	// 		Value: str,
	// 	})
	// }
	return dictItems
}

func (p *Plugin) extractStringLiterals(funcProfile *analysis.FunctionProfile, dictItems *[]DictItem) {
	if funcProfile == nil {
		return
	}

	if funcProfile.StringLiterals != nil {
		for _, literal := range funcProfile.StringLiterals {
			*dictItems = append(*dictItems, DictItem{
				Type:  "string",
				Value: literal,
			})
		}
	}
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
