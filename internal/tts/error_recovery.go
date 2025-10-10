package tts

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ErrorRecoveryManager handles comprehensive error recovery for TTS operations
type ErrorRecoveryManager struct {
	voiceManager  VoiceManager
	ttsManager    TTSManager
	messageQueue  MessageQueue
	configService ConfigService

	// Recovery configuration
	maxRetries          int
	retryDelay          time.Duration
	connectionTimeout   time.Duration
	healthCheckInterval time.Duration
	fallbackVoice       string

	// Connection monitoring
	connectionMonitor *ConnectionMonitor
	healthChecker     *HealthChecker

	// Error tracking
	errorStats map[string]*ErrorStats
	mu         sync.RWMutex

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ErrorStats tracks error statistics for a guild
type ErrorStats struct {
	GuildID                string
	VoiceConnectionErrors  int
	TTSConversionErrors    int
	AudioPlaybackErrors    int
	LastErrorTime          time.Time
	ConsecutiveFailures    int
	RecoveryAttempts       int
	LastSuccessfulActivity time.Time
}

// ConnectionMonitor monitors voice connection health
type ConnectionMonitor struct {
	voiceManager    VoiceManager
	errorRecovery   *ErrorRecoveryManager
	checkInterval   time.Duration
	connectionState map[string]*ConnectionState
	mu              sync.RWMutex
}

// ConnectionState tracks the state of a voice connection
type ConnectionState struct {
	GuildID            string
	IsHealthy          bool
	LastHealthCheck    time.Time
	ConsecutiveErrors  int
	LastError          error
	RecoveryInProgress bool
}

// HealthChecker performs periodic health checks on TTS components
type HealthChecker struct {
	ttsManager    TTSManager
	voiceManager  VoiceManager
	errorRecovery *ErrorRecoveryManager
	checkInterval time.Duration
	testText      string
	testConfig    TTSConfig
}

// NewErrorRecoveryManager creates a new comprehensive error recovery manager
func NewErrorRecoveryManager(voiceManager VoiceManager, ttsManager TTSManager, messageQueue MessageQueue, configService ConfigService) *ErrorRecoveryManager {
	return NewErrorRecoveryManagerWithConfig(voiceManager, ttsManager, messageQueue, configService, ErrorRecoveryConfig{})
}

// ErrorRecoveryConfig allows customizing error recovery behavior
type ErrorRecoveryConfig struct {
	MaxRetries          int
	RetryDelay          time.Duration
	ConnectionTimeout   time.Duration
	HealthCheckInterval time.Duration
	MonitorInterval     time.Duration
}

// NewErrorRecoveryManagerWithConfig creates a new error recovery manager with custom configuration
func NewErrorRecoveryManagerWithConfig(voiceManager VoiceManager, ttsManager TTSManager, messageQueue MessageQueue, configService ConfigService, config ErrorRecoveryConfig) *ErrorRecoveryManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults if not provided
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second * 2
	}
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = time.Second * 10
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = time.Minute * 2
	}
	if config.MonitorInterval == 0 {
		config.MonitorInterval = time.Second * 30
	}

	erm := &ErrorRecoveryManager{
		voiceManager:        voiceManager,
		ttsManager:          ttsManager,
		messageQueue:        messageQueue,
		configService:       configService,
		maxRetries:          config.MaxRetries,
		retryDelay:          config.RetryDelay,
		connectionTimeout:   config.ConnectionTimeout,
		healthCheckInterval: config.HealthCheckInterval,
		fallbackVoice:       DefaultVoice,
		errorStats:          make(map[string]*ErrorStats),
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Initialize connection monitor
	erm.connectionMonitor = &ConnectionMonitor{
		voiceManager:    voiceManager,
		errorRecovery:   erm,
		checkInterval:   config.MonitorInterval,
		connectionState: make(map[string]*ConnectionState),
	}

	// Initialize health checker
	erm.healthChecker = &HealthChecker{
		ttsManager:    ttsManager,
		voiceManager:  voiceManager,
		errorRecovery: erm,
		checkInterval: config.HealthCheckInterval,
		testText:      "Health check test",
		testConfig: TTSConfig{
			Voice:  DefaultVoice,
			Speed:  DefaultTTSSpeed,
			Volume: DefaultTTSVolume,
			Format: AudioFormatPCM,
		},
	}

	return erm
}

// Start begins error recovery monitoring
func (erm *ErrorRecoveryManager) Start() error {
	log.Println("Starting error recovery manager")

	// Start connection monitoring
	erm.wg.Add(1)
	go erm.connectionMonitor.start(erm.ctx, &erm.wg)

	// Start health checking
	erm.wg.Add(1)
	go erm.healthChecker.start(erm.ctx, &erm.wg)

	return nil
}

// Stop gracefully stops error recovery monitoring
func (erm *ErrorRecoveryManager) Stop() error {
	log.Println("Stopping error recovery manager")
	erm.cancel()
	erm.wg.Wait()
	return nil
}

// HandleVoiceDisconnection handles voice connection failures with automatic recovery
// Implements requirement 9.1: automatic reconnection logic for voice connections
func (erm *ErrorRecoveryManager) HandleVoiceDisconnection(guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	log.Printf("Handling voice disconnection for guild %s", guildID)

	// Update error statistics
	erm.updateErrorStats(guildID, "voice_connection")

	// Get connection state
	erm.connectionMonitor.mu.Lock()
	state, exists := erm.connectionMonitor.connectionState[guildID]
	if !exists {
		state = &ConnectionState{
			GuildID:   guildID,
			IsHealthy: false,
		}
		erm.connectionMonitor.connectionState[guildID] = state
	}

	// Check if recovery is already in progress
	if state.RecoveryInProgress {
		erm.connectionMonitor.mu.Unlock()
		return fmt.Errorf("recovery already in progress for guild %s", guildID)
	}

	state.RecoveryInProgress = true
	erm.connectionMonitor.mu.Unlock()

	defer func() {
		erm.connectionMonitor.mu.Lock()
		if state, exists := erm.connectionMonitor.connectionState[guildID]; exists {
			state.RecoveryInProgress = false
		}
		erm.connectionMonitor.mu.Unlock()
	}()

	// Attempt recovery with exponential backoff
	for attempt := 1; attempt <= erm.maxRetries; attempt++ {
		log.Printf("Voice connection recovery attempt %d/%d for guild %s", attempt, erm.maxRetries, guildID)

		// Wait before retry (exponential backoff)
		if attempt > 1 {
			backoffDelay := time.Duration(attempt*attempt) * erm.retryDelay
			time.Sleep(backoffDelay)
		}

		// Attempt to recover the connection
		err := erm.voiceManager.RecoverConnection(guildID)
		if err == nil {
			log.Printf("Successfully recovered voice connection for guild %s", guildID)

			// Update connection state
			erm.connectionMonitor.mu.Lock()
			if state, exists := erm.connectionMonitor.connectionState[guildID]; exists {
				state.IsHealthy = true
				state.ConsecutiveErrors = 0
				state.LastError = nil
				state.LastHealthCheck = time.Now()
			}
			erm.connectionMonitor.mu.Unlock()

			// Reset error stats on successful recovery
			erm.resetErrorStats(guildID)
			return nil
		}

		log.Printf("Voice connection recovery attempt %d failed for guild %s: %v", attempt, guildID, err)

		// Update connection state with error
		erm.connectionMonitor.mu.Lock()
		if state, exists := erm.connectionMonitor.connectionState[guildID]; exists {
			state.ConsecutiveErrors++
			state.LastError = err
		}
		erm.connectionMonitor.mu.Unlock()
	}

	// All recovery attempts failed
	err := fmt.Errorf("failed to recover voice connection after %d attempts", erm.maxRetries)
	log.Printf("Voice connection recovery failed for guild %s: %v", guildID, err)

	// Mark connection as unhealthy
	erm.connectionMonitor.mu.Lock()
	if state, exists := erm.connectionMonitor.connectionState[guildID]; exists {
		state.IsHealthy = false
		state.LastError = err
	}
	erm.connectionMonitor.mu.Unlock()

	return NewTTSError("voice_recovery", "automatic reconnection failed", guildID, "", err)
}

// HandleTTSFailure implements comprehensive fallback mechanisms for TTS failures
// Implements requirement 9.2: graceful handling of TTS engine failures
func (erm *ErrorRecoveryManager) HandleTTSFailure(text, voice string, config TTSConfig, guildID string) ([]byte, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}

	log.Printf("Handling TTS failure for guild %s", guildID)

	// Update error statistics
	erm.updateErrorStats(guildID, "tts_conversion")

	var lastErr error

	// Strategy 1: Retry with original configuration
	for attempt := 1; attempt <= erm.maxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("TTS retry attempt %d/%d for guild %s", attempt, erm.maxRetries, guildID)
			time.Sleep(erm.retryDelay)
		}

		audioData, err := erm.ttsManager.ConvertToSpeech(text, voice, config)
		if err == nil {
			log.Printf("TTS conversion succeeded on retry attempt %d for guild %s", attempt, guildID)
			erm.resetErrorStats(guildID)
			return audioData, nil
		}

		lastErr = err
		log.Printf("TTS retry attempt %d failed for guild %s: %v", attempt, guildID, err)

		// Don't retry fatal errors
		if IsFatalError(err) {
			break
		}
	}

	// Strategy 2: Try with fallback voice
	if voice != erm.fallbackVoice {
		log.Printf("Trying fallback voice %s for guild %s", erm.fallbackVoice, guildID)
		audioData, err := erm.ttsManager.ConvertToSpeech(text, erm.fallbackVoice, config)
		if err == nil {
			log.Printf("TTS conversion succeeded with fallback voice for guild %s", guildID)
			return audioData, nil
		}
		log.Printf("Fallback voice failed for guild %s: %v", guildID, err)
		lastErr = err
	}

	// Strategy 3: Try with simplified configuration
	fallbackConfig := TTSConfig{
		Voice:  erm.fallbackVoice,
		Speed:  DefaultTTSSpeed,
		Volume: DefaultTTSVolume,
		Format: config.Format,
	}

	log.Printf("Trying simplified configuration for guild %s", guildID)
	audioData, err := erm.ttsManager.ConvertToSpeech(text, "", fallbackConfig)
	if err == nil {
		log.Printf("TTS conversion succeeded with simplified config for guild %s", guildID)
		return audioData, nil
	}
	log.Printf("Simplified configuration failed for guild %s: %v", guildID, err)
	lastErr = err

	// Strategy 4: Try with truncated text
	if len(text) > 100 {
		truncatedText := text[:97] + "..."
		log.Printf("Trying truncated text for guild %s", guildID)
		audioData, err := erm.ttsManager.ConvertToSpeech(truncatedText, "", fallbackConfig)
		if err == nil {
			log.Printf("TTS conversion succeeded with truncated text for guild %s", guildID)
			return audioData, nil
		}
		log.Printf("Truncated text failed for guild %s: %v", guildID, err)
		lastErr = err
	}

	// Strategy 5: Try with error message as fallback
	errorMessage := "Sorry, I couldn't read that message."
	log.Printf("Trying error message fallback for guild %s", guildID)
	audioData, err = erm.ttsManager.ConvertToSpeech(errorMessage, "", fallbackConfig)
	if err == nil {
		log.Printf("Error message fallback succeeded for guild %s", guildID)
		return audioData, nil
	}

	// All strategies failed
	return nil, NewTTSError("conversion", "all fallback mechanisms failed", guildID, "", lastErr)
}

// HandleAudioPlaybackFailure handles audio playback failures
// Implements requirement 9.2: graceful handling of audio playback failures
func (erm *ErrorRecoveryManager) HandleAudioPlaybackFailure(guildID string, audioData []byte) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	log.Printf("Handling audio playback failure for guild %s", guildID)

	// Update error statistics
	erm.updateErrorStats(guildID, "audio_playback")

	// Check if voice connection is still healthy
	if !erm.voiceManager.IsConnected(guildID) {
		log.Printf("Voice connection lost during playback for guild %s, attempting recovery", guildID)
		if err := erm.HandleVoiceDisconnection(guildID); err != nil {
			return fmt.Errorf("failed to recover voice connection: %w", err)
		}
	}

	// Retry audio playback
	for attempt := 1; attempt <= erm.maxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("Audio playback retry attempt %d/%d for guild %s", attempt, erm.maxRetries, guildID)
			time.Sleep(erm.retryDelay)
		}

		err := erm.voiceManager.PlayAudio(guildID, audioData)
		if err == nil {
			log.Printf("Audio playback succeeded on retry attempt %d for guild %s", attempt, guildID)
			erm.resetErrorStats(guildID)
			return nil
		}

		log.Printf("Audio playback retry attempt %d failed for guild %s: %v", attempt, guildID, err)
	}

	return NewTTSError("audio_playback", "audio playback failed after retries", guildID, "", nil)
}

// CreateUserFriendlyErrorMessage creates user-friendly error messages for common failure scenarios
// Implements requirement 9.3: user-friendly error messages for common failure scenarios
func (erm *ErrorRecoveryManager) CreateUserFriendlyErrorMessage(err error, guildID string) string {
	if err == nil {
		return "An unknown error occurred."
	}

	errorStr := err.Error()

	// Voice connection errors
	if contains(errorStr, "voice connection") || contains(errorStr, "voice channel") {
		return "I'm having trouble connecting to the voice channel. Please try inviting me again, or check that I have the necessary permissions."
	}

	// Permission errors
	if contains(errorStr, "permission") || contains(errorStr, "access denied") {
		return "I don't have the necessary permissions to perform this action. Please check that I have voice channel and text channel permissions."
	}

	// TTS engine errors
	if contains(errorStr, "TTS") || contains(errorStr, "text-to-speech") {
		return "I'm having trouble converting text to speech right now. I'll keep trying, but some messages might be skipped."
	}

	// Rate limiting errors
	if contains(errorStr, "rate limit") || contains(errorStr, "quota") {
		return "I'm being rate limited by the text-to-speech service. Please wait a moment and try again."
	}

	// Network errors
	if contains(errorStr, "timeout") || contains(errorStr, "connection refused") {
		return "I'm having network connectivity issues. I'll keep trying to reconnect automatically."
	}

	// Configuration errors
	if contains(errorStr, "configuration") || contains(errorStr, "invalid") {
		return "There's an issue with the TTS configuration. Please check your settings or contact an administrator."
	}

	// Generic fallback
	return "I encountered an error, but I'll keep trying to work normally. If problems persist, please try restarting the TTS session."
}

// GetErrorStats returns error statistics for a guild
func (erm *ErrorRecoveryManager) GetErrorStats(guildID string) *ErrorStats {
	erm.mu.RLock()
	defer erm.mu.RUnlock()

	if stats, exists := erm.errorStats[guildID]; exists {
		// Return a copy to prevent external modification
		return &ErrorStats{
			GuildID:                stats.GuildID,
			VoiceConnectionErrors:  stats.VoiceConnectionErrors,
			TTSConversionErrors:    stats.TTSConversionErrors,
			AudioPlaybackErrors:    stats.AudioPlaybackErrors,
			LastErrorTime:          stats.LastErrorTime,
			ConsecutiveFailures:    stats.ConsecutiveFailures,
			RecoveryAttempts:       stats.RecoveryAttempts,
			LastSuccessfulActivity: stats.LastSuccessfulActivity,
		}
	}

	return &ErrorStats{GuildID: guildID}
}

// IsGuildHealthy returns whether a guild's TTS system is healthy
func (erm *ErrorRecoveryManager) IsGuildHealthy(guildID string) bool {
	erm.mu.RLock()
	stats, exists := erm.errorStats[guildID]
	erm.mu.RUnlock()

	if !exists {
		return true // No errors recorded
	}

	// Consider unhealthy if too many consecutive failures
	if stats.ConsecutiveFailures > 5 {
		return false
	}

	// Consider unhealthy if too many errors in recent time
	if time.Since(stats.LastErrorTime) < time.Minute*5 {
		totalErrors := stats.VoiceConnectionErrors + stats.TTSConversionErrors + stats.AudioPlaybackErrors
		if totalErrors > 10 {
			return false
		}
	}

	return true
}

// Helper methods

// updateErrorStats updates error statistics for a guild
func (erm *ErrorRecoveryManager) updateErrorStats(guildID, errorType string) {
	erm.mu.Lock()
	defer erm.mu.Unlock()

	stats, exists := erm.errorStats[guildID]
	if !exists {
		stats = &ErrorStats{
			GuildID:                guildID,
			LastSuccessfulActivity: time.Now(),
		}
		erm.errorStats[guildID] = stats
	}

	stats.LastErrorTime = time.Now()
	stats.ConsecutiveFailures++

	switch errorType {
	case "voice_connection":
		stats.VoiceConnectionErrors++
	case "tts_conversion":
		stats.TTSConversionErrors++
	case "audio_playback":
		stats.AudioPlaybackErrors++
	}
}

// resetErrorStats resets error statistics after successful operation
func (erm *ErrorRecoveryManager) resetErrorStats(guildID string) {
	erm.mu.Lock()
	defer erm.mu.Unlock()

	if stats, exists := erm.errorStats[guildID]; exists {
		stats.ConsecutiveFailures = 0
		stats.VoiceConnectionErrors = 0
		stats.TTSConversionErrors = 0
		stats.AudioPlaybackErrors = 0
		stats.LastSuccessfulActivity = time.Now()
	}
}

// Connection Monitor implementation

// start begins connection monitoring
func (cm *ConnectionMonitor) start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	log.Println("Starting connection monitor")

	for {
		select {
		case <-ctx.Done():
			log.Println("Connection monitor stopped")
			return
		case <-ticker.C:
			cm.checkAllConnections()
		}
	}
}

// checkAllConnections checks health of all active voice connections
func (cm *ConnectionMonitor) checkAllConnections() {
	activeGuilds := cm.voiceManager.GetActiveConnections()

	for _, guildID := range activeGuilds {
		cm.checkConnection(guildID)
	}
}

// checkConnection checks health of a specific voice connection
func (cm *ConnectionMonitor) checkConnection(guildID string) {
	cm.mu.Lock()
	state, exists := cm.connectionState[guildID]
	if !exists {
		state = &ConnectionState{
			GuildID:   guildID,
			IsHealthy: true,
		}
		cm.connectionState[guildID] = state
	}
	cm.mu.Unlock()

	// Perform health check
	healthResults := cm.voiceManager.HealthCheck()
	if err, exists := healthResults[guildID]; exists && err != nil {
		log.Printf("Voice connection health check failed for guild %s: %v", guildID, err)

		cm.mu.Lock()
		state.IsHealthy = false
		state.ConsecutiveErrors++
		state.LastError = err
		state.LastHealthCheck = time.Now()
		cm.mu.Unlock()

		// Trigger recovery if too many consecutive errors
		if state.ConsecutiveErrors >= 3 {
			go cm.errorRecovery.HandleVoiceDisconnection(guildID)
		}
	} else {
		cm.mu.Lock()
		state.IsHealthy = true
		state.ConsecutiveErrors = 0
		state.LastError = nil
		state.LastHealthCheck = time.Now()
		cm.mu.Unlock()
	}
}

// Health Checker implementation

// start begins health checking
func (hc *HealthChecker) start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	log.Println("Starting health checker")

	for {
		select {
		case <-ctx.Done():
			log.Println("Health checker stopped")
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// performHealthCheck performs comprehensive health checks
func (hc *HealthChecker) performHealthCheck() {
	// Test TTS engine
	_, err := hc.ttsManager.ConvertToSpeech(hc.testText, "", hc.testConfig)
	if err != nil {
		log.Printf("TTS health check failed: %v", err)
	} else {
		log.Printf("TTS health check passed")
	}

	// Test voice manager health
	activeGuilds := hc.voiceManager.GetActiveConnections()
	healthResults := hc.voiceManager.HealthCheck()

	healthyConnections := 0
	for _, guildID := range activeGuilds {
		if err, exists := healthResults[guildID]; !exists || err == nil {
			healthyConnections++
		}
	}

	log.Printf("Voice connections health: %d/%d healthy", healthyConnections, len(activeGuilds))
}

// Helper functions are defined in tts_errors.go
