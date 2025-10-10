package tts

import (
	"fmt"
	"log"
	"time"
)

// TTS-specific errors
var (
	ErrTTSEngineUnavailable  = fmt.Errorf("TTS engine is unavailable")
	ErrAudioConversionFailed = fmt.Errorf("audio format conversion failed")
	ErrInvalidVoiceConfig    = fmt.Errorf("invalid voice configuration")
	ErrTextTooLong           = fmt.Errorf("text exceeds maximum length")
	ErrEmptyText             = fmt.Errorf("text cannot be empty")
)

// TTSError represents a TTS-specific error with context
type TTSError struct {
	Type      string
	Message   string
	GuildID   string
	UserID    string
	Timestamp time.Time
	Cause     error
}

func (e *TTSError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("TTS %s error for guild %s: %s (caused by: %v)",
			e.Type, e.GuildID, e.Message, e.Cause)
	}
	return fmt.Sprintf("TTS %s error for guild %s: %s", e.Type, e.GuildID, e.Message)
}

func (e *TTSError) Unwrap() error {
	return e.Cause
}

// NewTTSError creates a new TTS error with context
func NewTTSError(errorType, message, guildID, userID string, cause error) *TTSError {
	return &TTSError{
		Type:      errorType,
		Message:   message,
		GuildID:   guildID,
		UserID:    userID,
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// ErrorRecovery handles TTS error recovery mechanisms
type ErrorRecovery struct {
	maxRetries    int
	retryDelay    time.Duration
	fallbackVoice string
}

// NewErrorRecovery creates a new error recovery handler
func NewErrorRecovery() *ErrorRecovery {
	return &ErrorRecovery{
		maxRetries:    3,
		retryDelay:    time.Second * 2,
		fallbackVoice: DefaultVoice,
	}
}

// HandleTTSFailure implements fallback mechanisms for TTS failures
func (er *ErrorRecovery) HandleTTSFailure(manager *GoogleTTSManager, text, voice string, config TTSConfig, guildID string) ([]byte, error) {
	var lastErr error

	// Check if manager has a valid client
	if manager.client == nil {
		return nil, NewTTSError("conversion", "TTS client not available", guildID, "", ErrTTSEngineUnavailable)
	}

	// Retry with original configuration
	for attempt := 0; attempt < er.maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying TTS conversion for guild %s, attempt %d/%d", guildID, attempt+1, er.maxRetries)
			time.Sleep(er.retryDelay)
		}

		audioData, err := manager.ConvertToSpeech(text, voice, config)
		if err == nil {
			return audioData, nil
		}

		lastErr = err
		log.Printf("TTS conversion attempt %d failed for guild %s: %v", attempt+1, guildID, err)
	}

	// Try with fallback voice
	if voice != er.fallbackVoice {
		log.Printf("Trying fallback voice %s for guild %s", er.fallbackVoice, guildID)
		audioData, err := manager.ConvertToSpeech(text, er.fallbackVoice, config)
		if err == nil {
			return audioData, nil
		}
		log.Printf("Fallback voice failed for guild %s: %v", guildID, err)
	}

	// Try with simplified config (default speed and volume)
	fallbackConfig := TTSConfig{
		Voice:  er.fallbackVoice,
		Speed:  DefaultTTSSpeed,
		Volume: DefaultTTSVolume,
		Format: config.Format,
	}

	log.Printf("Trying simplified config for guild %s", guildID)
	audioData, err := manager.ConvertToSpeech(text, "", fallbackConfig)
	if err == nil {
		return audioData, nil
	}

	// Try with truncated text if original was too long
	if len(text) > 100 {
		truncatedText := text[:97] + "..."
		log.Printf("Trying truncated text for guild %s", guildID)
		audioData, err := manager.ConvertToSpeech(truncatedText, "", fallbackConfig)
		if err == nil {
			return audioData, nil
		}
	}

	// All fallback mechanisms failed
	return nil, NewTTSError("conversion", "all fallback mechanisms failed", guildID, "", lastErr)
}

// HandleVoiceDisconnection handles voice connection failures
func (er *ErrorRecovery) HandleVoiceDisconnection(guildID string) error {
	log.Printf("Handling voice disconnection for guild %s", guildID)

	// TODO: This would typically interact with VoiceManager to attempt reconnection
	// For now, we'll just log the event

	return NewTTSError("voice_connection", "voice connection lost", guildID, "", nil)
}

// HandlePermissionError handles permission-related errors
func (er *ErrorRecovery) HandlePermissionError(userID, guildID string) error {
	log.Printf("Handling permission error for user %s in guild %s", userID, guildID)

	return NewTTSError("permission", "insufficient permissions", guildID, userID, nil)
}

// HandleRateLimit handles rate limiting from TTS service
func (er *ErrorRecovery) HandleRateLimit(retryAfter time.Duration, guildID string) error {
	log.Printf("Handling rate limit for guild %s, retry after %v", guildID, retryAfter)

	// Wait for the specified duration
	time.Sleep(retryAfter)

	return nil
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable error patterns
	errorStr := err.Error()

	// Network-related errors are typically retryable
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"rate limit",
		"quota exceeded",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// IsFatalError determines if an error is fatal and should not be retried
func IsFatalError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// Fatal error patterns that should not be retried
	fatalPatterns := []string{
		"invalid credentials",
		"permission denied",
		"invalid voice",
		"malformed request",
		"text too long",
	}

	for _, pattern := range fatalPatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TTSHealthChecker monitors TTS engine health
type TTSHealthChecker struct {
	manager       *GoogleTTSManager
	checkInterval time.Duration
	testText      string
	testConfig    TTSConfig
}

// NewTTSHealthChecker creates a new health checker
func NewTTSHealthChecker(manager *GoogleTTSManager) *TTSHealthChecker {
	return &TTSHealthChecker{
		manager:       manager,
		checkInterval: time.Minute * 5,
		testText:      "Health check test",
		testConfig: TTSConfig{
			Voice:  DefaultVoice,
			Speed:  DefaultTTSSpeed,
			Volume: DefaultTTSVolume,
			Format: AudioFormatPCM,
		},
	}
}

// StartHealthCheck starts periodic health checking
func (hc *TTSHealthChecker) StartHealthCheck() {
	ticker := time.NewTicker(hc.checkInterval)
	go func() {
		for range ticker.C {
			hc.performHealthCheck()
		}
	}()
}

// performHealthCheck performs a health check on the TTS engine
func (hc *TTSHealthChecker) performHealthCheck() {
	_, err := hc.manager.ConvertToSpeech(hc.testText, "", hc.testConfig)
	if err != nil {
		log.Printf("TTS health check failed: %v", err)
	} else {
		log.Printf("TTS health check passed")
	}
}
