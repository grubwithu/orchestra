package webcore

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gookit/goutil/maputil"
	"github.com/grubwithu/hfc/internal/analysis"
)

func generateTaskID() (TaskID, error) {
	return TaskID(uuid.New().String()), nil
}

type TaskID string

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type CorpusReportReqBody struct {
	Fuzzer   string   `json:"fuzzer"`
	Identity string   `json:"identity"`
	Corpus   []string `json:"corpus"`
}

func (s *Server) processCorpus(taskID TaskID, fuzzer string, corpus string) {

	workDir, err := os.MkdirTemp("", "hfc_work_")
	if err != nil {
		log.Printf("Error creating temporary work directory: %v\n", err)
		return
	}
	defer os.RemoveAll(workDir)
	cov, profdataPath, err := analysis.RunOnceForProfdata(workDir, *s.Executable, corpus)
	if err != nil {
		log.Printf("Error running analysis on corpus item %s for task %s: %v\n", corpus, taskID, err)
		return
	}
	progCovData, err := analysis.GetProgCov(workDir, *s.Executable, profdataPath)
	if err != nil {
		log.Printf("Error running analysis on corpus item %s for task %s: %v\n", corpus, taskID, err)
	}

	s.FuzzerCovsMutex.Lock()
	// update fuzzer covs
	if _, ok := s.FuzzerCovs[fuzzer]; !ok {
		s.FuzzerCovs[fuzzer] = []int{s.FuzzerCovs["__init__"][0], cov}
	} else {
		s.FuzzerCovs[fuzzer] = append(s.FuzzerCovs[fuzzer], cov)
	}

	s.FuzzerCovsMutex.Unlock()

	s.GlobalCovMutex.Lock()
	if s.GlobalCov == nil {
		s.GlobalCov = &progCovData
	} else {
		// s.GlobalCov.CorpusCount += coverage.CorpusCount
		for _, funcCoverage := range progCovData.Functions {
			existing := false
			for i := range s.GlobalCov.Functions {
				if s.GlobalCov.Functions[i].Name == funcCoverage.Name {
					s.GlobalCov.Functions[i].Count += funcCoverage.Count
					existing = true
					break
				}
			}
			if !existing {
				s.GlobalCov.Functions = append(s.GlobalCov.Functions, funcCoverage)
			}
		}
	}

	constrains := analysis.IdentifyImportantConstraints(s.CallTree, s.GlobalCov)
	groups := analysis.GroupConstraintsByFunction(constrains, s.GlobalCov, s.AST, s.SourceCode)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalImportance > groups[j].TotalImportance
	})

	s.GlobalCovMutex.Unlock()

	s.ConstraintGroupsMutex.Lock()
	functions := make(map[string]int)
	for index, group := range s.ConstraintGroups {
		functions[group.MainFunction] = index
	}
	for _, group := range groups {
		function := group.MainFunction
		if _, ok := functions[function]; ok {
			s.ConstraintGroups[functions[function]] = group
		} else {
			s.ConstraintGroups = append(s.ConstraintGroups, group)
		}
	}
	sort.Slice(s.ConstraintGroups, func(i, j int) bool {
		return s.ConstraintGroups[i].TotalImportance > s.ConstraintGroups[j].TotalImportance
	})
	s.ConstraintGroupsMutex.Unlock()

	lineCov, err := analysis.GetLineCov(workDir, *s.Executable, profdataPath)
	if err != nil {
		log.Printf("Error running analysis on corpus item %s for task %s: %v\n", corpus, taskID, err)
		return
	}

	s.FileLineCovsMutex.Lock()
	score := analysis.CalculateFuzzerScore(fuzzer, lineCov, s.FileLineCovs, s.AST, s.SourceCode, maputil.Keys(functions))
	s.FileLineCovs = lineCov
	s.FileLineCovsMutex.Unlock()

	s.FuzzerScoresMutex.Lock()
	s.FuzzerScores[fuzzer] = analysis.UpdateFuzzerScore(score, s.FuzzerScores[fuzzer])
	s.FuzzerScoresMutex.Unlock()

	log.Printf("Processed corpus item %s for task %s.\n", corpus, taskID)

}

type ReportCorpusResponse struct {
	TaskID TaskID `json:"task_id"`
}

func (s *Server) handleReportCorpus(c *gin.Context) {
	var report CorpusReportReqBody

	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(report.Corpus) == 0 {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Corpus is empty",
		})
		return
	}

	// report.Corpus[0] is the path to the corpus file, print the tree of the directory
	// print the tree of the directory
	corpusDir, err := os.MkdirTemp("", "hfc_run_")
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to create temporary directory",
		})
		return
	}

	for _, corpusItem := range report.Corpus {
		log.Println("Copying corpus item", corpusItem, "to", corpusDir)
		cmd := exec.Command("cp", "-r", corpusItem, corpusDir)
		if err := cmd.Run(); err != nil {
			log.Printf("Error copying corpus item %s to %s: %v\n", corpusItem, corpusDir, err)
			continue
		}
	}

	taskID, err := generateTaskID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to generate task ID",
		})
		return
	}

	go func() {
		s.processCorpus(taskID, report.Fuzzer, corpusDir)
		defer os.RemoveAll(corpusDir)
	}()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Corpus report received successfully. Processing in background.",
	})

}

type PeekResultResBody struct {
	ConstraintGroups []analysis.ConstraintGroup          `json:"constraint_groups"`
	FuzzerScores     map[string]analysis.ConstraintScore `json:"fuzzer_scores"`
	FuzzerCovInc     map[string]int                      `json:"fuzzer_cov_inc"`
}

func (s *Server) handlePeekResult(c *gin.Context) {
	s.ConstraintGroupsMutex.Lock()

	result := PeekResultResBody{}

	result.ConstraintGroups = s.ConstraintGroups

	s.FuzzerScoresMutex.Lock()
	result.FuzzerScores = make(map[string]analysis.ConstraintScore)
	for k := range s.FuzzerScores {
		result.FuzzerScores[k] = s.FuzzerScores[k].Copy()
	}
	s.FuzzerScoresMutex.Unlock()

	s.FuzzerCovsMutex.Lock()
	result.FuzzerCovInc = make(map[string]int)
	for k := range s.FuzzerCovs {
		length := len(s.FuzzerCovs[k])
		if length > 1 {
			log.Println(s.FuzzerCovs[k])
			result.FuzzerCovInc[k] = s.FuzzerCovs[k][length-1] - s.FuzzerCovs[k][length-2]
		}
	}
	s.FuzzerCovsMutex.Unlock()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Constraint groups retrieved",
		Data:    result,
	})

	s.ConstraintGroupsMutex.Unlock()

}

type LogReqBody struct {
	Log string `json:"log"`
}

func (s *Server) handleLog(c *gin.Context) {

	var logReq LogReqBody
	if err := c.ShouldBindJSON(&logReq); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	// just print the log
	log.Println(logReq.Log)
	// return success
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Log received successfully",
	})

}
