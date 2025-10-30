package acceptance

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// TestScenario defines the interface for acceptance test scenarios
type TestScenario interface {
	// Name returns the scenario name for reporting
	Name() string

	// Description returns a detailed description of what the scenario tests
	Description() string

	// Requirements returns the requirement IDs this scenario validates
	Requirements() []string

	// Setup prepares the test environment and dependencies
	Setup(ctx context.Context, env *TestEnvironment) error

	// Execute runs the actual test scenario
	Execute(ctx context.Context, env *TestEnvironment) error

	// Validate checks the test results and assertions
	Validate(ctx context.Context, env *TestEnvironment) error

	// Cleanup performs test cleanup and resource deallocation
	Cleanup(ctx context.Context, env *TestEnvironment) error

	// Timeout returns the maximum execution time for this scenario
	Timeout() time.Duration
}

// TestSuite orchestrates multiple test scenarios
type TestSuite struct {
	name      string
	scenarios []TestScenario
	config    *TestConfig
	results   *TestResults
	mu        sync.RWMutex
}

// TestEnvironment provides the test execution environment
type TestEnvironment struct {
	MockServer   MockDiscordAPI
	BotInstance  BotManager
	AudioCapture AudioValidator
	Config       *TestConfig
	Logger       *log.Logger
	Context      context.Context
	TestData     map[string]interface{}
	mu           sync.RWMutex
}

// MockDiscordAPI defines the interface for mock Discord API interactions
type MockDiscordAPI interface {
	Start(ctx context.Context) error
	Stop() error
	Reset() error
	IsHealthy() bool
	GetBaseURL() string
	GetGatewayURL() string
	SimulateUser(userID string) MockUser
	CreateGuild(guildID string) MockGuild
	GetInteractions() []Interaction
}

// MockUser represents a simulated Discord user
type MockUser interface {
	SendMessage(channelID, content string) error
	JoinVoiceChannel(channelID string) error
	LeaveVoiceChannel() error
	SendSlashCommand(command string, options map[string]interface{}) error
	GetID() string
	GetUsername() string
}

// MockGuild represents a simulated Discord guild
type MockGuild interface {
	CreateTextChannel(name string) MockChannel
	CreateVoiceChannel(name string) MockChannel
	AddUser(user MockUser) error
	SetPermissions(userID string, permissions int64) error
	GetID() string
	GetChannels() []MockChannel
}

// MockChannel represents a simulated Discord channel
type MockChannel interface {
	GetID() string
	GetName() string
	GetType() ChannelType
	SendMessage(content string, userID string) error
	GetMessages() []Message
}

// BotManager handles bot instance lifecycle for testing
type BotManager interface {
	Start(ctx context.Context, config *BotConfig) error
	Stop() error
	IsRunning() bool
	IsHealthy() bool
	GetLogs() []LogEntry
	SendCommand(command string) error
	WaitForReady(timeout time.Duration) error
	Restart() error
}

// AudioValidator handles audio processing validation
type AudioValidator interface {
	StartCapture(channelID string) error
	StopCapture(channelID string) (*AudioCaptureSession, error)
	ValidateFormat(channelID string, expectedFormat string) (*FormatValidation, error)
	AnalyzeQuality(channelID string) (*QualityReport, error)
	VerifyContent(audioData []byte, expectedText string) (*ContentValidation, error)
}

// TestConfig contains configuration for the test suite
type TestConfig struct {
	MockServer struct {
		Host           string        `yaml:"host"`
		Port           int           `yaml:"port"`
		GatewayPort    int           `yaml:"gateway_port"`
		StartupTimeout time.Duration `yaml:"startup_timeout"`
		ResponseDelay  time.Duration `yaml:"response_delay"`
	} `yaml:"mock_server"`

	Bot struct {
		BinaryPath     string        `yaml:"binary_path"`
		ConfigPath     string        `yaml:"config_path"`
		StartupTimeout time.Duration `yaml:"startup_timeout"`
		LogLevel       string        `yaml:"log_level"`
		WorkingDir     string        `yaml:"working_dir"`
	} `yaml:"bot"`

	Audio struct {
		CaptureTimeout    time.Duration `yaml:"capture_timeout"`
		ExpectedFormat    string        `yaml:"expected_format"`
		QualityThresholds struct {
			MinBitrate int           `yaml:"min_bitrate"`
			MaxLatency time.Duration `yaml:"max_latency"`
		} `yaml:"quality_thresholds"`
	} `yaml:"audio"`

	Test struct {
		DefaultTimeout time.Duration `yaml:"default_timeout"`
		RetryAttempts  int           `yaml:"retry_attempts"`
		RetryDelay     time.Duration `yaml:"retry_delay"`
		ParallelTests  int           `yaml:"parallel_tests"`
	} `yaml:"test"`
}

// TestResults contains the results of test execution
type TestResults struct {
	TestSuite    string           `json:"test_suite"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      time.Time        `json:"end_time"`
	Duration     time.Duration    `json:"duration"`
	TotalTests   int              `json:"total_tests"`
	PassedTests  int              `json:"passed_tests"`
	FailedTests  int              `json:"failed_tests"`
	SkippedTests int              `json:"skipped_tests"`
	TestCases    []TestCaseResult `json:"test_cases"`
	Artifacts    []TestArtifact   `json:"artifacts"`
	Environment  EnvironmentInfo  `json:"environment"`
}

// TestCaseResult contains the result of a single test case
type TestCaseResult struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Requirements []string      `json:"requirements"`
	Status       TestStatus    `json:"status"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	ErrorMsg     string        `json:"error_message,omitempty"`
	Logs         []string      `json:"logs,omitempty"`
	Metrics      TestMetrics   `json:"metrics,omitempty"`
	Artifacts    []string      `json:"artifacts,omitempty"`
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPassed  TestStatus = "passed"
	TestStatusFailed  TestStatus = "failed"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusRunning TestStatus = "running"
)

// TestMetrics contains performance and quality metrics
type TestMetrics struct {
	ExecutionTime     time.Duration `json:"execution_time"`
	MemoryUsage       int64         `json:"memory_usage"`
	AudioPackets      int           `json:"audio_packets"`
	AudioDuration     time.Duration `json:"audio_duration"`
	MessagesSent      int           `json:"messages_sent"`
	CommandsExecuted  int           `json:"commands_executed"`
	ErrorsEncountered int           `json:"errors_encountered"`
}

// TestArtifact represents a test artifact (logs, audio files, etc.)
type TestArtifact struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
}

// EnvironmentInfo contains information about the test environment
type EnvironmentInfo struct {
	OS            string            `json:"os"`
	Architecture  string            `json:"architecture"`
	GoVersion     string            `json:"go_version"`
	BotVersion    string            `json:"bot_version"`
	TestFramework string            `json:"test_framework"`
	Environment   map[string]string `json:"environment"`
}

// NewTestSuite creates a new test suite
func NewTestSuite(name string, config *TestConfig) *TestSuite {
	return &TestSuite{
		name:      name,
		scenarios: make([]TestScenario, 0),
		config:    config,
		results: &TestResults{
			TestSuite: name,
			TestCases: make([]TestCaseResult, 0),
			Artifacts: make([]TestArtifact, 0),
		},
	}
}

// AddScenario adds a test scenario to the suite
func (ts *TestSuite) AddScenario(scenario TestScenario) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.scenarios = append(ts.scenarios, scenario)
}

// Run executes all test scenarios in the suite
func (ts *TestSuite) Run(ctx context.Context) (*TestResults, error) {
	ts.results.StartTime = time.Now()
	ts.results.TotalTests = len(ts.scenarios)

	// Create test environment
	env, err := ts.createTestEnvironment(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create test environment: %w", err)
	}
	defer ts.cleanupTestEnvironment(env)

	// Execute scenarios
	for _, scenario := range ts.scenarios {
		result := ts.executeScenario(ctx, env, scenario)

		ts.mu.Lock()
		ts.results.TestCases = append(ts.results.TestCases, result)

		switch result.Status {
		case TestStatusPassed:
			ts.results.PassedTests++
		case TestStatusFailed:
			ts.results.FailedTests++
		case TestStatusSkipped:
			ts.results.SkippedTests++
		}
		ts.mu.Unlock()
	}

	ts.results.EndTime = time.Now()
	ts.results.Duration = ts.results.EndTime.Sub(ts.results.StartTime)

	return ts.results, nil
}

// executeScenario runs a single test scenario
func (ts *TestSuite) executeScenario(ctx context.Context, env *TestEnvironment, scenario TestScenario) TestCaseResult {
	result := TestCaseResult{
		Name:         scenario.Name(),
		Description:  scenario.Description(),
		Requirements: scenario.Requirements(),
		StartTime:    time.Now(),
		Status:       TestStatusRunning,
		Logs:         make([]string, 0),
		Artifacts:    make([]string, 0),
	}

	// Create scenario-specific context with timeout
	scenarioCtx, cancel := context.WithTimeout(ctx, scenario.Timeout())
	defer cancel()

	// Execute scenario phases
	phases := []struct {
		name string
		fn   func(context.Context, *TestEnvironment) error
	}{
		{"setup", scenario.Setup},
		{"execute", scenario.Execute},
		{"validate", scenario.Validate},
	}

	var lastError error

	for _, phase := range phases {
		if err := phase.fn(scenarioCtx, env); err != nil {
			result.ErrorMsg = fmt.Sprintf("Phase '%s' failed: %v", phase.name, err)
			result.Status = TestStatusFailed
			lastError = err
			break
		}
	}

	// Always run cleanup
	if err := scenario.Cleanup(scenarioCtx, env); err != nil {
		if lastError == nil {
			result.ErrorMsg = fmt.Sprintf("Cleanup failed: %v", err)
			result.Status = TestStatusFailed
		} else {
			result.ErrorMsg += fmt.Sprintf("; Cleanup also failed: %v", err)
		}
	}

	// Set final status if not already failed
	if result.Status == TestStatusRunning {
		result.Status = TestStatusPassed
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// createTestEnvironment sets up the test environment
func (ts *TestSuite) createTestEnvironment(ctx context.Context) (*TestEnvironment, error) {
	// Create logger
	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", ts.name), log.LstdFlags)

	env := &TestEnvironment{
		Config:   ts.config,
		Logger:   logger,
		Context:  ctx,
		TestData: make(map[string]interface{}),
	}

	// Initialize mock server (will be implemented in mock_server.go)
	mockServer, err := NewMockServer(ts.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create mock server: %w", err)
	}
	env.MockServer = mockServer

	// Initialize bot manager (will be implemented in bot_manager.go)
	botManager, err := NewBotManager(ts.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot manager: %w", err)
	}
	env.BotInstance = botManager

	// Initialize audio validator (will be implemented in audio_validator.go)
	audioValidator, err := NewAudioValidator(ts.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio validator: %w", err)
	}
	env.AudioCapture = audioValidator

	// Start mock server
	if err := env.MockServer.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start mock server: %w", err)
	}

	// Wait for mock server to be ready
	if !ts.waitForMockServer(env.MockServer, 30*time.Second) {
		return nil, fmt.Errorf("mock server failed to become ready")
	}

	return env, nil
}

// cleanupTestEnvironment cleans up the test environment
func (ts *TestSuite) cleanupTestEnvironment(env *TestEnvironment) {
	if env.BotInstance != nil {
		if err := env.BotInstance.Stop(); err != nil {
			env.Logger.Printf("Failed to stop bot instance: %v", err)
		}
	}

	if env.MockServer != nil {
		if err := env.MockServer.Stop(); err != nil {
			env.Logger.Printf("Failed to stop mock server: %v", err)
		}
	}
}

// waitForMockServer waits for the mock server to become ready
func (ts *TestSuite) waitForMockServer(server MockDiscordAPI, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if server.IsHealthy() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

// SetTestData stores test data in the environment
func (env *TestEnvironment) SetTestData(key string, value interface{}) {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.TestData[key] = value
}

// GetTestData retrieves test data from the environment
func (env *TestEnvironment) GetTestData(key string) (interface{}, bool) {
	env.mu.RLock()
	defer env.mu.RUnlock()
	value, exists := env.TestData[key]
	return value, exists
}

// LogInfo logs an informational message
func (env *TestEnvironment) LogInfo(format string, args ...interface{}) {
	env.Logger.Printf("[INFO] "+format, args...)
}

// LogError logs an error message
func (env *TestEnvironment) LogError(format string, args ...interface{}) {
	env.Logger.Printf("[ERROR] "+format, args...)
}

// LogDebug logs a debug message
func (env *TestEnvironment) LogDebug(format string, args ...interface{}) {
	env.Logger.Printf("[DEBUG] "+format, args...)
}
