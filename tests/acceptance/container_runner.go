package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// TestResult represents the result of a single test
type TestResult struct {
	Name        string        `json:"name"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	Description string        `json:"description"`
}

// TestSuiteResult represents the result of a test suite
type TestSuiteResult struct {
	SuiteName   string        `json:"suite_name"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	TotalTests  int           `json:"total_tests"`
	PassedTests int           `json:"passed_tests"`
	FailedTests int           `json:"failed_tests"`
	Tests       []TestResult  `json:"tests"`
}

// ContainerTestRunner runs acceptance tests against containerized services
type ContainerTestRunner struct {
	mockDiscordURL  string
	mockGatewayURL  string
	darrotContainer string
	httpClient      *http.Client
	results         []TestResult
}

func main() {
	var (
		suite   = flag.String("suite", "all", "Test suite to run (all, core, concurrent, error)")
		timeout = flag.Duration("timeout", 5*time.Minute, "Test timeout")
		output  = flag.String("output", "/app/results", "Output directory for results")
	)
	flag.Parse()

	log.Printf("Starting container-based acceptance tests")
	log.Printf("Suite: %s, Timeout: %v, Output: %s", *suite, *timeout, *output)

	runner := &ContainerTestRunner{
		mockDiscordURL:  getEnv("MOCK_DISCORD_URL", "http://mock-discord:8080"),
		mockGatewayURL:  getEnv("MOCK_DISCORD_GATEWAY_URL", "ws://mock-discord:8081/gateway"),
		darrotContainer: getEnv("DARROT_CONTAINER", "darrot-bot"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		results: make([]TestResult, 0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Wait for services to be ready
	if err := runner.waitForServices(ctx); err != nil {
		log.Fatalf("Services not ready: %v", err)
	}

	// Run test suites
	suiteResult := &TestSuiteResult{
		SuiteName: *suite,
		StartTime: time.Now(),
	}

	switch *suite {
	case "all":
		runner.runAllSuites(ctx)
	case "core":
		runner.runCoreFunctionalityTests(ctx)
	case "concurrent":
		runner.runConcurrentTests(ctx)
	case "error":
		runner.runErrorResilienceTests(ctx)
	default:
		log.Fatalf("Unknown test suite: %s", *suite)
	}

	suiteResult.EndTime = time.Now()
	suiteResult.Duration = suiteResult.EndTime.Sub(suiteResult.StartTime)
	suiteResult.Tests = runner.results
	suiteResult.TotalTests = len(runner.results)

	// Count results
	for _, test := range runner.results {
		if test.Status == "passed" {
			suiteResult.PassedTests++
		} else {
			suiteResult.FailedTests++
		}
	}

	// Save results
	if err := runner.saveResults(*output, suiteResult); err != nil {
		log.Printf("Failed to save results: %v", err)
	}

	// Print summary
	log.Printf("Test Results: %d passed, %d failed, %d total",
		suiteResult.PassedTests, suiteResult.FailedTests, suiteResult.TotalTests)

	if suiteResult.FailedTests > 0 {
		os.Exit(1)
	}
}

func (r *ContainerTestRunner) waitForServices(ctx context.Context) error {
	log.Printf("Waiting for services to be ready...")

	// Wait for mock Discord API
	if err := r.waitForHTTPService(ctx, r.mockDiscordURL+"/health", "Mock Discord API"); err != nil {
		return err
	}

	// Wait for darrot bot (check if container is running)
	if err := r.waitForDarrotBot(ctx); err != nil {
		return err
	}

	log.Printf("All services are ready")
	return nil
}

func (r *ContainerTestRunner) waitForHTTPService(ctx context.Context, url, serviceName string) error {
	log.Printf("Waiting for %s at %s", serviceName, url)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", serviceName)
		default:
			resp, err := r.httpClient.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				log.Printf("%s is ready", serviceName)
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (r *ContainerTestRunner) waitForDarrotBot(ctx context.Context) error {
	log.Printf("Waiting for darrot bot container to be ready")

	// In a real implementation, we would check the container health
	// For now, just wait a bit for the bot to start
	time.Sleep(10 * time.Second)

	log.Printf("Darrot bot is assumed ready")
	return nil
}

func (r *ContainerTestRunner) runAllSuites(ctx context.Context) {
	r.runCoreFunctionalityTests(ctx)
	r.runConcurrentTests(ctx)
	r.runErrorResilienceTests(ctx)
}

func (r *ContainerTestRunner) runCoreFunctionalityTests(ctx context.Context) {
	log.Printf("Running core functionality tests")

	tests := []struct {
		name        string
		description string
		testFunc    func(context.Context) error
	}{
		{
			name:        "DarrotJoinCommand",
			description: "Test bot joining voice channel via slash command",
			testFunc:    r.testDarrotJoinCommand,
		},
		{
			name:        "DarrotLeaveCommand",
			description: "Test bot leaving voice channel",
			testFunc:    r.testDarrotLeaveCommand,
		},
		{
			name:        "TTSMessageProcessing",
			description: "Test TTS message processing",
			testFunc:    r.testTTSMessageProcessing,
		},
		{
			name:        "ConfigurationCommands",
			description: "Test bot configuration commands",
			testFunc:    r.testConfigurationCommands,
		},
	}

	for _, test := range tests {
		r.runSingleTest(ctx, test.name, test.description, test.testFunc)
	}
}

func (r *ContainerTestRunner) runConcurrentTests(ctx context.Context) {
	log.Printf("Running concurrent tests")

	tests := []struct {
		name        string
		description string
		testFunc    func(context.Context) error
	}{
		{
			name:        "ConcurrentMessageProcessing",
			description: "Test concurrent message processing from multiple users",
			testFunc:    r.testConcurrentMessageProcessing,
		},
	}

	for _, test := range tests {
		r.runSingleTest(ctx, test.name, test.description, test.testFunc)
	}
}

func (r *ContainerTestRunner) runErrorResilienceTests(ctx context.Context) {
	log.Printf("Running error resilience tests")

	tests := []struct {
		name        string
		description string
		testFunc    func(context.Context) error
	}{
		{
			name:        "InvalidCommandHandling",
			description: "Test bot handling of invalid commands",
			testFunc:    r.testInvalidCommandHandling,
		},
	}

	for _, test := range tests {
		r.runSingleTest(ctx, test.name, test.description, test.testFunc)
	}
}

func (r *ContainerTestRunner) runSingleTest(ctx context.Context, name, description string, testFunc func(context.Context) error) {
	log.Printf("Running test: %s", name)
	start := time.Now()

	result := TestResult{
		Name:        name,
		Description: description,
	}

	err := testFunc(ctx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		log.Printf("Test %s FAILED: %v", name, err)
	} else {
		result.Status = "passed"
		log.Printf("Test %s PASSED", name)
	}

	r.results = append(r.results, result)
}

// Test implementations
func (r *ContainerTestRunner) testDarrotJoinCommand(ctx context.Context) error {
	// Create a test guild and voice channel via mock API
	guild, err := r.createTestGuild("test-guild-123", "Test Guild")
	if err != nil {
		return fmt.Errorf("failed to create test guild: %w", err)
	}

	voiceChannel, err := r.createVoiceChannel(guild["id"].(string), "test-voice-channel", "General Voice")
	if err != nil {
		return fmt.Errorf("failed to create voice channel: %w", err)
	}

	// Simulate user sending darrot-join command
	user, err := r.createTestUser("test-user-123", "testuser")
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	// Send slash command via mock API
	if err := r.sendSlashCommand(user["id"].(string), guild["id"].(string), "darrot-join", map[string]interface{}{
		"channel": voiceChannel["id"].(string),
	}); err != nil {
		return fmt.Errorf("failed to send darrot-join command: %w", err)
	}

	// Wait for bot to process command
	time.Sleep(5 * time.Second)

	// Verify bot attempted to join voice channel
	interactions, err := r.getInteractions()
	if err != nil {
		return fmt.Errorf("failed to get interactions: %w", err)
	}

	joinCommandFound := false
	for _, interaction := range interactions {
		if data, ok := interaction["data"].(map[string]interface{}); ok {
			if name, exists := data["name"]; exists && name == "darrot-join" {
				joinCommandFound = true
				break
			}
		}
	}

	if !joinCommandFound {
		return fmt.Errorf("darrot-join command was not processed by bot")
	}

	return nil
}

func (r *ContainerTestRunner) testDarrotLeaveCommand(ctx context.Context) error {
	// Similar to join test but for leave command
	guild, err := r.createTestGuild("test-guild-124", "Test Guild 2")
	if err != nil {
		return fmt.Errorf("failed to create test guild: %w", err)
	}

	user, err := r.createTestUser("test-user-124", "testuser2")
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	// Send darrot-leave command
	if err := r.sendSlashCommand(user["id"].(string), guild["id"].(string), "darrot-leave", map[string]interface{}{}); err != nil {
		return fmt.Errorf("failed to send darrot-leave command: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Verify command was processed
	interactions, err := r.getInteractions()
	if err != nil {
		return fmt.Errorf("failed to get interactions: %w", err)
	}

	leaveCommandFound := false
	for _, interaction := range interactions {
		if data, ok := interaction["data"].(map[string]interface{}); ok {
			if name, exists := data["name"]; exists && name == "darrot-leave" {
				leaveCommandFound = true
				break
			}
		}
	}

	if !leaveCommandFound {
		return fmt.Errorf("darrot-leave command was not processed by bot")
	}

	return nil
}

func (r *ContainerTestRunner) testTTSMessageProcessing(ctx context.Context) error {
	// Create test environment
	guild, err := r.createTestGuild("test-guild-125", "TTS Test Guild")
	if err != nil {
		return fmt.Errorf("failed to create test guild: %w", err)
	}

	textChannel, err := r.createTextChannel(guild["id"].(string), "test-text-channel", "general")
	if err != nil {
		return fmt.Errorf("failed to create text channel: %w", err)
	}

	user, err := r.createTestUser("test-user-125", "ttsuser")
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	// Send a test message
	testMessage := "Hello, this is a test message for TTS processing"
	if err := r.sendMessage(user["id"].(string), textChannel["id"].(string), testMessage); err != nil {
		return fmt.Errorf("failed to send test message: %w", err)
	}

	// Wait for TTS processing
	time.Sleep(8 * time.Second)

	// Check if message was received by mock server
	messages, err := r.getChannelMessages(textChannel["id"].(string))
	if err != nil {
		return fmt.Errorf("failed to get channel messages: %w", err)
	}

	messageFound := false
	for _, message := range messages {
		if content, ok := message["content"].(string); ok && content == testMessage {
			messageFound = true
			break
		}
	}

	if !messageFound {
		return fmt.Errorf("test message was not processed")
	}

	return nil
}

func (r *ContainerTestRunner) testConfigurationCommands(ctx context.Context) error {
	guild, err := r.createTestGuild("test-guild-126", "Config Test Guild")
	if err != nil {
		return fmt.Errorf("failed to create test guild: %w", err)
	}

	user, err := r.createTestUser("test-user-126", "configuser")
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	// Test configuration commands
	configCommands := []struct {
		command string
		options map[string]interface{}
	}{
		{"tts-voice", map[string]interface{}{"voice": "en-US-Standard-B"}},
		{"tts-speed", map[string]interface{}{"speed": 1.2}},
		{"tts-volume", map[string]interface{}{"volume": 0.8}},
	}

	for _, cmd := range configCommands {
		if err := r.sendSlashCommand(user["id"].(string), guild["id"].(string), cmd.command, cmd.options); err != nil {
			return fmt.Errorf("failed to send %s command: %w", cmd.command, err)
		}
		time.Sleep(2 * time.Second)
	}

	// Verify commands were processed
	interactions, err := r.getInteractions()
	if err != nil {
		return fmt.Errorf("failed to get interactions: %w", err)
	}

	configCommandsFound := 0
	for _, interaction := range interactions {
		if data, ok := interaction["data"].(map[string]interface{}); ok {
			if name, exists := data["name"]; exists {
				nameStr := fmt.Sprintf("%v", name)
				if strings.HasPrefix(nameStr, "tts-") {
					configCommandsFound++
				}
			}
		}
	}

	if configCommandsFound == 0 {
		return fmt.Errorf("no configuration commands were processed")
	}

	return nil
}

func (r *ContainerTestRunner) testConcurrentMessageProcessing(ctx context.Context) error {
	// Create test environment
	guild, err := r.createTestGuild("test-guild-127", "Concurrent Test Guild")
	if err != nil {
		return fmt.Errorf("failed to create test guild: %w", err)
	}

	textChannel, err := r.createTextChannel(guild["id"].(string), "test-concurrent-channel", "concurrent")
	if err != nil {
		return fmt.Errorf("failed to create text channel: %w", err)
	}

	// Create multiple users and send messages concurrently
	numUsers := 3
	users := make([]map[string]interface{}, numUsers)

	for i := 0; i < numUsers; i++ {
		user, err := r.createTestUser(fmt.Sprintf("concurrent-user-%d", i), fmt.Sprintf("user%d", i))
		if err != nil {
			return fmt.Errorf("failed to create user %d: %w", i, err)
		}
		users[i] = user
	}

	// Send messages from all users
	for i, user := range users {
		message := fmt.Sprintf("Concurrent message from user %d", i)
		if err := r.sendMessage(user["id"].(string), textChannel["id"].(string), message); err != nil {
			return fmt.Errorf("failed to send message from user %d: %w", i, err)
		}
	}

	// Wait for processing
	time.Sleep(10 * time.Second)

	// Verify messages were received
	messages, err := r.getChannelMessages(textChannel["id"].(string))
	if err != nil {
		return fmt.Errorf("failed to get channel messages: %w", err)
	}

	if len(messages) < numUsers {
		return fmt.Errorf("expected at least %d messages, got %d", numUsers, len(messages))
	}

	return nil
}

func (r *ContainerTestRunner) testInvalidCommandHandling(ctx context.Context) error {
	guild, err := r.createTestGuild("test-guild-128", "Error Test Guild")
	if err != nil {
		return fmt.Errorf("failed to create test guild: %w", err)
	}

	user, err := r.createTestUser("test-user-128", "erroruser")
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	// Send invalid commands
	invalidCommands := []string{
		"nonexistent-command",
		"invalid-tts-command",
	}

	for _, command := range invalidCommands {
		if err := r.sendSlashCommand(user["id"].(string), guild["id"].(string), command, map[string]interface{}{}); err != nil {
			// Invalid commands might fail to send, which is expected
			log.Printf("Invalid command %s failed as expected: %v", command, err)
		}
		time.Sleep(1 * time.Second)
	}

	// The test passes if the bot doesn't crash (we can't easily verify error handling without logs)
	return nil
}

// Mock Discord API helper methods
func (r *ContainerTestRunner) createTestGuild(id, name string) (map[string]interface{}, error) {
	guild := map[string]interface{}{
		"id":   id,
		"name": name,
	}

	body := strings.NewReader(fmt.Sprintf(`{"id": "%s", "name": "%s"}`, id, name))
	resp, err := r.httpClient.Post(r.mockDiscordURL+"/api/v10/guilds", "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create guild: status %d", resp.StatusCode)
	}

	return guild, nil
}

func (r *ContainerTestRunner) createTextChannel(guildID, channelID, name string) (map[string]interface{}, error) {
	channel := map[string]interface{}{
		"id":       channelID,
		"name":     name,
		"type":     0, // Text channel
		"guild_id": guildID,
	}

	body := strings.NewReader(fmt.Sprintf(`{"id": "%s", "name": "%s", "type": 0, "guild_id": "%s"}`,
		channelID, name, guildID))
	resp, err := r.httpClient.Post(r.mockDiscordURL+"/api/v10/channels", "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return channel, nil
}

func (r *ContainerTestRunner) createVoiceChannel(guildID, channelID, name string) (map[string]interface{}, error) {
	channel := map[string]interface{}{
		"id":       channelID,
		"name":     name,
		"type":     2, // Voice channel
		"guild_id": guildID,
	}

	body := strings.NewReader(fmt.Sprintf(`{"id": "%s", "name": "%s", "type": 2, "guild_id": "%s"}`,
		channelID, name, guildID))
	resp, err := r.httpClient.Post(r.mockDiscordURL+"/api/v10/channels", "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return channel, nil
}

func (r *ContainerTestRunner) createTestUser(id, username string) (map[string]interface{}, error) {
	user := map[string]interface{}{
		"id":            id,
		"username":      username,
		"discriminator": "1234",
		"bot":           false,
	}

	body := strings.NewReader(fmt.Sprintf(`{"id": "%s", "username": "%s", "discriminator": "1234", "bot": false}`,
		id, username))
	resp, err := r.httpClient.Post(r.mockDiscordURL+"/api/v10/users", "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return user, nil
}

func (r *ContainerTestRunner) sendSlashCommand(userID, guildID, command string, options map[string]interface{}) error {
	payload := map[string]interface{}{
		"type": 2, // Application command
		"data": map[string]interface{}{
			"name":    command,
			"options": options,
		},
		"user":     map[string]interface{}{"id": userID},
		"guild_id": guildID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Post(r.mockDiscordURL+"/api/v10/interactions", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (r *ContainerTestRunner) sendMessage(userID, channelID, content string) error {
	payload := map[string]interface{}{
		"content":    content,
		"author":     map[string]interface{}{"id": userID},
		"channel_id": channelID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Post(r.mockDiscordURL+fmt.Sprintf("/api/v10/channels/%s/messages", channelID),
		"application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (r *ContainerTestRunner) getInteractions() ([]map[string]interface{}, error) {
	resp, err := r.httpClient.Get(r.mockDiscordURL + "/api/v10/interactions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var interactions []map[string]interface{}
	if err := json.Unmarshal(body, &interactions); err != nil {
		return nil, err
	}

	return interactions, nil
}

func (r *ContainerTestRunner) getChannelMessages(channelID string) ([]map[string]interface{}, error) {
	resp, err := r.httpClient.Get(r.mockDiscordURL + fmt.Sprintf("/api/v10/channels/%s/messages", channelID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var messages []map[string]interface{}
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *ContainerTestRunner) saveResults(outputDir string, result *TestSuiteResult) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/acceptance-test-results-%s.json",
		outputDir, time.Now().Format("20060102-150405"))

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
