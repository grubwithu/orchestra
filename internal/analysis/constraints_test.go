package analysis

import (
	"testing"
	"github.com/grubwithu/hfc/internal/utils/cdf"
)

// TestIdentifyImportantConstraints tests the IdentifyImportantConstraints function
func TestIdentifyImportantConstraints(t *testing.T) {
	// Create a simple call tree for testing
	callTree := &CallTree{
		Root: &CallTreeNode{
			FunctionProfile: &FunctionProfile{
				FunctionName:         "main",
				CyclomaticComplexity: 1,
			},
		},
		Nodes: []*CallTreeNode{
			{
				FunctionProfile: &FunctionProfile{
					FunctionName:         "main",
					CyclomaticComplexity: 1,
				},
			},
			{
				FunctionProfile: &FunctionProfile{
					FunctionName:         "foo",
					CyclomaticComplexity: 5,
				},
				Parent: nil,
			},
			{
				FunctionProfile: &FunctionProfile{
					FunctionName:         "bar",
					CyclomaticComplexity: 3,
				},
				Parent: nil,
			},
		},
		MaxDepth:                2,
		MaxCyclomaticComplexity: 5,
	}

	// Create program coverage data
	progCovData := &ProgCovData{
		Functions: []FuncCov{
			{Name: "main", Count: 100},
			{Name: "foo", Count: 10},
			{Name: "bar", Count: 50},
		},
	}

	// Test the function
	constraints := IdentifyImportantConstraints(callTree, progCovData)

	// Check that we got some constraints
	if len(constraints) == 0 {
		t.Error("Expected at least one constraint, got none")
	}

	// Check that constraints are sorted by importance score
	for i := 1; i < len(constraints); i++ {
		if constraints[i-1].ImportanceScore < constraints[i].ImportanceScore {
			t.Errorf("Constraints should be sorted by importance score in descending order")
			break
		}
	}
}

// TestGroupConstraintsByFunction tests the GroupConstraintsByFunction function
func TestGroupConstraintsByFunction(t *testing.T) {
	// Create some mock constraints
	constraints := []ImportantConstraint{
		{
			CallTreeNode: &CallTreeNode{
				FunctionProfile: &FunctionProfile{
					FunctionName:         "foo",
					FunctionSourceFile:    "test.cc",
					CyclomaticComplexity: 5,
				},
				Parent: &CallTreeNode{
					FunctionProfile: &FunctionProfile{
						FunctionName: "main",
					},
				},
			},
			ImportanceScore: 0.8,
		},
		{
			CallTreeNode: &CallTreeNode{
				FunctionProfile: &FunctionProfile{
					FunctionName:         "foo",
					FunctionSourceFile:    "test.cc",
					CyclomaticComplexity: 5,
				},
				Parent: &CallTreeNode{
					FunctionProfile: &FunctionProfile{
						FunctionName: "main",
					},
				},
			},
			ImportanceScore: 0.6,
		},
		{
			CallTreeNode: &CallTreeNode{
				FunctionProfile: &FunctionProfile{
					FunctionName:         "bar",
					FunctionSourceFile:    "test.cc",
					CyclomaticComplexity: 3,
				},
				Parent: &CallTreeNode{
					FunctionProfile: &FunctionProfile{
						FunctionName: "main",
					},
				},
			},
			ImportanceScore: 0.7,
		},
	}

	// Create program coverage data
	progCovData := &ProgCovData{
		Functions: []FuncCov{
			{Name: "main", Count: 100},
			{Name: "foo", Count: 10},
			{Name: "bar", Count: 50},
		},
	}

	// Test the function
	groups := GroupConstraintsByFunction(constraints, progCovData, nil, nil, nil)

	// Check that we got the expected number of groups
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	// Check that groups are sorted by total importance
	if len(groups) > 1 && groups[0].TotalImportance < groups[1].TotalImportance {
		t.Errorf("Groups should be sorted by total importance in descending order")
	}

	// Check that paths are generated
	for _, group := range groups {
		if len(group.Paths) == 0 {
			t.Errorf("Expected at least one path for group %s", group.MainFunction)
		}
	}
}

// TestCDFIntegration tests the CDF integration
func TestCDFIntegration(t *testing.T) {
	// Create a CDF and add some values
	covCdf := cdf.NewCDF()
	covCdf.Add(10.0)
	covCdf.Add(20.0)
	covCdf.Add(30.0)

	// Test CDF values
	if val := covCdf.GetCDFValue(10.0); val != 1.0/3.0 {
		t.Errorf("Expected CDF value 0.333 for 10.0, got %f", val)
	}

	if val := covCdf.GetCDFValue(20.0); val != 2.0/3.0 {
		t.Errorf("Expected CDF value 0.666 for 20.0, got %f", val)
	}

	if val := covCdf.GetCDFValue(30.0); val != 1.0 {
		t.Errorf("Expected CDF value 1.0 for 30.0, got %f", val)
	}
}
