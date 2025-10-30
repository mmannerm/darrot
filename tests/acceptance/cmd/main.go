package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	acceptance "acceptance"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to test configuration file")
		suite      = flag.String("suite", "", "Specific test suite to run (leave empty to run all)")
		timeout    = flag.Duration("timeout", 10*time.Minute, "Overall test timeout")
		outputDir  = flag.String("output", "tests/results", "Output directory for test results")
		listSuites = flag.Bool("list", false, "List available test suites")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Set up logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	// Load configuration
	config, err := acceptance.LoadTestConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load test configuration: %v", err)
	}

	// Validate configuration
	if err := acceptance.ValidateTestConfig(config); err != nil {
		log.Fatalf("Invalid test configuration: %v", err)
	}

	// Create test runner
	runner := acceptance.NewTestRunner(config)
	runner.SetOutputDir(*outputDir)

	// Register test suites
	runner.AddSuite("CoreFunctionality", runner.CreateCoreFunctionalitySuite())
	runner.AddSuite("ConcurrentTesting", runner.CreateConcurrentTestingSuite())
	runner.AddSuite("ErrorResilience", runner.CreateErrorResilienceSuite())

	// List suites if requested
	if *listSuites {
		fmt.Println("Available test suites:")
		for _, suiteName := range runner.ListSuites() {
			fmt.Printf("  - %s\n", suiteName)
		}
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Run tests
	var err error
	if *suite != "" {
		// Run specific suite
		fmt.Printf("Running test suite: %s\n", *suite)
		err = runner.RunSuite(ctx, *suite)
	} else {
		// Run all suites
		fmt.Println("Running all test suites")
		err = runner.RunAll(ctx)
	}

	if err != nil {
		log.Fatalf("Test execution failed: %v", err)
	}

	fmt.Println("All tests completed successfully!")
}

// Example usage functions

// RunCoreTests demonstrates running only core functionality tests
func RunCoreTests() error {
	config := acceptance.DefaultTestConfig()

	// Customize config for core tests
	config.Test.DefaultTimeout = 30 * time.Second
	config.Bot.LogLevel = "INFO"

	runner := acceptance.NewTestRunner(config)
	suite := runner.CreateCoreFunctionalitySuite()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	results, err := suite.Run(ctx)
	if err != nil {
		return fmt.Errorf("core tests failed: %w", err)
	}

	fmt.Printf("Core tests completed: %d passed, %d failed\n",
		results.PassedTests, results.FailedTests)

	return nil
}

// RunQuickSmokeTest demonstrates running a minimal smoke test
func RunQuickSmokeTest() error {
	config := acceptance.DefaultTestConfig()

	// Quick test configuration
	config.Test.DefaultTimeout = 15 * time.Second
	config.Bot.StartupTimeout = 10 * time.Second
	config.MockServer.StartupTimeout = 5 * time.Second

	suite := acceptance.NewTestSuite("SmokeTest", config)

	// Add only essential scenarios for smoke testing
	// suite.AddScenario(scenarios.NewDarrotJoinScenario())
	// suite.AddScenario(scenarios.NewTTSMessageProcessingScenario())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	results, err := suite.Run(ctx)
	if err != nil {
		return fmt.Errorf("smoke test failed: %w", err)
	}

	if results.FailedTests > 0 {
		return fmt.Errorf("smoke test had %d failures", results.FailedTests)
	}

	fmt.Println("Smoke test passed!")
	return nil
}

// RunWithCustomConfig demonstrates running tests with custom configuration
func RunWithCustomConfig(configOverrides map[string]interface{}) error {
	config := acceptance.DefaultTestConfig()

	// Apply custom overrides
	if botPath, ok := configOverrides["bot_path"].(string); ok {
		config.Bot.BinaryPath = botPath
	}
	if logLevel, ok := configOverrides["log_level"].(string); ok {
		config.Bot.LogLevel = logLevel
	}
	if timeout, ok := configOverrides["timeout"].(time.Duration); ok {
		config.Test.DefaultTimeout = timeout
	}

	runner := acceptance.NewTestRunner(config)
	runner.AddSuite("Custom", runner.CreateCoreFunctionalitySuite())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return runner.RunAll(ctx)
}

// RunCITests demonstrates running tests in CI environment
func RunCITests() error {
	config := acceptance.DefaultTestConfig()

	// CI-specific configuration
	config.Bot.LogLevel = "DEBUG"                 // More verbose logging for CI
	config.Test.RetryAttempts = 2                 // Retry failed tests in CI
	config.Test.DefaultTimeout = 45 * time.Second // Longer timeout for CI

	// Check for CI environment variables
	if os.Getenv("CI") == "true" {
		fmt.Println("Running in CI environment")

		// Use different binary path if specified
		if ciBotPath := os.Getenv("CI_BOT_PATH"); ciBotPath != "" {
			config.Bot.BinaryPath = ciBotPath
		}

		// Use different output directory
		if ciOutputDir := os.Getenv("CI_OUTPUT_DIR"); ciOutputDir != "" {
			// Will be set on runner
		}
	}

	runner := acceptance.NewTestRunner(config)

	// Add all suites for comprehensive CI testing
	runner.AddSuite("CoreFunctionality", runner.CreateCoreFunctionalitySuite())
	runner.AddSuite("ConcurrentTesting", runner.CreateConcurrentTestingSuite())
	runner.AddSuite("ErrorResilience", runner.CreateErrorResilienceSuite())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	err := runner.RunAll(ctx)

	// In CI, always save results even if tests fail
	results := runner.GetAllResults()
	for suiteName, result := range results {
		fmt.Printf("Suite %s: %d/%d tests passed\n",
			suiteName, result.PassedTests, result.TotalTests)
	}

	return err
}

// init sets up environment-specific defaults
func init() {
	// Set default configuration based on environment
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// GitHub Actions specific setup
		os.Setenv("CI", "true")
	}

	if os.Getenv("CI") == "true" {
		// CI environment defaults
		if os.Getenv("BOT_BINARY_PATH") == "" {
			os.Setenv("BOT_BINARY_PATH", "./darrot")
		}
	}
}
