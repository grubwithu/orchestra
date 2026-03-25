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
	"github.com/grubwithu/hfc/internal/utils/cdf"
	sitter "github.com/smacker/go-tree-sitter"
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
)

type ConstraintGroup struct {
	GroupId         string          `json:"group_id"`
	Path            []string        `json:"path"`             // Path from root to leaf
	LeafFunction    string          `json:"leaf_function"`    // The leaf function
	FileName        string          `json:"file_name"`        // File name of the leaf function
	TotalImportance float64         `json:"importance"`       // Total weighted score
	ConstraintScore ConstraintScore `json:"constraint_score"` // Constraint preference score
}

// GetConstraintGroups generates constraint groups based on call tree leaf nodes
// Each constraint group represents a path from root to leaf in the call tree
func GetConstraintGroups(callTree *CallTree, progCovData *ProgCovData, ast map[string]*sitter.Tree, sourceCode map[string][]byte, functionProfiles []*FunctionProfile) []ConstraintGroup {
	// create a map of function coverage and region coverage
	MAX_NUM_CHILDREN := len(callTree.Nodes) - 1
	MIN_HITS := math.MaxInt
	MAX_HITS := math.MinInt
	covCdf := cdf.NewCDF()
	coverageMap := make(map[string]int)
	regionCoverageMap := make(map[string]float64)
	for _, funcCoverage := range progCovData.Functions {
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
	for _, node := range callTree.Nodes {
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
			depthWeight := math.Sqrt(float64(node.GetUpperDepth()) / float64(callTree.MaxDepth))
			branchWeight := float64(node.CountDescendantNode()) / float64(MAX_NUM_CHILDREN)
			complexityWeight := float64(node.FunctionProfile.CyclomaticComplexity) / float64(callTree.MaxCyclomaticComplexity)
			// Add region coverage weight (inverted, so lower coverage gives higher weight)
			regionCoverageWeight := 1.0 - regionCoverageMap[node.FunctionProfile.FunctionName]
			nodeScore := W_HitFreq*hitFreqWeight + W_Rarity*rarityWeight + W_Depth*depthWeight + W_Branch*branchWeight + W_Complexity*complexityWeight + 0.1*regionCoverageWeight

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
		group.ConstraintScore = calculateScore(group, ast, sourceCode, functionProfiles)
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

func calculateScore(group ConstraintGroup, ast map[string]*sitter.Tree, sourceCode map[string][]byte, functionProfiles []*FunctionProfile) ConstraintScore {
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
	for _, profile := range functionProfiles {
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
		tree, hasAST := ast[profile.FunctionSourceFile]
		if !hasAST {
			continue
		}

		// Find function node
		var funcNode *sitter.Node
		if strings.HasPrefix(funcName, "_Z") {
			funcNode = findFunctionAtLine(tree, uint32(profile.FunctionLinenumber))
		} else {
			funcNode = findFunctionWithQuery(tree, sourceCode[profile.FunctionSourceFile], funcName)
		}

		if funcNode == nil {
			continue
		}

		// Analyze function
		scores := analyzeFunction(funcNode, sourceCode[profile.FunctionSourceFile])

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
