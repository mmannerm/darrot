package acceptance

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// BotConfig contains configuration for the bot instance
type BotConfig struct {
	BinaryPath    string            `yaml:"binary_path"`
	ConfigPath    string            `yaml:"config_path"`
	WorkingDir    string            `yaml:"working_dir"`
	Environment   map[string]string `yaml:"environment"`
	LogLevel      string            `yaml:"log_level"`
	DiscordToken  string            `yaml:"discord_token"`
	DiscordAPIURL string            `yaml:"discord_api_url"`
}

// LogEntry represents a log entry from the bot
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

// botManager implements the BotManager interface
type botManager struct {
	config    *TestConfig
	botConfig *BotConfig
	process   *exec.Cmd
	logs      []LogEntry
	logsMu    sync.RWMutex
	running   bool
	healthy   bool
	mu        sync.RWMutex
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	logFile   *os.File
}

// NewBotManager creates a new bot manager instance
func NewBotManager(config *TestConfig) (BotManager, error) {
	botConfig := &BotConfig{
		BinaryPath:    config.Bot.BinaryPath,
		ConfigPath:    config.Bot.ConfigPath,
		WorkingDir:    config.Bot.WorkingDir,
		LogLevel:      config.Bot.LogLevel,
		Environment:   make(map[string]string),
		DiscordToken:  "test-bot-token-123",
		DiscordAPIURL: fmt.Sprintf("http://%s:%d", config.MockServer.Host, config.MockServer.Port),
	}

	// Set default binary path if not specified
	if botConfig.BinaryPath == "" {
		botConfig.BinaryPath = "./darrot"
	}

	// Set default working directory if not specified
	if botConfig.WorkingDir == "" {
		botConfig.WorkingDir = "."
	}

	// Create bot manager
	manager := &botManager{
		config:    config,
		botConfig: botConfig,
		logs:      make([]LogEntry, 0),
		running:   false,
		healthy:   false,
	}

	return manager, nil
}

// Start starts the bot instance
func (bm *botManager) Start(ctx context.Context, config *BotConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.running {
		return fmt.Errorf("bot is already running")
	}

	// Update config if provided
	if config != nil {
		bm.botConfig = config
	}

	// Prepare bot configuration file
	if err := bm.prepareBotConfig(); err != nil {
		return fmt.Errorf("failed to prepare bot config: %w", err)
	}

	// Create log file for bot output
	logFile, err := bm.createLogFile()
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	bm.logFile = logFile

	// Prepare command
	cmd := exec.CommandContext(ctx, bm.botConfig.BinaryPath)
	cmd.Dir = bm.botConfig.WorkingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range bm.botConfig.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add Discord configuration
	cmd.Env = append(cmd.Env, fmt.Sprintf("DISCORD_TOKEN=%s", bm.botConfig.DiscordToken))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DISCORD_API_URL=%s", bm.botConfig.DiscordAPIURL))
	cmd.Env = append(cmd.Env, fmt.Sprintf("LOG_LEVEL=%s", bm.botConfig.LogLevel))

	// Set up pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	bm.stdout = stdout

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	bm.stderr = stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start bot process: %w", err)
	}

	bm.process = cmd
	bm.running = true

	// Start log monitoring goroutines
	go bm.monitorLogs(stdout, "stdout")
	go bm.monitorLogs(stderr, "stderr")

	// Start health monitoring
	go bm.monitorHealth(ctx)

	return nil
}

// Stop stops the bot instance
func (bm *botManager) Stop() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if !bm.running || bm.process == nil {
		return nil
	}

	// Send SIGTERM to gracefully shutdown
	if err := bm.process.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill
		if killErr := bm.process.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill bot process: %w", killErr)
		}
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- bm.process.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "signal: terminated") {
			return fmt.Errorf("bot process exited with error: %w", err)
		}
	case <-time.After(10 * time.Second):
		// Force kill if graceful shutdown takes too long
		if err := bm.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill bot process: %w", err)
		}
		<-done // Wait for the process to actually exit
	}

	// Close log file
	if bm.logFile != nil {
		bm.logFile.Close()
		bm.logFile = nil
	}

	bm.running = false
	bm.healthy = false
	bm.process = nil

	return nil
}

// IsRunning returns true if the bot is currently running
func (bm *botManager) IsRunning() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.running
}

// IsHealthy returns true if the bot is running and healthy
func (bm *botManager) IsHealthy() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.running && bm.healthy
}

// GetLogs returns all collected log entries
func (bm *botManager) GetLogs() []LogEntry {
	bm.logsMu.RLock()
	defer bm.logsMu.RUnlock()

	logs := make([]LogEntry, len(bm.logs))
	copy(logs, bm.logs)
	return logs
}

// SendCommand sends a command to the bot (not applicable for Discord bots)
func (bm *botManager) SendCommand(command string) error {
	return fmt.Errorf("sending commands to Discord bot not supported")
}

// WaitForReady waits for the bot to become ready
func (bm *botManager) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if bm.IsHealthy() {
			return nil
		}

		// Check for startup errors in logs
		logs := bm.GetLogs()
		for _, log := range logs {
			if log.Level == "ERROR" || log.Level == "FATAL" {
				return fmt.Errorf("bot startup failed: %s", log.Message)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("bot failed to become ready within %v", timeout)
}

// Restart restarts the bot instance
func (bm *botManager) Restart() error {
	if err := bm.Stop(); err != nil {
		return fmt.Errorf("failed to stop bot for restart: %w", err)
	}

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)

	if err := bm.Start(context.Background(), nil); err != nil {
		return fmt.Errorf("failed to start bot after restart: %w", err)
	}

	return nil
}

// prepareBotConfig creates a temporary configuration file for the bot
func (bm *botManager) prepareBotConfig() error {
	if bm.botConfig.ConfigPath == "" {
		// Create temporary config file
		tempDir := os.TempDir()
		configPath := filepath.Join(tempDir, fmt.Sprintf("darrot-test-config-%d.yaml", time.Now().UnixNano()))

		configContent := fmt.Sprintf(`
discord:
  token: "%s"
  api_url: "%s"
  gateway_url: "%s"

tts:
  enabled: true
  voice: "en-US-Standard-A"
  speed: 1.0
  volume: 0.8

logging:
  level: "%s"
  format: "json"

data:
  directory: "%s"
`,
			bm.botConfig.DiscordToken,
			bm.botConfig.DiscordAPIURL,
			strings.Replace(bm.botConfig.DiscordAPIURL, "http://", "ws://", 1)+"/gateway",
			bm.botConfig.LogLevel,
			filepath.Join(bm.botConfig.WorkingDir, "test-data"),
		)

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		bm.botConfig.ConfigPath = configPath
	}

	return nil
}

// createLogFile creates a log file for bot output
func (bm *botManager) createLogFile() (*os.File, error) {
	logDir := filepath.Join(bm.botConfig.WorkingDir, "test-logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("bot-%d.log", time.Now().UnixNano()))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return logFile, nil
}

// monitorLogs monitors bot output and parses log entries
func (bm *botManager) monitorLogs(reader io.ReadCloser, source string) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Write to log file
		if bm.logFile != nil {
			fmt.Fprintf(bm.logFile, "[%s] %s: %s\n", time.Now().Format(time.RFC3339), source, line)
		}

		// Parse log entry
		entry := bm.parseLogEntry(line, source)

		bm.logsMu.Lock()
		bm.logs = append(bm.logs, entry)
		bm.logsMu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		bm.logsMu.Lock()
		bm.logs = append(bm.logs, LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Log monitoring error: %v", err),
			Source:    source,
		})
		bm.logsMu.Unlock()
	}
}

// parseLogEntry parses a log line into a structured log entry
func (bm *botManager) parseLogEntry(line, source string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   line,
		Source:    source,
	}

	// Try to parse structured log format (JSON or key=value)
	if strings.Contains(line, "level=") {
		// Parse key=value format
		parts := strings.Fields(line)
		for _, part := range parts {
			if strings.HasPrefix(part, "level=") {
				entry.Level = strings.TrimPrefix(part, "level=")
			} else if strings.HasPrefix(part, "msg=") {
				entry.Message = strings.Trim(strings.TrimPrefix(part, "msg="), "\"")
			} else if strings.HasPrefix(part, "time=") {
				if t, err := time.Parse(time.RFC3339, strings.Trim(strings.TrimPrefix(part, "time="), "\"")); err == nil {
					entry.Timestamp = t
				}
			}
		}
	} else {
		// Simple heuristic for log level detection
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "FATAL") {
			entry.Level = "ERROR"
		} else if strings.Contains(upperLine, "WARN") {
			entry.Level = "WARN"
		} else if strings.Contains(upperLine, "DEBUG") {
			entry.Level = "DEBUG"
		}
	}

	return entry
}

// monitorHealth monitors bot health status
func (bm *botManager) monitorHealth(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bm.checkHealth()
		}
	}
}

// checkHealth checks if the bot is healthy based on logs and process status
func (bm *botManager) checkHealth() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if !bm.running || bm.process == nil {
		bm.healthy = false
		return
	}

	// Check if process is still running
	if bm.process.ProcessState != nil && bm.process.ProcessState.Exited() {
		bm.healthy = false
		bm.running = false
		return
	}

	// Check logs for health indicators
	logs := bm.GetLogs()
	recentLogs := make([]LogEntry, 0)

	// Get logs from last 30 seconds
	cutoff := time.Now().Add(-30 * time.Second)
	for _, log := range logs {
		if log.Timestamp.After(cutoff) {
			recentLogs = append(recentLogs, log)
		}
	}

	// Look for health indicators
	hasReady := false
	hasError := false

	for _, log := range recentLogs {
		if strings.Contains(strings.ToLower(log.Message), "ready") ||
			strings.Contains(strings.ToLower(log.Message), "connected") ||
			strings.Contains(strings.ToLower(log.Message), "started") {
			hasReady = true
		}

		if log.Level == "ERROR" || log.Level == "FATAL" {
			hasError = true
		}
	}

	// Bot is healthy if it has shown ready status and no recent errors
	bm.healthy = hasReady && !hasError
}
