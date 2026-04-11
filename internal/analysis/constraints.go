package analysis

import (
	"math"
	"math/rand"
	"slices"
	"sort"

	"github.com/google/uuid"
	"github.com/grubwithu/orchestra/internal/utils/cdf"
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
	W_Region     = 0.10
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

type InputGetConstraintGroups struct {
	CallTree         *CallTree
	ProgCovData      *ProgCovData
	AST              map[string]*sitter.Tree
	SourceCode       map[string][]byte
	FunctionProfiles []*FunctionProfile
	LineCovs         []FileLineCov
}

/**
 * GetConstraintGroups - get constraint groups from call tree leaf nodes.
 **/
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
		pathDetail := []*FunctionProfile{}
		node := leafNode
		for node != nil {
			path = append(path, node.FunctionProfile.FunctionName)
			pathDetail = append(pathDetail, node.FunctionProfile)
			node = node.Parent
		}
		slices.Reverse(path)
		slices.Reverse(pathDetail)

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
			PathDetail:      pathDetail,
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

type ConstraintGroup struct {
	GroupId         string             `json:"group_id"`
	Path            []string           `json:"path"`             // Path from root to leaf (function names for JSON output)
	PathDetail      []*FunctionProfile `json:"-"`                // Detailed path with FunctionProfile (internal use)
	LeafFunction    string             `json:"leaf_function"`    // The leaf function
	FileName        string             `json:"file_name"`        // File name of the leaf function
	TotalImportance float64            `json:"importance"`       // Total weighted score
	ConstraintScore ConstraintScore    `json:"constraint_score"` // Constraint preference score
}

// selectConstraintGroup selects a constraint group with weighted random selection
// Groups with higher TotalImportance have higher probability of being selected
func SelectConstraintGroup(groups []ConstraintGroup) *ConstraintGroup {
	if len(groups) == 0 {
		return nil
	}

	// Calculate weights based on TotalImportance
	// Use exponential weighting to give higher importance to top groups
	weights := make([]float64, len(groups))
	totalWeight := 0.0

	for i, group := range groups {
		// Weight = TotalImportance * (1 + (len(groups) - i) / len(groups))
		// This gives extra weight to groups that appear earlier in the sorted list
		positionBonus := 1.0 + float64(len(groups)-i)/float64(len(groups))
		weights[i] = group.TotalImportance * positionBonus
		totalWeight += weights[i]
	}

	// If all weights are 0, return a random group
	if totalWeight == 0 {
		return &groups[rand.Intn(len(groups))]
	}

	// Weighted random selection
	r := rand.Float64() * totalWeight
	cumulativeWeight := 0.0

	for i, weight := range weights {
		cumulativeWeight += weight
		if r <= cumulativeWeight {
			return &groups[i]
		}
	}

	// Fallback to last group
	return &groups[len(groups)-1]
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
