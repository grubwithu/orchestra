package analysis

import (
	"testing"

	"github.com/grubwithu/orchestra/internal/utils/cdf"
)

// TestGetConstraintGroups tests the GetConstraintGroups function
func TestGetConstraintGroups(t *testing.T) {
	// Create a simple call tree for testing with leaf nodes
	root := &CallTreeNode{
		FunctionProfile: &FunctionProfile{
			FunctionName:         "main",
			CyclomaticComplexity: 1,
		},
	}

	foo := &CallTreeNode{
		FunctionProfile: &FunctionProfile{
			FunctionName:         "foo",
			FunctionSourceFile:   "test.cc",
			CyclomaticComplexity: 5,
		},
		Parent: root,
	}

	bar := &CallTreeNode{
		FunctionProfile: &FunctionProfile{
			FunctionName:         "bar",
			FunctionSourceFile:   "test.cc",
			CyclomaticComplexity: 3,
		},
		Parent: root,
	}

	// Add children to make foo and bar leaf nodes
	root.Children = []*CallTreeNode{foo, bar}

	callTree := &CallTree{
		Root:                    root,
		Nodes:                   []*CallTreeNode{root, foo, bar},
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
	groups := GetConstraintGroups(InputGetConstraintGroups{
		CallTree:    callTree,
		ProgCovData: progCovData,
	})

	// Check that we got the expected number of groups (should equal number of leaf nodes)
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups (one per leaf node), got %d", len(groups))
	}

	// Check that groups are sorted by total importance
	if len(groups) > 1 && groups[0].TotalImportance < groups[1].TotalImportance {
		t.Errorf("Groups should be sorted by total importance in descending order")
	}

	// Check that paths are generated correctly
	for _, group := range groups {
		if len(group.Path) == 0 {
			t.Errorf("Expected non-empty path for group %s", group.LeafFunction)
		}
		// Path should start with root function
		if group.Path[0] != "main" {
			t.Errorf("Expected path to start with 'main', got %s", group.Path[0])
		}
		// Path should end with leaf function
		if group.Path[len(group.Path)-1] != group.LeafFunction {
			t.Errorf("Expected path to end with leaf function %s, got %s", group.LeafFunction, group.Path[len(group.Path)-1])
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
