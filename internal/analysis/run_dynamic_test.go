package analysis

import (
	"testing"
)

// TestLLVMLineCovPreprocess tests the llvmLineCovPreprocess function
func TestLLVMLineCovPreprocess(t *testing.T) {
	// Mock llvm-cov output
	mockOutput := `/path/to/file1.cc:
1|0|int main() {
2|1|  return 0;
3|0|}

/path/to/file2.cc:
1|0|void foo() {
2|1|  int x = 5;
3|1|  bar(x);
4|0|}
`

	result := llvmLineCovPreprocess(mockOutput)

	// Check number of files
	if len(result) != 2 {
		t.Errorf("Expected 2 files, got %d", len(result))
	}

	// Check first file
	if result[0].File != "/path/to/file1.cc" {
		t.Errorf("Expected file path /path/to/file1.cc, got %s", result[0].File)
	}
	if len(result[0].Lines) != 3 {
		t.Errorf("Expected 3 lines in file1, got %d", len(result[0].Lines))
	}
	if result[0].Lines[0].LineNumber != 1 {
		t.Errorf("Expected line 1, got %d", result[0].Lines[0].LineNumber)
	}
	if result[0].Lines[0].Count != 0 {
		t.Errorf("Expected count 0 for line 1, got %d", result[0].Lines[0].Count)
	}
	if string(result[0].Lines[0].Code) != "int main() {" {
		t.Errorf("Expected code 'int main() {', got '%s'", string(result[0].Lines[0].Code))
	}

	// Check second file
	if result[1].File != "/path/to/file2.cc" {
		t.Errorf("Expected file path /path/to/file2.cc, got %s", result[1].File)
	}
	if len(result[1].Lines) != 4 {
		t.Errorf("Expected 4 lines in file2, got %d", len(result[1].Lines))
	}
	if result[1].Lines[1].LineNumber != 2 {
		t.Errorf("Expected line 2, got %d", result[1].Lines[1].LineNumber)
	}
	if result[1].Lines[1].Count != 1 {
		t.Errorf("Expected count 1 for line 2, got %d", result[1].Lines[1].Count)
	}
	if string(result[1].Lines[1].Code) != "  int x = 5;" {
		t.Errorf("Expected code '  int x = 5;', got '%s'", string(result[1].Lines[1].Code))
	}
}

// TestFileLineCovGetOriginCode tests the GetOriginCode method
func TestFileLineCovGetOriginCode(t *testing.T) {
	flc := FileLineCov{
		File: "test.cc",
		Lines: []LineCov{
			{LineNumber: 1, Count: 0, Code: []byte("int main() {")},
			{LineNumber: 2, Count: 1, Code: []byte("  return 0;")},
			{LineNumber: 3, Count: 0, Code: []byte("}")},
		},
	}

	expectedCode := "int main() {\n  return 0;\n}\n"
	actualCode := string(flc.GetOriginCode())

	if actualCode != expectedCode {
		t.Errorf("Expected code %q, got %q", expectedCode, actualCode)
	}
}

// TestFileLineCovResetCov tests the ResetCov method
func TestFileLineCovResetCov(t *testing.T) {
	flc := FileLineCov{
		File: "test.cc",
		Lines: []LineCov{
			{LineNumber: 1, Count: 5, Code: []byte("int main() {")},
			{LineNumber: 2, Count: 3, Code: []byte("  return 0;")},
			{LineNumber: 3, Count: 1, Code: []byte("}")},
		},
	}

	flc.ResetCov()

	for i, line := range flc.Lines {
		if line.Count != 0 {
			t.Errorf("Expected count 0 for line %d, got %d", i+1, line.Count)
		}
	}
}

// TestFileLineCovEdgeCases tests edge cases for FileLineCov
func TestFileLineCovEdgeCases(t *testing.T) {
	// Test empty Lines
	flc := FileLineCov{
		File:  "test.cc",
		Lines: []LineCov{},
	}

	// Test GetOriginCode with empty Lines
	code := flc.GetOriginCode()
	if len(code) != 0 {
		t.Errorf("Expected empty code, got %q", string(code))
	}

	// Test ResetCov with empty Lines
	flc.ResetCov() // Should not panic
}

// TestLLVMLineCovPreprocessEdgeCases tests edge cases for llvmLineCovPreprocess
func TestLLVMLineCovPreprocessEdgeCases(t *testing.T) {
	// Test empty input
	result := llvmLineCovPreprocess("")
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d files", len(result))
	}

	// Test input with only file path
	result = llvmLineCovPreprocess("/path/to/file.cc:\n")
	if len(result) != 0 {
		t.Errorf("Expected empty result for file with no lines, got %d files", len(result))
	}

	// Test input with malformed lines
	mockOutput := `/path/to/file.cc:
1|0|int main() {
malformed line
3|1|  return 0;
`
	result = llvmLineCovPreprocess(mockOutput)
	if len(result) != 1 {
		t.Errorf("Expected 1 file, got %d", len(result))
	}
	if len(result[0].Lines) != 2 {
		t.Errorf("Expected 2 valid lines, got %d", len(result[0].Lines))
	}
}
