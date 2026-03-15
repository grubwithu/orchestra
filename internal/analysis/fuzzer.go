package analysis

import (
	"log"
	"sort"
	"strings"

	"github.com/gookit/goutil/arrutil"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

type ConstraintType string

const (
	// 值比较类
	CT_EQUALITY_COMPARISON    ConstraintType = "eq_cmp"   // equality comparison (==, !=)
	CT_RELATIONAL_COMPARISON  ConstraintType = "rel_cmp"  // relational comparison (<, >, <=, >=)
	
	// 运算类
	CT_ARITHMETIC_BASIC      ConstraintType = "arith_basic" // basic arithmetic (+, -, *, /, %)
	CT_BITWISE_OPERATION     ConstraintType = "bit_opr"     // bitwise operation (&, |, ^, ~, <<, >>)
	CT_LOGICAL_OPERATION     ConstraintType = "logical_opr" // logical operation (&&, ||, !)
	
	// 字符串类
	CT_STRING_EXACT_MATCH    ConstraintType = "str_exact"  // exact string match (strcmp, ==)
	CT_STRING_PATTERN_MATCH  ConstraintType = "str_pattern" // pattern string match (strstr, strncmp, memcmp)
	
	// 复杂类型
	CT_POINTER_COMPARISON    ConstraintType = "ptr_cmp"    // pointer comparison
	CT_TYPE_CONVERSION       ConstraintType = "type_conv"  // type conversion/check
	CT_COMPOUND_CONDITION    ConstraintType = "comp_cond"  // compound condition
)

type ConstraintScore map[ConstraintType]float64

func (cs ConstraintScore) Copy() ConstraintScore {
	res := ConstraintScore{}
	for k, v := range cs {
		res[k] = v
	}
	return res
}

func UpdateFuzzerScore(cur ConstraintScore, prev ConstraintScore) ConstraintScore {
	if prev == nil {
		prev = ConstraintScore{}
	}

	for ct := range cur {
		if _, ok := prev[ct]; ok {
			prev[ct] = cur[ct] + prev[ct]/10.0
		} else {
			prev[ct] = cur[ct]
		}
	}
	return prev
}

// line is 0-based
func drillDownToLine(node *sitter.Node, targetLine uint32) *sitter.Node {
	if node.StartPoint().Row == targetLine && node.EndPoint().Row == targetLine {
		return node
	}
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	if cursor.GoToFirstChild() {
		for {
			child := cursor.CurrentNode()
			sRow := child.StartPoint().Row
			eRow := child.EndPoint().Row

			if sRow == targetLine && eRow == targetLine {
				return child
			}

			if sRow <= targetLine && eRow >= targetLine {
				return drillDownToLine(child, targetLine)
			}
			if !cursor.GoToNextSibling() {
				break
			}
		}
	}

	return node
}

func findFunctionAtLine(tree *sitter.Tree, line uint32) *sitter.Node {
	if tree == nil || tree.RootNode() == nil {
		return nil
	}

	p := sitter.Point{Row: line - 1, Column: 0}
	root := tree.RootNode()
	node := root.NamedDescendantForPointRange(p, p)

	if node == nil {
		return nil
	}

	targetNode := drillDownToLine(node, line-1)

	current := targetNode
	for current != nil {
		if current.Type() == "function_definition" {
			if current.StartPoint().Row+1 <= line && line <= current.EndPoint().Row+1 {
				return current
			}
		}
		current = current.Parent()
	}

	return nil
}

func getFunctionName(node *sitter.Node, sourceCode []byte) string {
	if node == nil {
		return ""
	}

	if node.Type() == "function_definition" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "function_declarator" {
				return getFunctionName(child, sourceCode)
			}
		}
	} else if node.Type() == "function_declarator" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" {
				return getFunctionName(child, sourceCode)
			}
		}
	} else if node.Type() == "identifier" {
		return node.Content(sourceCode)
	}

	return ""
}

// line is 1-based
func findIfStatementAtLine(tree *sitter.Tree, line uint32) *sitter.Node {
	if tree == nil || tree.RootNode() == nil {
		return nil
	}

	p := sitter.Point{Row: line - 1, Column: 0}
	node := tree.RootNode().NamedDescendantForPointRange(p, p)

	if node == nil {
		return nil
	}

	targetNode := drillDownToLine(node, line-1)

	current := targetNode
	for current != nil {
		if current.Type() == "if_statement" {
			if current.StartPoint().Row+1 <= line && line <= current.EndPoint().Row+1 {
				return current
			}
		}
		current = current.Parent()
	}

	return nil
}

// TODO: use Query to boost the performance
// line is 1-based
func findStatementAtLine(tree *sitter.Tree, targetLine uint32) *sitter.Node {
	if tree == nil || tree.RootNode() == nil {
		return nil
	}

	var foundNode *sitter.Node
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	for {
		node := cursor.CurrentNode()
		startLine := node.StartPoint().Row + 1 // tree-sitter的行号从0开始

		if startLine == targetLine && node.Type() != "if_statement" && strings.HasSuffix(node.Type(), "_statement") {
			foundNode = node
			break
		}

		if cursor.GoToFirstChild() {
			continue
		}

		if cursor.GoToNextSibling() {
			continue
		}

		for {
			if !cursor.GoToParent() {
				return foundNode
			}
			if cursor.GoToNextSibling() {
				break
			}
		}
	}
	return foundNode
}

func findNearestIfStatement(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for {
		if node.Type() == "if_statement" {
			return node
		}
		node = node.Parent()
		if node == nil {
			return nil
		}
	}
}

func findIfCondition(node *sitter.Node) *sitter.Node {
	if node.Type() != "if_statement" {
		return nil
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "parenthesized_expression" || child.Type() == "condition_clause" {
			return child
		}
	}
	return nil
}

func findIfBody(node *sitter.Node) *sitter.Node {
	if node.Type() != "if_statement" {
		return nil
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if strings.HasSuffix(child.Type(), "_statement") {
			return child
		}
	}
	return nil
}

// analyzeIfClause 分析if语句的条件表达式，确定约束类型
func analyzeIfClause(condition *sitter.Node, sourceCode []byte) ConstraintType {
	if condition == nil {
		return CT_COMPOUND_CONDITION
	}

	conditionCode := condition.Content(sourceCode)
	
	// 转换为小写便于匹配
	lowerCode := strings.ToLower(conditionCode)

	// 1. 检查字符串操作
	hasStringLiteral := strings.Contains(conditionCode, "\"") || strings.Contains(conditionCode, "'")
	hasStrcmp := strings.Contains(lowerCode, "strcmp")
	hasStrncmp := strings.Contains(lowerCode, "strncmp")
	hasMemcmp := strings.Contains(lowerCode, "memcmp")
	hasStrstr := strings.Contains(lowerCode, "strstr")
	
	if hasStrcmp || hasStrncmp || hasMemcmp {
		// 精确字符串匹配
		return CT_STRING_EXACT_MATCH
	}
	
	if hasStrstr {
		// 模式匹配
		return CT_STRING_PATTERN_MATCH
	}
	
	if hasStringLiteral {
		// 如果有字符串字面量但没识别到字符串函数，可能是直接比较
		return CT_STRING_EXACT_MATCH
	}

	// 2. 检查指针操作
	if strings.Contains(conditionCode, "*") && (strings.Contains(conditionCode, "==") || 
		strings.Contains(conditionCode, "!=") || strings.Contains(conditionCode, ">") || 
		strings.Contains(conditionCode, "<") || strings.Contains(conditionCode, ">=") || 
		strings.Contains(conditionCode, "<=")) {
		// 检查是否是指针比较（排除乘法运算）
		codeWithoutSpaces := strings.ReplaceAll(conditionCode, " ", "")
		if strings.Contains(codeWithoutSpaces, "*==") || strings.Contains(codeWithoutSpaces, "*!=") ||
			strings.Contains(codeWithoutSpaces, "*>") || strings.Contains(codeWithoutSpaces, "*<") ||
			strings.Contains(codeWithoutSpaces, "*>=") || strings.Contains(codeWithoutSpaces, "*<=") ||
			strings.Contains(conditionCode, "->") {
			return CT_POINTER_COMPARISON
		}
	}

	// 3. 检查类型转换
	if strings.Contains(lowerCode, "sizeof") || strings.Contains(lowerCode, "typeid") ||
		strings.Contains(lowerCode, "dynamic_cast") || strings.Contains(lowerCode, "static_cast") ||
		strings.Contains(lowerCode, "reinterpret_cast") || strings.Contains(lowerCode, "const_cast") ||
		strings.Contains(conditionCode, "(") && strings.Contains(conditionCode, ")") {
		// 简单的类型转换检查
		return CT_TYPE_CONVERSION
	}

	// 4. 检查逻辑运算
	if strings.Contains(conditionCode, "&&") || strings.Contains(conditionCode, "||") ||
		strings.Contains(conditionCode, "!") && !strings.Contains(conditionCode, "!=") {
		return CT_LOGICAL_OPERATION
	}

	// 5. 检查比较运算符
	hasEquality := strings.Contains(conditionCode, "==") || strings.Contains(conditionCode, "!=")
	hasRelational := strings.Contains(conditionCode, ">") || strings.Contains(conditionCode, "<") ||
		strings.Contains(conditionCode, ">=") || strings.Contains(conditionCode, "<=")
	
	if hasEquality && !hasRelational {
		// 只有等值比较
		return CT_EQUALITY_COMPARISON
	}
	
	if hasRelational && !hasEquality {
		// 只有关系比较
		return CT_RELATIONAL_COMPARISON
	}
	
	if hasEquality && hasRelational {
		// 混合比较，可能是复杂条件
		return CT_COMPOUND_CONDITION
	}

	// 6. 检查算术运算
	if strings.Contains(conditionCode, "+") || strings.Contains(conditionCode, "-") ||
		strings.Contains(conditionCode, "*") || strings.Contains(conditionCode, "/") ||
		strings.Contains(conditionCode, "%") {
		return CT_ARITHMETIC_BASIC
	}

	// 7. 检查位运算
	if strings.Contains(conditionCode, "&") || strings.Contains(conditionCode, "|") ||
		strings.Contains(conditionCode, "^") || strings.Contains(conditionCode, "~") ||
		strings.Contains(conditionCode, "<<") || strings.Contains(conditionCode, ">>") {
		return CT_BITWISE_OPERATION
	}

	// 8. 默认返回复合条件
	return CT_COMPOUND_CONDITION
}

func CalculateFuzzerScore(
	fuzzerName string,
	curFileLineCovs []FileLineCov,
	prevFileLineCovs []FileLineCov,
	ast map[string]*sitter.Tree,
	sourceCode map[string][]byte,
	importantFunctions []string,
) ConstraintScore {
	score := ConstraintScore{
		// 值比较类
		CT_EQUALITY_COMPARISON:   0,
		CT_RELATIONAL_COMPARISON: 0,
		
		// 运算类
		CT_ARITHMETIC_BASIC:      0,
		CT_BITWISE_OPERATION:     0,
		CT_LOGICAL_OPERATION:     0,
		
		// 字符串类
		CT_STRING_EXACT_MATCH:    0,
		CT_STRING_PATTERN_MATCH:  0,
		
		// 复杂类型
		CT_POINTER_COMPARISON:    0,
		CT_TYPE_CONVERSION:       0,
		CT_COMPOUND_CONDITION:    0,
	}

	AllIncreaseCount := 0
	ImportantIncreaseCount := 0

	for i := range curFileLineCovs {
		curFileLineCov := &curFileLineCovs[i]
		fileName := curFileLineCov.File

		prevFileLineCov := &prevFileLineCovs[i]
		if prevFileLineCov.File != fileName {
			for i := range prevFileLineCovs {
				if prevFileLineCovs[i].File == fileName {
					prevFileLineCov = &prevFileLineCovs[i]
					break
				}
			}
		}

		// log.Println(fileName)

		tree, hasAST := ast[fileName]
		if !hasAST {
			continue
		}

		for index := 0; index < len(curFileLineCov.Lines); index++ {
			curLine := &curFileLineCov.Lines[index]
			lineNum := curLine.LineNumber
			prevLine := &prevFileLineCov.Lines[index]

			if prevLine.LineNumber != lineNum || index+1 != int(lineNum) {
				log.Printf("Warning: Line number %d in file %s does not match", lineNum, fileName)
				break
			}

			prevCount := prevFileLineCov.Lines[index].Count
			curCount := curLine.Count

			if prevCount == 0 && curCount > 0 {
				/*
					statementNode := findStatementAtLine(tree, lineNum)
					if statementNode == nil {
						continue
					}
					// log.Println("find statement at", lineNum, "type", statementNode.Type())
					ifNode := findNearestIfStatement(statementNode)
				*/
				jumpLine := 0
				ifNode := findIfStatementAtLine(tree, lineNum)
				if ifNode != nil {
					// log.Println("find an increasing if at", lineNum)
					/*
						if_statments contains:
							if
							condition_clause
							*_statement
							else_clause
						or
							if
							condition_clause
							expression_statement
					*/
					body := findIfBody(ifNode)
					// fmt.Println(body.StartPoint().Row+1, lineNum, body.EndPoint().Row+1)
					if body != nil && body.StartPoint().Row+1 <= lineNum && lineNum <= body.EndPoint().Row+1 {
						condition := findIfCondition(ifNode)
						if condition != nil {
							constraintType := analyzeIfClause(condition, sourceCode[fileName])
							score[constraintType] += 1.0
						}
					}
					jumpLine = int(ifNode.EndPoint().Row) - index
					index = int(ifNode.EndPoint().Row)
					// log.Println("jump to ", index)
				}

				functionName := getFunctionName(findFunctionAtLine(tree, lineNum), sourceCode[fileName])
				// log.Println("New line in function ", functionName, " at ", lineNum)

				if arrutil.Contains(importantFunctions, functionName) {
					ImportantIncreaseCount += jumpLine + 1
				}
				AllIncreaseCount += jumpLine + 1

			}
		}
	}
	log.Println("Fuzzer", fuzzerName, "find", AllIncreaseCount, "increases in total,", ImportantIncreaseCount, "of them are important")
	// log.Printf("Fuzzer score calculated: %+v", score)
	return score
}

func findFunctionWithQuery(ast *sitter.Tree, sourceCode []byte, funcName string) *sitter.Node {
	root := ast.RootNode()

	queryStr := `
		(function_definition
			(function_declarator
				(identifier) @target_id
			)
		) @target_def
	`

	q, err := sitter.NewQuery([]byte(queryStr), cpp.GetLanguage())
	if err != nil {
		return nil
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(q, root)

	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}

		var defNode *sitter.Node
		var idName string

		for _, capture := range match.Captures {
			captureName := q.CaptureNameForId(capture.Index)

			switch captureName {
			case "target_id":
				idName = capture.Node.Content(sourceCode)
			case "target_def":
				defNode = capture.Node
			}
		}

		if idName == funcName {
			return defNode
		}
	}

	return nil
}

func analyzeFunction(funcNode *sitter.Node, sourceCode []byte) ConstraintScore {
	if funcNode == nil {
		return ConstraintScore{}
	}

	queryStr := `(if_statement) @target`

	q, err := sitter.NewQuery([]byte(queryStr), cpp.GetLanguage())
	if err != nil {
		return nil
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(q, funcNode)

	score := ConstraintScore{}
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			if q.CaptureNameForId(capture.Index) == "target" {
				condition := findIfCondition(capture.Node)
				if condition != nil {
					constraintType := analyzeIfClause(condition, sourceCode)
					score[constraintType] += 1.0
				}
			}
		}
	}

	// normalize the score
	maxScore := 0.0
	for _, v := range score {
		if v > maxScore {
			maxScore = v
		}
	}
	for k := range score {
		score[k] /= maxScore
	}

	return score
}

func SortConstraintGroup(constraintGroups []ConstraintGroup, fuzzerScore ConstraintScore, ast map[string]*sitter.Tree, sourceCode map[string][]byte) []ConstraintGroup {
	// test whether fuzzerScore is all zero
	allZero := true
	for _, v := range fuzzerScore {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return constraintGroups
	}

	fuzzerCompatibility := map[string]float64{}

	for _, group := range constraintGroups {
		// check whether ast contains group.FileName
		tree, hasAST := ast[group.FileName]
		if !hasAST {
			log.Println("Warning: cannot find ast of " + group.FileName)
			fuzzerCompatibility[group.GroupId] = 0.0
			continue
		}

		funcNode := findFunctionWithQuery(tree, sourceCode[group.FileName], group.MainFunction)
		if funcNode == nil {
			log.Println("Warning: cannot find function " + group.MainFunction + " in file " + group.FileName)
			fuzzerCompatibility[group.GroupId] = 0.0
			continue
		}

		scores := analyzeFunction(funcNode, sourceCode[group.FileName])
		total := 0.0
		for index := range scores {
			total += scores[index] * fuzzerScore[index]
		}
		fuzzerCompatibility[group.GroupId] = total
		log.Println("Function "+group.MainFunction+" ", total)
	}

	// sort constraintGroups by fuzzerCompatibility
	sort.Slice(constraintGroups, func(i, j int) bool {
		return fuzzerCompatibility[constraintGroups[i].GroupId] > fuzzerCompatibility[constraintGroups[j].GroupId]
	})

	return constraintGroups

}
