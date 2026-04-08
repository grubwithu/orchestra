package analysis

/**
 * GetConstraintGroups - get constraint groups from call tree leaf nodes.
 **/

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/grubwithu/orchestra/internal/utils/cdf"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

var (
	IMPORTANCE_SCORE_THRESHOLD = 0.5
	MAX_CONSTRAINTS            = math.MaxInt
)
var (
	W_HitFreq    = 0.10
	W_Rarity     = 0.20
	W_Depth      = 0.20
	W_Branch     = 0.35
	W_Complexity = 0.15
	W_Region     = 0.10
)

type ConstraintGroup struct {
	GroupId         string          `json:"group_id"`
	Path            []string        `json:"path"`             // Path from root to leaf
	LeafFunction    string          `json:"leaf_function"`    // The leaf function
	FileName        string          `json:"file_name"`        // File name of the leaf function
	TotalImportance float64         `json:"importance"`       // Total weighted score
	ConstraintScore ConstraintScore `json:"constraint_score"` // Constraint preference score
}

type InputGetConstraintGroups struct {
	CallTree         *CallTree
	ProgCovData      *ProgCovData
	AST              map[string]*sitter.Tree
	SourceCode       map[string][]byte
	FunctionProfiles []*FunctionProfile
	LineCovs         []FileLineCov
}

// GetConstraintGroups generates constraint groups based on call tree leaf nodes
// Each constraint group represents a path from root to leaf in the call tree
func GetConstraintGroups(input InputGetConstraintGroups) []ConstraintGroup {
	// create a map of function coverage and region coverage
	MAX_NUM_CHILDREN := len(input.CallTree.Nodes) - 1
	MIN_HITS := math.MaxInt
	MAX_HITS := math.MinInt
	covCdf := cdf.NewCDF()
	coverageMap := make(map[string]int)
	regionCoverageMap := make(map[string]float64)
	for _, funcCoverage := range input.ProgCovData.Functions {
		coverageMap[funcCoverage.Name] = funcCoverage.Count
		covCdf.Add(float64(funcCoverage.Count))
		if funcCoverage.Count < MIN_HITS {
			MIN_HITS = funcCoverage.Count
		}
		if funcCoverage.Count > MAX_HITS {
			MAX_HITS = funcCoverage.Count
		}

		// Calculate region coverage ratio
		totalRegions := len(funcCoverage.Regions)
		executedRegions := 0
		for _, region := range funcCoverage.Regions {
			if len(region) > REGION_EXEC_CNT && region[REGION_EXEC_CNT] > 0 {
				executedRegions++
			}
		}
		if totalRegions > 0 {
			regionCoverageMap[funcCoverage.Name] = float64(executedRegions) / float64(totalRegions)
		} else {
			regionCoverageMap[funcCoverage.Name] = 0
		}
	}

	// Find all leaf nodes in the call tree
	leafNodes := []*CallTreeNode{}
	for _, node := range input.CallTree.Nodes {
		if node.FunctionProfile != nil && len(node.Children) == 0 {
			leafNodes = append(leafNodes, node)
		}
	}

	// Generate constraint groups from leaf nodes
	groups := []ConstraintGroup{}
	for _, leafNode := range leafNodes {
		// Generate path from root to leaf
		path := []string{}
		node := leafNode
		for node != nil {
			path = append(path, node.FunctionProfile.FunctionName)
			node = node.Parent
		}
		slices.Reverse(path)

		// Calculate weighted score for this path
		totalScore := 0.0
		node = leafNode
		depth := 0
		// Calculate score for each node in the path with weight
		for node != nil {
			if node.FunctionProfile == nil || coverageMap[node.FunctionProfile.FunctionName] == 0 {
				node = node.Parent
				depth++
				continue
			}

			// Calculate individual node score
			hitFreqWeight := (float64(coverageMap[node.FunctionProfile.FunctionName]) - float64(MIN_HITS)) / float64(MAX_HITS-MIN_HITS)
			rarityWeight := 1.0 - covCdf.GetCDFValue(float64(coverageMap[node.FunctionProfile.FunctionName]))
			depthWeight := math.Sqrt(float64(node.GetUpperDepth()) / float64(input.CallTree.MaxDepth))
			branchWeight := float64(node.CountDescendantNode()) / float64(MAX_NUM_CHILDREN)
			complexityWeight := float64(node.FunctionProfile.CyclomaticComplexity) / float64(input.CallTree.MaxCyclomaticComplexity)
			// Add region coverage weight (inverted, so lower coverage gives higher weight)
			regionCoverageWeight := 1.0 - regionCoverageMap[node.FunctionProfile.FunctionName]
			nodeScore := W_HitFreq*hitFreqWeight + W_Rarity*rarityWeight + W_Depth*depthWeight + W_Branch*branchWeight + W_Complexity*complexityWeight + W_Region*regionCoverageWeight

			totalScore += nodeScore

			node = node.Parent
			depth++
		}

		// Create constraint group
		group := ConstraintGroup{
			GroupId:         uuid.New().String(),
			Path:            path,
			LeafFunction:    leafNode.FunctionProfile.FunctionName,
			FileName:        leafNode.FunctionProfile.FunctionSourceFile,
			TotalImportance: totalScore,
		}

		// Calculate constraint preference score
		group.ConstraintScore = calculateScore(group, input)
		groups = append(groups, group)
	}

	// Sort groups by total importance
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalImportance > groups[j].TotalImportance
	})

	// Limit to MAX_CONSTRAINTS
	cut := MAX_CONSTRAINTS
	if len(groups) < MAX_CONSTRAINTS {
		cut = len(groups)
	}

	return groups[:cut]
}

func calculateScore(group ConstraintGroup, input InputGetConstraintGroups) ConstraintScore {
	// Initialize total scores
	totalScores := ConstraintScore{
		CT_VALUE_COMPARISON:     0,
		CT_BITWISE_OPERATION:    0,
		CT_STRING_MATCH:         0,
		CT_ARITHMETIC_OPERATION: 0,
		CT_COMPOUND_OPERATION:   0,
	}

	// Create function profile map for quick lookup
	functionProfileMap := make(map[string]*FunctionProfile)
	for _, profile := range input.FunctionProfiles {
		functionProfileMap[profile.FunctionName] = profile
	}

	// Calculate score for each function in the path
	for i, funcName := range group.Path {
		// Get function profile
		profile, ok := functionProfileMap[funcName]
		if !ok {
			continue
		}

		// Check if AST exists for this file
		tree, hasAST := input.AST[profile.FunctionSourceFile]
		if !hasAST {
			continue
		}

		// Find function node
		var funcNode *sitter.Node
		if strings.HasPrefix(funcName, "_Z") {
			funcNode = findFunctionAtLine(tree, uint32(profile.FunctionLinenumber))
		} else {
			funcNode = findFunctionWithQuery(tree, input.SourceCode[profile.FunctionSourceFile], funcName)
		}

		if funcNode == nil {
			continue
		}

		var lineCov *FileLineCov
		for _, line := range input.LineCovs {
			if line.File == profile.FunctionSourceFile {
				lineCov = &line
				break
			}
		}

		// Analyze function
		scores := analyzeFunction(funcNode, input.SourceCode[profile.FunctionSourceFile], lineCov)

		// Apply weight: closer to leaf = higher weight
		// Weight ranges from 0.5 (root) to 1.5 (leaf)
		weight := 0.5 + (float64(i) / float64(len(group.Path)-1))

		// Add weighted scores to total
		for constraintType, score := range scores {
			totalScores[constraintType] += score * weight
		}
	}

	// Normalize the scores
	maxScore := 0.0
	for _, score := range totalScores {
		if score > maxScore {
			maxScore = score
		}
	}

	if maxScore > 0 {
		for constraintType, score := range totalScores {
			totalScores[constraintType] = score / maxScore
		}
	} else {
		// If all scores are 0, set them to 0 (avoid NaN)
		for constraintType := range totalScores {
			totalScores[constraintType] = 0
		}
	}

	return totalScores
}

func analyzeFunction(funcNode *sitter.Node, sourceCode []byte, lineCov *FileLineCov) ConstraintScore {
	if funcNode == nil {
		return ConstraintScore{}
	}

	var maxLineHit uint32
	if lineCov != nil {
		for i := funcNode.StartPoint().Row; i <= funcNode.EndPoint().Row; i++ {
			if lineCov.Lines[i].Count > maxLineHit {
				maxLineHit = lineCov.Lines[i].Count
			}
		}
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

					var scale float64 = 1.0
					if lineCov != nil {
						var avgHit uint32
						for row := capture.Node.StartPoint().Row; row <= capture.Node.EndPoint().Row; row++ {
							avgHit += lineCov.Lines[row].Count
						}
						avgHit /= capture.Node.EndPoint().Row - capture.Node.StartPoint().Row + 1
						scale = (1 - float64(avgHit)/float64(maxLineHit)) + 0.5 // Range 0.5-1.5, low hit gives higher score
					}

					score[constraintType] += 1.0 * scale
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
