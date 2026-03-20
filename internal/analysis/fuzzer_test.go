package analysis

import (
	"context"
	"fmt"
	"testing"

	"github.com/gookit/goutil/arrutil"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

// TestConstraintScoreCopy tests the Copy method of ConstraintScore
func TestConstraintScoreCopy(t *testing.T) {
	original := ConstraintScore{
		CT_VALUE_COMPARISON:     1.0,
		CT_BITWISE_OPERATION:    2.0,
		CT_STRING_MATCH:         3.0,
		CT_ARITHMETIC_OPERATION: 4.0,
		CT_COMPOUND_OPERATION:   5.0,
	}

	copy := original.Copy()

	// Check that all values are copied
	for key, value := range original {
		if copy[key] != value {
			t.Errorf("Expected copy[%s] to be %f, got %f", key, value, copy[key])
		}
	}

	// Check that copy is a different map
	copy[CT_VALUE_COMPARISON] = 10.0
	if original[CT_VALUE_COMPARISON] == 10.0 {
		t.Error("Original should not be modified when copy is modified")
	}
}

// TestUpdateFuzzerScore tests the UpdateFuzzerScore function
func TestUpdateFuzzerScore(t *testing.T) {
	cur := ConstraintScore{
		CT_VALUE_COMPARISON:  1.0,
		CT_BITWISE_OPERATION: 2.0,
	}

	prev := ConstraintScore{
		CT_VALUE_COMPARISON: 5.0,
		CT_STRING_MATCH:     3.0,
	}

	result := UpdateFuzzerScore(cur, prev)

	// Check that existing values are updated
	expectedValueComparison := 1.0 + 5.0/10.0 // 1.5
	if result[CT_VALUE_COMPARISON] != expectedValueComparison {
		t.Errorf("Expected CT_VALUE_COMPARISON to be %f, got %f", expectedValueComparison, result[CT_VALUE_COMPARISON])
	}

	// Check that new values are added
	if result[CT_BITWISE_OPERATION] != 2.0 {
		t.Errorf("Expected CT_BITWISE_OPERATION to be 2.0, got %f", result[CT_BITWISE_OPERATION])
	}

	// Check that existing values not in cur are preserved
	if result[CT_STRING_MATCH] != 3.0 {
		t.Errorf("Expected CT_STRING_MATCH to be 3.0, got %f", result[CT_STRING_MATCH])
	}
}

// TestUpdateFuzzerScoreWithNilPrev tests UpdateFuzzerScore with nil prev
func TestUpdateFuzzerScoreWithNilPrev(t *testing.T) {
	cur := ConstraintScore{
		CT_VALUE_COMPARISON:  1.0,
		CT_BITWISE_OPERATION: 2.0,
	}

	result := UpdateFuzzerScore(cur, nil)

	// Check that all values are copied
	for key, value := range cur {
		if result[key] != value {
			t.Errorf("Expected result[%s] to be %f, got %f", key, value, result[key])
		}
	}
}

// TestAnalyzeIfClause tests the analyzeIfClause function with actual AST parsing
func TestAnalyzeIfClause(t *testing.T) {
	// C++ code with various if conditions
	sourceCode := []byte(`
#include <cstring>

void testFunction(int x, int y, const char* s) {
    // String match
    if (strcmp(s, "test") == 0) {
        // Do something
    }
    
    // Arithmetic operation
    if (x + y > 5) {
        // Do something
    }
    
    // Bitwise operation
    if (x & 1 == 0) {
        // Do something
    }
    
    // Value comparison
    if (x == 5) {
        // Do something
    }
    
    // Compound operation
    if (x && y) {
        // Do something
    }
}
`)

	// Parse the code using tree-sitter
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cpp.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	// Print all if statements with their line numbers for debugging
	root := tree.RootNode()
	var findIfStatements func(*sitter.Node)
	findIfStatements = func(node *sitter.Node) {
		if node.Type() == "if_statement" {
			startLine := node.StartPoint().Row + 1
			endLine := node.EndPoint().Row + 1
			t.Logf("Found if statement at lines %d-%d", startLine, endLine)
			// Print the condition
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "parenthesized_expression" {
					conditionCode := child.Content(sourceCode)
					t.Logf("  Condition: %s", conditionCode)
					break
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findIfStatements(node.Child(i))
		}
	}
	findIfStatements(root)

	// Test cases with expected constraint types for each if statement
	testCases := []struct {
		line     uint32
		expected ConstraintType
	}{
		{6, CT_STRING_MATCH},          // strcmp
		{11, CT_ARITHMETIC_OPERATION}, // x + y > 5
		{16, CT_BITWISE_OPERATION},    // x & 1 == 0
		{21, CT_VALUE_COMPARISON},     // x == 5
		{26, CT_COMPOUND_OPERATION},   // x && y
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Line %d", tc.line), func(t *testing.T) {
			// Find if statement at the specified line
			ifNode := findIfStatementAtLine(tree, tc.line)
			if ifNode == nil {
				t.Fatalf("Failed to find if statement at line %d", tc.line)
			}

			// Find the condition node
			condition := findIfCondition(ifNode)
			if condition == nil {
				t.Fatalf("Failed to find condition for if statement at line %d", tc.line)
			}

			// Analyze the condition
			result := analyzeIfClause(condition, sourceCode)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestCalculateFuzzerScoreEdgeCases tests edge cases for CalculateFuzzerScore
func TestCalculateFuzzerScoreEdgeCases(t *testing.T) {
	// Test with empty inputs
	score := CalculateFuzzerScore("test", []FileLineCov{}, []FileLineCov{}, nil, nil, nil)
	// Check that all scores are 0
	expectedScore := ConstraintScore{
		CT_VALUE_COMPARISON:     0,
		CT_BITWISE_OPERATION:    0,
		CT_STRING_MATCH:         0,
		CT_ARITHMETIC_OPERATION: 0,
		CT_COMPOUND_OPERATION:   0,
	}
	for key, value := range expectedScore {
		if score[key] != value {
			t.Errorf("Expected score[%s] to be %f, got %f", key, value, score[key])
		}
	}

	// Test with nil AST
	score = CalculateFuzzerScore("test", []FileLineCov{{File: "test.cc"}}, []FileLineCov{{File: "test.cc"}}, nil, nil, nil)
	// Check that all scores are 0
	for key, value := range expectedScore {
		if score[key] != value {
			t.Errorf("Expected score[%s] to be %f, got %f", key, value, score[key])
		}
	}
}

// TestArrutilContains tests the arrutil.Contains function (used in CalculateFuzzerScore)
func TestArrutilContains(t *testing.T) {
	list := []string{"foo", "bar", "baz"}

	if !arrutil.Contains(list, "foo") {
		t.Error("Expected arrutil.Contains to find 'foo' in list")
	}

	if arrutil.Contains(list, "qux") {
		t.Error("Expected arrutil.Contains to not find 'qux' in list")
	}
}
