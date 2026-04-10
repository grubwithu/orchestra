package analysis

/**
 * GetFuzzerScore - get fuzzer score from constraint groups.
 *
 * We believe that different fuzzers have varying preferences for different types of constraints.
 * Therefore, we can calculate a preference score for each fuzzer regarding various constraint types.
 **/

import (
	"log"
	"math/rand"
	"strings"

	"github.com/gookit/goutil/arrutil"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

type ConstraintType string

const (
	CT_VALUE_COMPARISON     ConstraintType = "val_cmp"  // value comparison
	CT_BITWISE_OPERATION    ConstraintType = "bit_opr"  // bitwise operation
	CT_STRING_MATCH         ConstraintType = "str_mat"  // string match
	CT_ARITHMETIC_OPERATION ConstraintType = "art_opr"  // arithmetic operation
	CT_COMPOUND_OPERATION   ConstraintType = "comp_opr" // compound operation
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

// line is 1-based
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

// analyzeIfClause
func analyzeIfClause(condition *sitter.Node, sourceCode []byte) ConstraintType {
	if condition == nil {
		return CT_COMPOUND_OPERATION
	}

	conditionCode := condition.Content(sourceCode)

	if strings.Contains(conditionCode, "strcmp") || strings.Contains(conditionCode, "strstr") ||
		strings.Contains(conditionCode, "strncmp") || strings.Contains(conditionCode, "memcmp") ||
		strings.Contains(conditionCode, "\"") || strings.Contains(conditionCode, "'") {
		return CT_STRING_MATCH
	}

	if strings.Contains(conditionCode, "+") || strings.Contains(conditionCode, "-") ||
		strings.Contains(conditionCode, "*") || strings.Contains(conditionCode, "/") ||
		strings.Contains(conditionCode, "%") {
		return CT_ARITHMETIC_OPERATION
	}

	// Check for bitwise operations (excluding logical operators)
	if (strings.Contains(conditionCode, "&") && !strings.Contains(conditionCode, "&&")) ||
		(strings.Contains(conditionCode, "|") && !strings.Contains(conditionCode, "||")) ||
		strings.Contains(conditionCode, "^") || strings.Contains(conditionCode, "~") ||
		strings.Contains(conditionCode, "<<") || strings.Contains(conditionCode, ">>") {
		return CT_BITWISE_OPERATION
	}

	if strings.Contains(conditionCode, "==") || strings.Contains(conditionCode, "!=") ||
		strings.Contains(conditionCode, ">") || strings.Contains(conditionCode, "<") ||
		strings.Contains(conditionCode, ">=") || strings.Contains(conditionCode, "<=") {
		return CT_VALUE_COMPARISON
	}

	return CT_COMPOUND_OPERATION
}

type InputCalculateFuzzerScore struct {
	FuzzerName         string
	CurFileLineCovs    []FileLineCov
	PrevFileLineCovs   []FileLineCov
	AST                map[string]*sitter.Tree
	SourceCode         map[string][]byte
	ImportantFunctions []string
}

func CalculateFuzzerScore(input InputCalculateFuzzerScore) ConstraintScore {
	score := ConstraintScore{
		CT_VALUE_COMPARISON:     0,
		CT_BITWISE_OPERATION:    0,
		CT_STRING_MATCH:         0,
		CT_ARITHMETIC_OPERATION: 0,
		CT_COMPOUND_OPERATION:   0,
	}

	AllIncreaseCount := 0
	ImportantIncreaseCount := 0

	for i := range input.CurFileLineCovs {
		curFileLineCov := &input.CurFileLineCovs[i]
		fileName := curFileLineCov.File

		prevFileLineCov := &input.PrevFileLineCovs[i]
		if prevFileLineCov.File != fileName {
			for i := range input.PrevFileLineCovs {
				if input.PrevFileLineCovs[i].File == fileName {
					prevFileLineCov = &input.PrevFileLineCovs[i]
					break
				}
			}
		}

		// log.Println(fileName)

		tree, hasAST := input.AST[fileName]
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
							constraintType := analyzeIfClause(condition, input.SourceCode[fileName])
							score[constraintType] += 1.0
						}
					}
					jumpLine = int(ifNode.EndPoint().Row) - index
					index = int(ifNode.EndPoint().Row)
					// log.Println("jump to ", index)
				}

				functionName := getFunctionName(findFunctionAtLine(tree, lineNum), input.SourceCode[fileName])
				// log.Println("New line in function ", functionName, " at ", lineNum)

				if arrutil.Contains(input.ImportantFunctions, functionName) {
					ImportantIncreaseCount += jumpLine + 1
				}
				AllIncreaseCount += jumpLine + 1

			}
		}
	}
	log.Println("Fuzzer", input.FuzzerName, "find", AllIncreaseCount, "increases in total,", ImportantIncreaseCount, "of them are important")
	// log.Printf("Fuzzer score calculated: %+v", score)
	return score
}

func NormalizeScore(score ConstraintScore) ConstraintScore {
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

func SelectFuzzerByScores(constraintGroup ConstraintGroup, fuzzerScores map[string]ConstraintScore) string {
	fuzzerName := ""

	if len(fuzzerScores) == 0 {
		return fuzzerName
	}

	type fuzzerScore struct {
		name       string
		dotProduct float64
	}

	var fuzzerScoresList []fuzzerScore

	for fuzzerNameKey, fuzzerConstraints := range fuzzerScores {
		dotProduct := 0.0

		for constraintName, groupScore := range constraintGroup.ConstraintScore {
			if fuzzerScoreVal, ok := fuzzerConstraints[constraintName]; ok {
				dotProduct += groupScore * fuzzerScoreVal
			}
		}

		fuzzerScoresList = append(fuzzerScoresList, fuzzerScore{name: fuzzerNameKey, dotProduct: dotProduct})
	}

	sumScores := 0.0
	for _, entry := range fuzzerScoresList {
		sumScores += entry.dotProduct
	}

	if sumScores > 0 {
		randomValue := float64(rand.Intn(1000)) / 1000.0
		cumulativeProbability := 0.0

		for _, entry := range fuzzerScoresList {
			probability := entry.dotProduct / sumScores
			cumulativeProbability += probability

			if randomValue <= cumulativeProbability {
				fuzzerName = entry.name
				break
			}
		}

		if fuzzerName == "" && len(fuzzerScoresList) > 0 {
			fuzzerName = fuzzerScoresList[0].name
		}
	} else if len(fuzzerScoresList) > 0 {
		randomIndex := rand.Intn(len(fuzzerScoresList))
		fuzzerName = fuzzerScoresList[randomIndex].name
	}

	return fuzzerName
}
