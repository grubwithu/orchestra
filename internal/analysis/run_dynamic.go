package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type FuncCov struct {
	// branches
	Count     int      `json:"count"`
	FileNames []string `json:"filenames"`
	// mcdc_records
	Name string `json:"name"`
	// regions
}

type ProgCovData struct {
	// Files
	Functions []FuncCov `json:"functions"`
	// totals
}

type ProgCovFile struct {
	Data    []ProgCovData `json:"data"`
	Type    string        `json:"type"`
	Version string        `json:"version"`
}

type LineCov struct {
	LineNumber uint32
	Count      uint32
	Code       []byte
}

type FileLineCov struct {
	File  string
	Lines []LineCov
}

func (plc *FileLineCov) GetOriginCode() []byte {
	code := []byte{}
	for _, line := range plc.Lines {
		code = append(code, line.Code...)
		code = append(code, '\n')
	}
	return code
}

func (flc *FileLineCov) ResetCov() {
	for i := range flc.Lines {
		flc.Lines[i].Count = 0
	}
}

func llvmLineCovPreprocess(output string) []FileLineCov {
	lines := strings.Split(output, "\n")
	progLineCov := []FileLineCov{}
	var curItem *FileLineCov
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if curItem != nil {
				progLineCov = append(progLineCov, *curItem)
			}
			curItem = &FileLineCov{
				File: strings.Split(line, ":")[0],
			}
			continue
		}
		if curItem == nil {
			log.Fatalf("line %s: curItem is nil", line)
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			// log.Printf("Warning: ignored one line %v in %v, %v", len(curItem.Lines), curItem.File, index+1)
			continue
		}
		lineNumber, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			// log.Printf("Warning: ignored one line %v in %v", len(curItem.Lines), curItem.File)
			continue
		}
		count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			count = 0
		}
		curItem.Lines = append(curItem.Lines, LineCov{
			LineNumber: uint32(lineNumber),
			Count:      uint32(count),
			Code:       []byte(parts[2]),
		})

	}

	// Add the last item if it exists and has lines
	if curItem != nil && len(curItem.Lines) > 0 {
		progLineCov = append(progLineCov, *curItem)
	}

	return progLineCov

}

func RunOnceForProfdata(workDir string, progPath string, corpusPath string) (cov int, profdataPath string, err error) {
	tempDir, err := os.MkdirTemp(workDir, "corpus")
	if err != nil {
		return 0, "", fmt.Errorf("failed to create corpus directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Transform programPath to absolute path
	progPath, err = filepath.Abs(progPath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get absolute path of program: %w", err)
	}

	// 2. Execute shell command
	cmd := exec.Command(progPath, tempDir, corpusPath, "-merge=1", "-rss_limit_mb=0")
	cmd.Dir = workDir
	cmd.Stdout = nil
	var outBuffer bytes.Buffer
	cmd.Stderr = &outBuffer

	if err := cmd.Run(); err != nil {
		return 0, "", fmt.Errorf("command execution failed: %w", err)
	}

	// There must be a line "MERGE-OUTER: 2 new files with 2321 new features added; 2114 new coverage edges"
	// Extracting the number before "new coverage edges"
	lines := strings.Split(outBuffer.String(), "\n")
	cov = 0
	for _, line := range lines {
		if strings.HasPrefix(line, "MERGE-OUTER:") {
			parts := strings.Split(line, ";")
			if len(parts) != 2 {
				continue
			}
			coverageEdges, err := strconv.Atoi(strings.TrimSpace(strings.Split(parts[1], "new coverage edges")[0]))
			if err != nil {
				continue
			}
			log.Println("Covearge Update: ", coverageEdges)
			cov = coverageEdges
		}
	}

	// 3. Find default.profraw in tempDir
	defaultProfrawPath := filepath.Join(workDir, "default.profraw")
	if _, err := os.Stat(defaultProfrawPath); os.IsNotExist(err) {
		return cov, "", fmt.Errorf("default.profraw not found in %s", workDir)
	}

	// 4. Running command to process default.profraw
	cmd = exec.Command("llvm-profdata", "merge", "-sparse", defaultProfrawPath, "-o", "merged_cov.profdata")
	cmd.Dir = workDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return cov, "", fmt.Errorf("command execution failed: %w", err)
	}

	return cov, filepath.Join(workDir, "merged_cov.profdata"), nil
}

func GetProgCov(workDir string, progPath string, profdataPath string) (ProgCovData, error) {

	cmd := exec.Command("llvm-cov", "export", "-instr-profile", profdataPath, "-object="+progPath)
	cmd.Dir = workDir
	var outBuffer bytes.Buffer // Create a buffer to capture stdout
	cmd.Stdout = &outBuffer
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return ProgCovData{}, fmt.Errorf("command execution failed: %w", err)
	}
	llvmCovExport := outBuffer.String()
	var programCoverageFile ProgCovFile
	err := json.Unmarshal([]byte(llvmCovExport), &programCoverageFile)
	if err != nil {
		return ProgCovData{}, fmt.Errorf("error parsing JSON: %w", err)
	}
	if len(programCoverageFile.Data) > 1 {
		log.Println("Attention: more than one ProgramCoverageData found in the JSON file")
	}

	return programCoverageFile.Data[0], nil
}

func GetLineCov(workDir string, progPath string, profdataPath string) ([]FileLineCov, error) {
	cmd := exec.Command("llvm-cov", "show", "--use-color=0", "-instr-profile", profdataPath, "-object="+progPath)
	cmd.Dir = workDir
	var outBuffer bytes.Buffer // Create a buffer to capture stdout
	cmd.Stdout = &outBuffer
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return []FileLineCov{}, fmt.Errorf("command execution failed: %w", err)
	}
	llvmCovShow := outBuffer.String()
	lineCoverage := llvmLineCovPreprocess(llvmCovShow)
	return lineCoverage, nil
}
