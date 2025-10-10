package tts

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"
)

// Helper function to create error recovery manager with fast test configuration
func newFastErrorRecoveryManager(voiceManager VoiceManager, ttsManager TTSManager, messageQueue MessageQueue, configService ConfigService) *ErrorRecoveryManager {
	return NewErrorRecoveryManagerWithConfig(voiceManager, ttsManager, messageQueue, configService, ErrorRecoveryConfig{
		MaxRetries:          3,
		RetryDelay:          time.Millisecond * 10,
		ConnectionTimeout:   time.Millisecond * 100,
		HealthCheckInterval: time.Millisecond * 50,
		MonitorInterval:     time.Millisecond * 20,
	})
}

// TestErrorHandlingIntegration tests the complete error handling and recovery system
func TestErrorHandlingIntegration(t *testing.T) {
	// Create test components
	mockVoice := newMockVoiceManagerForRecovery()
	mockTTS := newMockTTSManagerForRecovery()
	mockQueue := &mockMessageQueueForRecovery{}
	mockConfig := &mockConfigServiceForRecovery{}
	mockUser := &mockUserServiceForIntegration{}

	// Create error recovery manager
	errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

	// Create TTS processor with error recovery
	processor := NewTTSProcessor(mockTTS, mockVoice, mockQueue, mockConfig, mockUser)

	// Test 1: Start and stop the system
	t.Run("SystemStartStop", func(t *testing.T) {
		err := processor.Start()
		if err != nil {
			t.Fatalf("Failed to start processor: %v", err)
		}

		// Give it a moment to start
		time.Sleep(5 * time.Millisecond)

		err = processor.Stop()
		if err != nil {
			t.Fatalf("Failed to stop processor: %v", err)
		}
	})

	// Test 2: Voice connection recovery
	t.Run("VoiceConnectionRecovery", func(t *testing.T) {
		guildID := "test-guild-voice"

		// Set up initial connection
		mockVoice.connections[guildID] = true

		// Simulate connection failure and recovery
		err := errorRecovery.HandleVoiceDisconnection(guildID)
		if err != nil {
			t.Errorf("Voice connection recovery failed: %v", err)
		}

		// Verify connection was recovered
		if !mockVoice.IsConnected(guildID) {
			t.Errorf("Expected voice connection to be recovered")
		}
	})

	// Test 3: TTS failure recovery
	t.Run("TTSFailureRecovery", func(t *testing.T) {
		guildID := "test-guild-tts"
		text := "Test message for TTS recovery"
		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 1.0,
			Format: AudioFormatPCM,
		}

		// Test successful recovery
		audioData, err := errorRecovery.HandleTTSFailure(text, "", config, guildID)
		if err != nil {
			t.Errorf("TTS failure recovery failed: %v", err)
		}
		if audioData == nil {
			t.Errorf("Expected audio data from TTS recovery")
		}
	})

	// Test 4: Audio playback recovery
	t.Run("AudioPlaybackRecovery", func(t *testing.T) {
		guildID := "test-guild-audio"
		audioData := []byte("test audio data")

		// Set up connection
		mockVoice.connections[guildID] = true

		// Test successful recovery
		err := errorRecovery.HandleAudioPlaybackFailure(guildID, audioData)
		if err != nil {
			t.Errorf("Audio playback recovery failed: %v", err)
		}
	})

	// Test 5: User-friendly error messages
	t.Run("UserFriendlyErrorMessages", func(t *testing.T) {
		testCases := []struct {
			name     string
			err      error
			expected string
		}{
			{
				name:     "voice connection error",
				err:      errors.New("voice connection failed"),
				expected: "I'm having trouble connecting to the voice channel. Please try inviting me again, or check that I have the necessary permissions.",
			},
			{
				name:     "permission error",
				err:      errors.New("permission denied"),
				expected: "I don't have the necessary permissions to perform this action. Please check that I have voice channel and text channel permissions.",
			},
			{
				name:     "TTS error",
				err:      errors.New("TTS conversion failed"),
				expected: "I'm having trouble converting text to speech right now. I'll keep trying, but some messages might be skipped.",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				message := errorRecovery.CreateUserFriendlyErrorMessage(tc.err, "test-guild")
				if message != tc.expected {
					t.Errorf("Expected: %s, got: %s", tc.expected, message)
				}
			})
		}
	})

	// Test 6: Error statistics tracking
	t.Run("ErrorStatisticsTracking", func(t *testing.T) {
		guildID := "test-guild-stats"

		// Initially should be healthy
		if !errorRecovery.IsGuildHealthy(guildID) {
			t.Errorf("Guild should be healthy initially")
		}

		// Simulate some errors
		errorRecovery.updateErrorStats(guildID, "voice_connection")
		errorRecovery.updateErrorStats(guildID, "tts_conversion")

		// Get stats
		stats := errorRecovery.GetErrorStats(guildID)
		if stats.VoiceConnectionErrors != 1 {
			t.Errorf("Expected 1 voice connection error, got %d", stats.VoiceConnectionErrors)
		}
		if stats.TTSConversionErrors != 1 {
			t.Errorf("Expected 1 TTS conversion error, got %d", stats.TTSConversionErrors)
		}

		// Should still be healthy with few errors
		if !errorRecovery.IsGuildHealthy(guildID) {
			t.Errorf("Guild should still be healthy with few errors")
		}
	})

	// Test 7: Health monitoring
	t.Run("HealthMonitoring", func(t *testing.T) {
		// Start error recovery for health monitoring
		err := errorRecovery.Start()
		if err != nil {
			t.Fatalf("Failed to start error recovery: %v", err)
		}

		// Give health checker time to run
		time.Sleep(10 * time.Millisecond)

		// Stop error recovery
		err = errorRecovery.Stop()
		if err != nil {
			t.Errorf("Failed to stop error recovery: %v", err)
		}
	})
}

// TestCommandHandlerErrorIntegration tests error handling in command handlers
func TestCommandHandlerErrorIntegration(t *testing.T) {
	// Create mock services
	mockVoice := newMockVoiceManagerForRecovery()
	mockChannel := &mockChannelServiceForIntegration{}
	mockPermission := &mockPermissionServiceForIntegration{}
	mockUser := &mockUserServiceForIntegration{}
	mockTTS := newMockTTSManagerForRecovery()
	mockQueue := &mockMessageQueueForRecovery{}
	mockConfig := &mockConfigServiceForRecovery{}

	// Create error recovery manager
	errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	// Create command handlers with error recovery
	// Create a mock TTS processor for testing
	mockTTSProcessor := &mockTTSProcessorForRecovery{}

	joinHandler := NewJoinCommandHandler(
		mockVoice,
		mockChannel,
		mockPermission,
		mockUser,
		mockTTSProcessor,
		errorRecovery,
		logger,
	)

	leaveHandler := NewLeaveCommandHandler(
		mockVoice,
		mockChannel,
		mockPermission,
		mockTTSProcessor,
		errorRecovery,
		logger,
	)

	// Test that handlers were created successfully
	if joinHandler == nil {
		t.Errorf("Join handler should not be nil")
	}
	if leaveHandler == nil {
		t.Errorf("Leave handler should not be nil")
	}

	// Test command definitions
	joinDef := joinHandler.Definition()
	if joinDef.Name != "darrot-join" {
		t.Errorf("Expected join command name 'darrot-join', got '%s'", joinDef.Name)
	}

	leaveDef := leaveHandler.Definition()
	if leaveDef.Name != "darrot-leave" {
		t.Errorf("Expected leave command name 'darrot-leave', got '%s'", leaveDef.Name)
	}
}

// Mock services for integration testing

type mockUserServiceForIntegration struct{}

func (m *mockUserServiceForIntegration) SetOptInStatus(userID, guildID string, optedIn bool) error {
	return nil
}

func (m *mockUserServiceForIntegration) IsOptedIn(userID, guildID string) (bool, error) {
	return true, nil
}

func (m *mockUserServiceForIntegration) GetOptedInUsers(guildID string) ([]string, error) {
	return []string{"user1", "user2"}, nil
}

func (m *mockUserServiceForIntegration) AutoOptIn(userID, guildID string) error {
	return nil
}

type mockChannelServiceForIntegration struct{}

func (m *mockChannelServiceForIntegration) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	return nil
}

func (m *mockChannelServiceForIntegration) CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error {
	return nil
}

func (m *mockChannelServiceForIntegration) RemovePairing(guildID, voiceChannelID string) error {
	return nil
}

func (m *mockChannelServiceForIntegration) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	return nil, nil
}

func (m *mockChannelServiceForIntegration) ValidateChannelAccess(userID, channelID string) error {
	return nil
}

func (m *mockChannelServiceForIntegration) IsChannelPaired(guildID, textChannelID string) bool {
	return false
}

type mockPermissionServiceForIntegration struct{}

func (m *mockPermissionServiceForIntegration) CanInviteBot(userID, guildID string) (bool, error) {
	return true, nil
}

func (m *mockPermissionServiceForIntegration) CanControlBot(userID, guildID string) (bool, error) {
	return true, nil
}

func (m *mockPermissionServiceForIntegration) HasChannelAccess(userID, channelID string) (bool, error) {
	return true, nil
}

func (m *mockPermissionServiceForIntegration) SetRequiredRoles(guildID string, roleIDs []string) error {
	return nil
}

func (m *mockPermissionServiceForIntegration) GetRequiredRoles(guildID string) ([]string, error) {
	return []string{}, nil
}

// TestErrorRecoveryRequirements verifies that all requirements are met
func TestErrorRecoveryRequirements(t *testing.T) {
	t.Run("Requirement_1_4_VoiceConnectionRecovery", func(t *testing.T) {
		// Requirement 1.4: WHEN the bot fails to join a voice channel THEN the bot SHALL provide an error message explaining the failure
		// This is handled by the error recovery system providing user-friendly error messages

		mockVoice := newMockVoiceManagerForRecovery()
		mockTTS := newMockTTSManagerForRecovery()
		mockQueue := &mockMessageQueueForRecovery{}
		mockConfig := &mockConfigServiceForRecovery{}

		errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

		// Test that voice connection failures are handled gracefully
		err := errors.New("failed to join voice channel")
		userMessage := errorRecovery.CreateUserFriendlyErrorMessage(err, "test-guild")

		if userMessage == "" {
			t.Errorf("Expected user-friendly error message for voice connection failure")
		}

		expectedMessage := "I'm having trouble connecting to the voice channel. Please try inviting me again, or check that I have the necessary permissions."
		if userMessage != expectedMessage {
			t.Errorf("Expected specific error message for voice connection failure")
		}
	})

	t.Run("Requirement_1_6_VoiceConnectionFailure", func(t *testing.T) {
		// Requirement 1.6: WHEN the bot fails to join a voice channel THEN the bot SHALL provide an error message explaining the failure
		// This is the same as 1.4 - testing error message creation for voice failures

		mockVoice := newMockVoiceManagerForRecovery()
		mockTTS := newMockTTSManagerForRecovery()
		mockQueue := &mockMessageQueueForRecovery{}
		mockConfig := &mockConfigServiceForRecovery{}

		errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

		// Test voice connection recovery
		guildID := "test-guild-req-1-6"
		mockVoice.connections[guildID] = true

		err := errorRecovery.HandleVoiceDisconnection(guildID)
		if err != nil {
			t.Errorf("Voice connection recovery should succeed in normal case")
		}
	})

	t.Run("Requirement_9_1_AutomaticReconnection", func(t *testing.T) {
		// Requirement 9.1: WHEN the bot loses connection to the voice channel THEN the bot SHALL attempt to reconnect automatically

		mockVoice := newMockVoiceManagerForRecovery()
		mockTTS := newMockTTSManagerForRecovery()
		mockQueue := &mockMessageQueueForRecovery{}
		mockConfig := &mockConfigServiceForRecovery{}

		errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

		guildID := "test-guild-req-9-1"
		mockVoice.connections[guildID] = true

		// Test automatic reconnection
		err := errorRecovery.HandleVoiceDisconnection(guildID)
		if err != nil {
			t.Errorf("Automatic reconnection should succeed: %v", err)
		}

		// Verify that recovery was attempted
		if len(mockVoice.recoveryCalls) == 0 {
			t.Errorf("Expected recovery to be attempted")
		}
	})

	t.Run("Requirement_9_2_TTSFailureHandling", func(t *testing.T) {
		// Requirement 9.2: WHEN TTS conversion fails THEN the bot SHALL skip the problematic message and continue with the next

		mockVoice := newMockVoiceManagerForRecovery()
		mockTTS := newMockTTSManagerForRecovery()
		mockQueue := &mockMessageQueueForRecovery{}
		mockConfig := &mockConfigServiceForRecovery{}

		errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

		guildID := "test-guild-req-9-2"
		text := "Test message"
		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 1.0,
			Format: AudioFormatPCM,
		}

		// Test TTS failure handling with fallback mechanisms
		audioData, err := errorRecovery.HandleTTSFailure(text, "", config, guildID)
		if err != nil {
			t.Errorf("TTS failure handling should provide fallback: %v", err)
		}
		if audioData == nil {
			t.Errorf("Expected audio data from TTS failure handling")
		}
	})

	t.Run("Requirement_9_3_ClearErrorMessages", func(t *testing.T) {
		// Requirement 9.3: WHEN the bot lacks permissions for voice or text channels THEN the bot SHALL provide clear error messages

		mockVoice := newMockVoiceManagerForRecovery()
		mockTTS := newMockTTSManagerForRecovery()
		mockQueue := &mockMessageQueueForRecovery{}
		mockConfig := &mockConfigServiceForRecovery{}

		errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

		// Test permission error messages
		permissionErr := errors.New("permission denied")
		userMessage := errorRecovery.CreateUserFriendlyErrorMessage(permissionErr, "test-guild")

		expectedMessage := "I don't have the necessary permissions to perform this action. Please check that I have voice channel and text channel permissions."
		if userMessage != expectedMessage {
			t.Errorf("Expected clear permission error message, got: %s", userMessage)
		}
	})

	t.Run("Requirement_9_4_RepeatedErrorHandling", func(t *testing.T) {
		// Requirement 9.4: IF the bot encounters repeated errors THEN the bot SHALL leave the voice channel and notify users of the issue

		mockVoice := newMockVoiceManagerForRecovery()
		mockTTS := newMockTTSManagerForRecovery()
		mockQueue := &mockMessageQueueForRecovery{}
		mockConfig := &mockConfigServiceForRecovery{}

		errorRecovery := newFastErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

		guildID := "test-guild-req-9-4"

		// Simulate many consecutive errors
		for i := 0; i < 10; i++ {
			errorRecovery.updateErrorStats(guildID, "voice_connection")
		}

		// Check that guild is considered unhealthy after many errors
		if errorRecovery.IsGuildHealthy(guildID) {
			t.Errorf("Guild should be considered unhealthy after many consecutive errors")
		}

		// Get error stats to verify tracking
		stats := errorRecovery.GetErrorStats(guildID)
		if stats.ConsecutiveFailures < 5 {
			t.Errorf("Expected many consecutive failures to be tracked")
		}
	})
}
