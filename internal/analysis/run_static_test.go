package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseDebugInfoFromFile tests the ParseDebugInfoFromFile function
func TestParseDebugInfoFromFile(t *testing.T) {
	// Create a temporary debug info file with sample content
	tempDir, err := os.MkdirTemp("", "hfc_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	debugInfoPath := filepath.Join(tempDir, "debug_info.txt")
	debugInfoContent := `--- Debug Information for Module 2.0 ---
Compile unit: DW_LANG_C_plus_plus_14 from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc

## Functions defined in module
Subprogram: btowc
 from /usr/include/wchar.h:285
 - Operand Type: Name: {  wint_t}Type:  from /usr/include/x86_64-linux-gnu/bits/types/wint_t.h:20DW_TAG_typedef,  wint_t
 - Operand Type: Name: {  int}Type:  DW_ATE_signed

Subprogram: testSwitch
 from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc:30 ('_ZN15TestControlFlow10testSwitchEi')
 - Operand Type: Name: {  int}Type:  DW_ATE_signed
 - Operand Type: Type: DW_TAG_pointer_type,  TestControlFlow
 - Operand Type: Name: {  int}Type:  DW_ATE_signed

## Global variables in module
Global variable: __ioinit from /usr/lib/gcc/x86_64-linux-gnu/12/../../../../include/c++/12/iostream:74 ('_ZStL8__ioinit')
Global variable:  from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc:46
Global variable:  from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc:49

## Types defined in module
Type: Name: {  Init} from /usr/lib/gcc/x86_64-linux-gnu/12/../../../../include/c++/12/bits/ios_base.h:635 DW_TAG_class_type Composite type
 (identifier: '_ZTSNSt8ios_base4InitE') - Elements: 0

Type: Name: {  ios_base} from /usr/lib/gcc/x86_64-linux-gnu/12/../../../../include/c++/12/bits/ios_base.h:229 DW_TAG_class_type Composite type
 - Elements: 0

Type: Name: {  char} DW_ATE_signed_char
Type: Name: {  mbstate_t} from /usr/include/x86_64-linux-gnu/bits/types/mbstate_t.h:6 DW_TAG_typedef
Type: Name: {  __mbstate_t} from /usr/include/x86_64-linux-gnu/bits/types/__mbstate_t.h:21 DW_TAG_typedef
Type: Name: {  __count} from /usr/include/x86_64-linux-gnu/bits/types/__mbstate_t.h:15 DW_TAG_member
`

	if err := os.WriteFile(debugInfoPath, []byte(debugInfoContent), 0644); err != nil {
		t.Fatalf("Failed to write debug info file: %v", err)
	}

	// Parse the debug info file
	debugInfo, err := ParseDebugInfoFromFile(debugInfoPath)
	if err != nil {
		t.Fatalf("ParseDebugInfoFromFile failed: %v", err)
	}

	// Test module info
	if debugInfo.ModuleInfo == "" {
		t.Error("ModuleInfo should not be empty")
	}

	// Test compile units
	if len(debugInfo.CompileUnits) != 1 {
		t.Errorf("Expected 1 compile unit, got %d", len(debugInfo.CompileUnits))
	}

	// Test functions
	expectedFunctions := map[string]bool{
		"btowc":                              true,
		"_ZN15TestControlFlow10testSwitchEi": true,
	}

	for expectedFunc := range expectedFunctions {
		if _, ok := debugInfo.Functions[expectedFunc]; !ok {
			t.Errorf("Expected function %s not found", expectedFunc)
		}
	}

	// Test function details
	if testSwitch, ok := debugInfo.Functions["_ZN15TestControlFlow10testSwitchEi"]; ok {
		if testSwitch.Name != "testSwitch" {
			t.Errorf("Expected function name 'testSwitch', got '%s'", testSwitch.Name)
		}
		if testSwitch.MangledName != "_ZN15TestControlFlow10testSwitchEi" {
			t.Errorf("Expected mangled name '_ZN15TestControlFlow10testSwitchEi', got '%s'", testSwitch.MangledName)
		}
		if testSwitch.SourceFile != "/home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc" {
			t.Errorf("Expected source file '/home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc', got '%s'", testSwitch.SourceFile)
		}
		if testSwitch.LineNumber != 30 {
			t.Errorf("Expected line number 30, got %d", testSwitch.LineNumber)
		}
		if len(testSwitch.OperandTypes) != 3 {
			t.Errorf("Expected 3 operand types, got %d", len(testSwitch.OperandTypes))
		}
	}

	// Test global variables and types raw data
	if debugInfo.GlobalVariablesRaw == "" {
		t.Error("GlobalVariablesRaw should not be empty")
	}
	if debugInfo.TypesRaw == "" {
		t.Error("TypesRaw should not be empty")
	}
}

// TestParseDebugInfoFromFileWithMultipleCompileUnits tests parsing with multiple compile units
func TestParseDebugInfoFromFileWithMultipleCompileUnits(t *testing.T) {
	// Create a temporary debug info file with multiple compile units
	tempDir, err := os.MkdirTemp("", "hfc_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	debugInfoPath := filepath.Join(tempDir, "debug_info_multi_units.txt")
	debugInfoContent := `--- Debug Information for Module 2.0 ---
Compile unit: DW_LANG_C_plus_plus_14 from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc
Compile unit: DW_LANG_C from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/utils.c

## Functions defined in module
Subprogram: btowc
 from /usr/include/wchar.h:285

Subprogram: testSwitch
 from /home/grub/workspace/parallel_fuzz/hfc-introspector/test/main.cc:30 ('_ZN15TestControlFlow10testSwitchEi')
`

	if err := os.WriteFile(debugInfoPath, []byte(debugInfoContent), 0644); err != nil {
		t.Fatalf("Failed to write debug info file: %v", err)
	}

	// Parse the debug info file
	debugInfo, err := ParseDebugInfoFromFile(debugInfoPath)
	if err != nil {
		t.Fatalf("ParseDebugInfoFromFile failed: %v", err)
	}

	// Test compile units
	if len(debugInfo.CompileUnits) != 2 {
		t.Errorf("Expected 2 compile units, got %d", len(debugInfo.CompileUnits))
	}
}

// TestParseDebugInfoFromFileEdgeCases tests edge cases
func TestParseDebugInfoFromFileEdgeCases(t *testing.T) {
	// Test with non-existent file
	_, err := ParseDebugInfoFromFile("non_existent_file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with empty file
	tempDir, err := os.MkdirTemp("", "hfc_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	emptyFilePath := filepath.Join(tempDir, "empty.txt")
	if err := os.WriteFile(emptyFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	debugInfo, err := ParseDebugInfoFromFile(emptyFilePath)
	if err != nil {
		t.Fatalf("ParseDebugInfoFromFile failed for empty file: %v", err)
	}

	if debugInfo.ModuleInfo != "" {
		t.Error("ModuleInfo should be empty for empty file")
	}
	if len(debugInfo.CompileUnits) != 0 {
		t.Errorf("Expected 0 compile units for empty file, got %d", len(debugInfo.CompileUnits))
	}
	if len(debugInfo.Functions) != 0 {
		t.Errorf("Expected 0 functions for empty file, got %d", len(debugInfo.Functions))
	}
}
