package prerun

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/grubwithu/hfc/internal/analysis"
	"github.com/grubwithu/hfc/internal/plugin"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

type PrerunData struct {
	Cov        int
	ProgCov    analysis.ProgCovData
	LineCov    []analysis.FileLineCov
	AST        map[string]*sitter.Tree
	SourceCode map[string][]byte
}

// Plugin handles initial corpus processing
type Plugin struct {
	config        plugin.PluginConfig
	initData      PrerunData
	isInitialized bool
}

// NewPlugin creates a new prerun plugin
func NewPlugin() *Plugin {
	return &Plugin{
		isInitialized: false,
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "prerun"
}

// Require checks if the plugin should process the given data
// Prerun plugin always processes data
func (p *Plugin) Require(data *plugin.PluginData) bool {
	return true
}

// Init initializes the plugin with executable and generates initial coverage data
func (p *Plugin) Init(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config

	// Get executable from config
	if config.Executable == nil {
		log.Printf("PrerunPlugin: executable not provided in config\n")
		return nil
	}
	executable := *config.Executable

	// 1. Create a temp directory for initial corpus
	corpusDir, err := os.MkdirTemp("", "hfc_corpus_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(corpusDir)

	// 2. Create seed file
	seedPath := filepath.Join(corpusDir, "seed")
	if err := os.WriteFile(seedPath, []byte("0"), 0644); err != nil {
		return err
	}

	// 3. Create a temp work directory
	workDir, err := os.MkdirTemp("", "hfc_work_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	// 4. Run analysis to generate profdata
	cov, profdataPath, err := analysis.RunOnceForProfdata(workDir, executable, corpusDir)
	if err != nil {
		return err
	}

	// 5. Get program coverage data
	progCovData, err := analysis.GetProgCov(workDir, executable, profdataPath)
	if err != nil {
		return err
	}

	// 6. Get line coverage data
	fileLineCovs, err := analysis.GetLineCov(workDir, executable, profdataPath)
	if err != nil {
		return err
	}

	// Reset coverage for fresh start
	for i := range fileLineCovs {
		fileLineCovs[i].ResetCov()
	}

	// 7. Parse the code in files
	ast := map[string]*sitter.Tree{}
	sourceCode := map[string][]byte{}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cpp.GetLanguage()) // C++ is a superset of C
	for _, file := range fileLineCovs {
		code := file.GetOriginCode()
		sourceCode[file.File] = code
		tree, _ := parser.ParseCtx(context.Background(), nil, code)
		ast[file.File] = tree
	}

	// Store initialization data
	p.initData = PrerunData{
		Cov:        cov,
		ProgCov:    progCovData,
		LineCov:    fileLineCovs,
		AST:        ast,
		SourceCode: sourceCode,
	}
	p.isInitialized = true

	log.Printf("PrerunPlugin initialized with coverage: %d\n", cov)
	return nil
}

// GetInitData returns the initialization data for other plugins
func (p *Plugin) GetInitData() PrerunData {
	return p.initData
}

// Process processes the corpus and generates coverage data
func (p *Plugin) Process(ctx context.Context, data *plugin.PluginData) error {
	// Get executable from config
	if p.config.Executable == nil {
		return nil
	}
	executable := *p.config.Executable

	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "hfc_work_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	// Run analysis to generate profdata
	cov, profdataPath, err := analysis.RunOnceForProfdata(workDir, executable, data.Corpus)
	if err != nil {
		return err
	}

	// Get program coverage data
	progCovData, err := analysis.GetProgCov(workDir, executable, profdataPath)
	if err != nil {
		return err
	}

	// Get line coverage data
	lineCov, err := analysis.GetLineCov(workDir, executable, profdataPath)
	if err != nil {
		return err
	}

	// Store results in PluginData
	prerunResult := PrerunData{
		Cov:        cov,
		ProgCov:    progCovData,
		LineCov:    lineCov,
		AST:        p.initData.AST,
		SourceCode: p.initData.SourceCode,
	}

	// Store in shared data for other plugins
	if data.Data == nil {
		data.Data = make(map[string]any)
	}
	data.Data["prerun"] = prerunResult

	log.Printf("PrerunPlugin processed corpus %s for fuzzer %s\n", data.Corpus, data.Fuzzer)

	return nil
}

// Prerun plugin should return nothing
func (p *Plugin) Result(ctx context.Context) (any, error) {
	return nil, nil
}

// Cleanup cleans up the plugin resources
func (p *Plugin) Cleanup(ctx context.Context) error {
	log.Println("PrerunPlugin cleanup")
	return nil
}

// Priority returns the plugin priority
func (p *Plugin) Priority() int {
	return 1000 // Highest priority to run first
}
