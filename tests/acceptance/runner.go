package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"acceptance/scenarios"
)

// TestRunner orchestrates the execution of acceptance test suites
type TestRunner struct {
	config    *TestConfig
	suites    map[string]*TestSuite
	results   map[string]*TestResults
	outputDir string
}

// NewTestRunner creates a new test runner
func NewTestRunner(config *TestConfig) *TestRunner {
	if config == nil {
		config = DefaultTestConfig()
	}

	return &TestRunner{
		config:    config,
		suites:    make(map[string]*TestSuite),
		results:   make(map[string]*TestResults),
		outputDir: filepath.Join("tests", "results"),
	}
}

// AddSuite adds a test suite to the runner
func (tr *TestRunner) AddSuite(name string, suite *TestSuite) {
	tr.suites[name] = suite
}

// CreateCoreFunctionalitySuite creates a test suite for core bot functionality
func (tr *TestRunner) CreateCoreFunctionalitySuite() *TestSuite {
	suite := NewTestSuite("CoreFunctionality", tr.config)

	// Add core functionality scenarios
	suite.AddScenario(scenarios.NewDarrotJoinScenario())
	suite.AddScenario(scenarios.NewDarrotLeaveScenario())
	suite.AddScenario(scenarios.NewTTSMessageProcessingScenario())
	suite.AddScenario(scenarios.NewConfigurationCommandScenario())

	return suite
}

// CreateConcurrentTestingSuite creates a test suite for concurrent and multi-user testing
func (tr *TestRunner) CreateConcurrentTestingSuite() *TestSuite {
	suite := NewTestSuite("ConcurrentTesting", tr.config)

	// Add concurrent testing scenarios
	suite.AddScenario(scenarios.NewConcurrentMessageProcessingScenario())
	suite.AddScenario(scenarios.NewVoiceChannelUserManagementScenario())
	suite.AddScenario(scenarios.NewPermissionBasedAccessControlScenario())

	return suite
}

// CreateErrorResilienceSuite creates a test suite for error handling and resilience
func (tr *TestRunner) CreateErrorResilienceSuite() *TestSuite {
	suite := NewTestSuite("ErrorResilience", tr.config)

	// Add error resilience scenarios
	suite.AddScenario(scenarios.NewNetworkFailureScenario())
	suite.AddScenario(scenarios.NewRateLimitingScenario())
	suite.AddScenario(scenarios.NewInvalidCommandScenario())
	suite.AddScenario(scenarios.NewMockServerErrorScenario())

	return suite
}

// RunAll executes all registered test suites
func (tr *TestRunner) RunAll(ctx context.Context) error {
	fmt.Printf("Starting acceptance test execution with %d suites\n", len(tr.suites))

	// Create output directory
	if err := os.MkdirAll(tr.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	overallStart := time.Now()
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0

	// Execute each suite
	for suiteName, suite := range tr.suites {
		fmt.Printf("\n=== Running Test Suite: %s ===\n", suiteName)

		suiteStart := time.Now()
		results, err := suite.Run(ctx)
		suiteDuration := time.Since(suiteStart)

		if err != nil {
			fmt.Printf("Suite %s failed to execute: %v\n", suiteName, err)
			continue
		}

		tr.results[suiteName] = results

		// Print suite summary
		fmt.Printf("Suite %s completed in %v\n", suiteName, suiteDuration)
		fmt.Printf("  Tests: %d passed, %d failed, %d skipped\n",
			results.PassedTests, results.FailedTests, results.SkippedTests)

		totalPassed += results.PassedTests
		totalFailed += results.FailedTests
		totalSkipped += results.SkippedTests

		// Save suite results
		if err := tr.saveResults(suiteName, results); err != nil {
			fmt.Printf("Failed to save results for suite %s: %v\n", suiteName, err)
		}

		// Print failed test details
		if results.FailedTests > 0 {
			fmt.Printf("  Failed tests:\n")
			for _, testCase := range results.TestCases {
				if testCase.Status == TestStatusFailed {
					fmt.Printf("    - %s: %s\n", testCase.Name, testCase.ErrorMsg)
				}
			}
		}
	}

	overallDuration := time.Since(overallStart)

	// Print overall summary
	fmt.Printf("\n=== Overall Test Results ===\n")
	fmt.Printf("Total execution time: %v\n", overallDuration)
	fmt.Printf("Total tests: %d passed, %d failed, %d skipped\n",
		totalPassed, totalFailed, totalSkipped)

	// Generate combined report
	if err := tr.generateCombinedReport(); err != nil {
		fmt.Printf("Failed to generate combined report: %v\n", err)
	}

	if totalFailed > 0 {
		return fmt.Errorf("%d tests failed", totalFailed)
	}

	fmt.Printf("All tests passed successfully!\n")
	return nil
}

// RunSuite executes a specific test suite
func (tr *TestRunner) RunSuite(ctx context.Context, suiteName string) error {
	suite, exists := tr.suites[suiteName]
	if !exists {
		return fmt.Errorf("test suite %s not found", suiteName)
	}

	fmt.Printf("Running test suite: %s\n", suiteName)

	results, err := suite.Run(ctx)
	if err != nil {
		return fmt.Errorf("suite execution failed: %w", err)
	}

	tr.results[suiteName] = results

	// Print results
	fmt.Printf("Suite %s completed in %v\n", suiteName, results.Duration)
	fmt.Printf("Tests: %d passed, %d failed, %d skipped\n",
		results.PassedTests, results.FailedTests, results.SkippedTests)

	// Save results
	if err := tr.saveResults(suiteName, results); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	if results.FailedTests > 0 {
		return fmt.Errorf("%d tests failed in suite %s", results.FailedTests, suiteName)
	}

	return nil
}

// saveResults saves test results to a JSON file
func (tr *TestRunner) saveResults(suiteName string, results *TestResults) error {
	// Add environment information
	results.Environment = EnvironmentInfo{
		OS:            runtime.GOOS,
		Architecture:  runtime.GOARCH,
		GoVersion:     runtime.Version(),
		BotVersion:    "dev", // Could be read from version file
		TestFramework: "darrot-acceptance-tests",
		Environment:   make(map[string]string),
	}

	// Add relevant environment variables
	envVars := []string{"CI", "GITHUB_ACTIONS", "DOCKER_HOST"}
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			results.Environment.Environment[envVar] = value
		}
	}

	filename := fmt.Sprintf("%s-results-%s.json",
		suiteName, time.Now().Format("20060102-150405"))
	filepath := filepath.Join(tr.outputDir, filename)

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	fmt.Printf("Results saved to: %s\n", filepath)
	return nil
}

// generateCombinedReport generates a combined report of all test results
func (tr *TestRunner) generateCombinedReport() error {
	if len(tr.results) == 0 {
		return nil
	}

	combinedReport := struct {
		GeneratedAt   time.Time               `json:"generated_at"`
		TotalSuites   int                     `json:"total_suites"`
		TotalTests    int                     `json:"total_tests"`
		TotalPassed   int                     `json:"total_passed"`
		TotalFailed   int                     `json:"total_failed"`
		TotalSkipped  int                     `json:"total_skipped"`
		TotalDuration time.Duration           `json:"total_duration"`
		Suites        map[string]*TestResults `json:"suites"`
		Environment   EnvironmentInfo         `json:"environment"`
	}{
		GeneratedAt: time.Now(),
		TotalSuites: len(tr.results),
		Suites:      tr.results,
		Environment: EnvironmentInfo{
			OS:            runtime.GOOS,
			Architecture:  runtime.GOARCH,
			GoVersion:     runtime.Version(),
			BotVersion:    "dev",
			TestFramework: "darrot-acceptance-tests",
			Environment:   make(map[string]string),
		},
	}

	// Calculate totals
	var totalDuration time.Duration
	for _, results := range tr.results {
		combinedReport.TotalTests += results.TotalTests
		combinedReport.TotalPassed += results.PassedTests
		combinedReport.TotalFailed += results.FailedTests
		combinedReport.TotalSkipped += results.SkippedTests
		totalDuration += results.Duration
	}
	combinedReport.TotalDuration = totalDuration

	filename := fmt.Sprintf("combined-results-%s.json",
		time.Now().Format("20060102-150405"))
	filepath := filepath.Join(tr.outputDir, filename)

	data, err := json.MarshalIndent(combinedReport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal combined report: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write combined report: %w", err)
	}

	fmt.Printf("Combined report saved to: %s\n", filepath)
	return nil
}

// GetResults returns the results for a specific suite
func (tr *TestRunner) GetResults(suiteName string) *TestResults {
	return tr.results[suiteName]
}

// GetAllResults returns all test results
func (tr *TestRunner) GetAllResults() map[string]*TestResults {
	results := make(map[string]*TestResults)
	for k, v := range tr.results {
		results[k] = v
	}
	return results
}

// SetOutputDir sets the output directory for test results
func (tr *TestRunner) SetOutputDir(dir string) {
	tr.outputDir = dir
}

// ListSuites returns the names of all registered test suites
func (tr *TestRunner) ListSuites() []string {
	suites := make([]string, 0, len(tr.suites))
	for name := range tr.suites {
		suites = append(suites, name)
	}
	return suites
}
