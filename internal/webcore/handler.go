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
	Period   string   `json:"period,omitempty"` // "begin" or "end", optional
}

func (s *Server) processCorpus(taskID TaskID, fuzzer string, corpus string, period string) {

	workDir, err := os.MkdirTemp("", "hfc_work_")
	if err != nil {
		log.Printf("Error creating temporary work directory: %v\n", err)
		return
	}
	defer os.RemoveAll(workDir)
	
	log.Printf("Processing corpus for task %s, fuzzer: %s, period: %s\n", taskID, fuzzer, period)
	
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
	if period == "begin" {
		// For begin period, we might want to reset or start fresh tracking
		log.Printf("Period 'begin' for fuzzer: %s. Initializing coverage tracking.\n", fuzzer)
		s.FuzzerCovs[fuzzer] = []int{cov}  // Start fresh for this period
	} else {
		// For end period or no period specified, append to existing coverage
		if _, ok := s.FuzzerCovs[fuzzer]; !ok {
			// If fuzzer doesn't exist yet, initialize with __init__ as starting point
			if initCov, hasInit := s.FuzzerCovs["__init__"]; hasInit && len(initCov) > 0 {
				s.FuzzerCovs[fuzzer] = []int{initCov[0], cov}
			} else {
				s.FuzzerCovs[fuzzer] = []int{cov}
			}
		} else {
			s.FuzzerCovs[fuzzer] = append(s.FuzzerCovs[fuzzer], cov)
		}
	}

	s.FuzzerCovsMutex.Unlock()

	s.GlobalCovMutex.Lock()
	if s.GlobalCov == nil {
		s.GlobalCov = &progCovData
	} else {
		// Handle global coverage based on period
		if period == "begin" {
			// For begin period, we might want to track coverage separately or reset
			log.Println("Period 'begin' detected. Starting new coverage tracking.")
			// We could start fresh tracking or mark this as a baseline
		}
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
	
	// Handle line coverage based on period
	var prevFileLineCovs []analysis.FileLineCov
	if period == "begin" {
		// For begin period, we might want to start fresh comparison
		log.Println("Period 'begin': Starting fresh line coverage comparison")
		// We could use empty or initial coverage as baseline
		prevFileLineCovs = make([]analysis.FileLineCov, len(lineCov))
		// Copy structure but reset counts to 0 for fresh start
		for i := range lineCov {
			prevFileLineCovs[i] = lineCov[i]
			prevFileLineCovs[i].ResetCov()
		}
	} else {
		// For end period or no period, use existing FileLineCovs for comparison
		prevFileLineCovs = s.FileLineCovs
	}
	
	score := analysis.CalculateFuzzerScore(fuzzer, lineCov, prevFileLineCovs, s.AST, s.SourceCode, maputil.Keys(functions))
	
	// Update FileLineCovs for next comparison
	if period != "begin" {
		// Only update if not starting a new period
		s.FileLineCovs = lineCov
	} else {
		// For begin period, we might want to set this as the baseline
		log.Println("Period 'begin': Setting current coverage as baseline")
		s.FileLineCovs = lineCov
	}
	
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

	// Validate period parameter if provided
	if report.Period != "" && report.Period != "begin" && report.Period != "end" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid period value. Must be 'begin' or 'end' if provided",
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
		s.processCorpus(taskID, report.Fuzzer, corpusDir, report.Period)
		defer os.RemoveAll(corpusDir)
	}()

	responseMsg := "Corpus report received successfully. Processing in background."
	if report.Period != "" {
		responseMsg += " Period: " + report.Period
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: responseMsg,
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
