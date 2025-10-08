package tts

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTSError(t *testing.T) {
	tests := []struct {
		name           string
		errorType      string
		message        string
		guildID        string
		userID         string
		cause          error
		expectedString string
	}{
		{
			name:           "error without cause",
			errorType:      "conversion",
			message:        "failed to convert text",
			guildID:        "guild123",
			userID:         "user123",
			cause:          nil,
			expectedString: "TTS conversion error for guild guild123: failed to convert text",
		},
		{
			name:           "error with cause",
			errorType:      "connection",
			message:        "connection failed",
			guildID:        "guild456",
			userID:         "user456",
			cause:          errors.New("network timeout"),
			expectedString: "TTS connection error for guild guild456: connection failed (caused by: network timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTTSError(tt.errorType, tt.message, tt.guildID, tt.userID, tt.cause)

			assert.Equal(t, tt.errorType, err.Type)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.guildID, err.GuildID)
			assert.Equal(t, tt.userID, err.UserID)
			assert.Equal(t, tt.cause, err.Cause)
			assert.Equal(t, tt.expectedString, err.Error())

			if tt.cause != nil {
				assert.Equal(t, tt.cause, err.Unwrap())
			} else {
				assert.Nil(t, err.Unwrap())
			}
		})
	}
}

func TestNewErrorRecovery(t *testing.T) {
	recovery := NewErrorRecovery()

	assert.NotNil(t, recovery)
	assert.Equal(t, 3, recovery.maxRetries)
	assert.Equal(t, time.Second*2, recovery.retryDelay)
	assert.Equal(t, DefaultVoice, recovery.fallbackVoice)
}

func TestErrorRecovery_HandleVoiceDisconnection(t *testing.T) {
	recovery := NewErrorRecovery()

	err := recovery.HandleVoiceDisconnection("guild123")

	assert.Error(t, err)

	var ttsErr *TTSError
	assert.True(t, errors.As(err, &ttsErr))
	assert.Equal(t, "voice_connection", ttsErr.Type)
	assert.Equal(t, "guild123", ttsErr.GuildID)
}

func TestErrorRecovery_HandlePermissionError(t *testing.T) {
	recovery := NewErrorRecovery()

	err := recovery.HandlePermissionError("user123", "guild456")

	assert.Error(t, err)

	var ttsErr *TTSError
	assert.True(t, errors.As(err, &ttsErr))
	assert.Equal(t, "permission", ttsErr.Type)
	assert.Equal(t, "guild456", ttsErr.GuildID)
	assert.Equal(t, "user123", ttsErr.UserID)
}

func TestErrorRecovery_HandleRateLimit(t *testing.T) {
	recovery := NewErrorRecovery()

	start := time.Now()
	err := recovery.HandleRateLimit(time.Millisecond*100, "guild123")
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, time.Millisecond*100)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "connection refused error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "temporary failure",
			err:      errors.New("temporary failure in name resolution"),
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      errors.New("service unavailable"),
			expected: true,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "quota exceeded",
			err:      errors.New("quota exceeded"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("invalid input"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFatalError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "invalid credentials",
			err:      errors.New("invalid credentials provided"),
			expected: true,
		},
		{
			name:     "permission denied",
			err:      errors.New("permission denied"),
			expected: true,
		},
		{
			name:     "invalid voice",
			err:      errors.New("invalid voice specified"),
			expected: true,
		},
		{
			name:     "malformed request",
			err:      errors.New("malformed request body"),
			expected: true,
		},
		{
			name:     "text too long",
			err:      errors.New("text too long for processing"),
			expected: true,
		},
		{
			name:     "retryable error",
			err:      errors.New("connection timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFatalError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "timeout",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "substring at beginning",
			s:        "timeout error",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "connection timeout",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "substring in middle",
			s:        "request timeout error",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "no match",
			s:        "connection refused",
			substr:   "timeout",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "test string",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "found at beginning",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "found in middle",
			s:        "hello world",
			substr:   "lo wo",
			expected: true,
		},
		{
			name:     "found at end",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "not found",
			s:        "hello world",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSubstring(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTTSHealthChecker(t *testing.T) {
	mockQueue := &MockMessageQueue{}
	manager := &GoogleTTSManager{
		client:       nil,
		messageQueue: mockQueue,
		voiceConfigs: make(map[string]TTSConfig),
	}

	checker := NewTTSHealthChecker(manager)

	assert.NotNil(t, checker)
	assert.Equal(t, manager, checker.manager)
	assert.Equal(t, time.Minute*5, checker.checkInterval)
	assert.Equal(t, "Health check test", checker.testText)
	assert.Equal(t, DefaultVoice, checker.testConfig.Voice)
	assert.Equal(t, float32(DefaultTTSSpeed), checker.testConfig.Speed)
	assert.Equal(t, float32(DefaultTTSVolume), checker.testConfig.Volume)
	assert.Equal(t, AudioFormatPCM, checker.testConfig.Format)
}

// Note: We don't test StartHealthCheck and performHealthCheck as they involve
// goroutines and actual TTS calls, which would require integration testing
// with proper Google Cloud credentials.
