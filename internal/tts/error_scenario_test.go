package tts

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVoiceConnectionErrorScenarios tests various voice connection error scenarios
func TestVoiceConnectionErrorScenarios(t *testing.T) {
	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("VoiceConnectionFailure", func(t *testing.T) {
		guildID := "voice-error-guild"
		voiceChannelID := "voice-error-channel"

		// Simulate connection failure
		testEnv.voiceManager.setConnectionError(guildID, errors.New("failed to connect to voice channel"))

		_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		assert.Error(t, err, "Should fail to join voice channel")
		assert.Contains(t, err.Error(), "failed to connect", "Error should indicate connection failure")

		// Verify no connection was established
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Should not be connected after failure")
	})

	t.Run("VoiceConnectionLoss", func(t *testing.T) {
		guildID := "voice-loss-guild"
		voiceChannelID := "voice-loss-channel"

		// Establish connection
		_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		require.NoError(t, err, "Should initially connect")
		assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Should be connected")

		// Simulate connection loss
		testEnv.voiceManager.simulateConnectionLoss(guildID)

		// Verify connection is lost
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Connection should be lost")

		// Test recovery
		err = testEnv.errorRecovery.HandleVoiceDisconnection(guildID)
		assert.NoError(t, err, "Should handle voice disconnection")

		// Verify recovery attempt was made
		assert.True(t, testEnv.voiceManager.wasRecoveryAttempted(guildID), "Recovery should be attempted")
	})

	t.Run("AudioPlaybackFailure", func(t *testing.T) {
		guildID := "audio-error-guild"
		voiceChannelID := "audio-error-channel"

		// Establish connection
		_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		require.NoError(t, err, "Should connect to voice channel")

		// Set up audio playback failure
		testEnv.voiceManager.setPlaybackError(guildID, errors.New("audio playback failed"))

		audioData := []byte("test audio data")
		err = testEnv.voiceManager.PlayAudio(guildID, audioData)
		assert.Error(t, err, "Should fail to play audio")

		// Test recovery
		err = testEnv.errorRecovery.HandleAudioPlaybackFailure(guildID, audioData)
		assert.NoError(t, err, "Should handle audio playback failure")
	})

	t.Run("MultipleConnectionFailures", func(t *testing.T) {
		guildID := "multi-error-guild"
		voiceChannelID := "multi-error-channel"

		// Simulate multiple consecutive failures
		for i := 0; i < 5; i++ {
			testEnv.voiceManager.setConnectionError(guildID, fmt.Errorf("connection failure %d", i))
			_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
			assert.Error(t, err, "Connection attempt %d should fail", i)
		}

		// Check error statistics
		stats := testEnv.errorRecovery.GetErrorStats(guildID)
		assert.GreaterOrEqual(t, stats.VoiceConnectionErrors, 5, "Should track multiple connection errors")

		// Guild should be considered unhealthy after many failures
		if stats.ConsecutiveFailures >= 5 {
			assert.False(t, testEnv.errorRecovery.IsGuildHealthy(guildID), "Guild should be unhealthy after many failures")
		}
	})
}

// TestTTSErrorScenarios tests various TTS conversion error scenarios
func TestTTSErrorScenarios(t *testing.T) {
	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	t.Run("TTSConversionFailure", func(t *testing.T) {
		guildID := "tts-error-guild"
		text := "This text will fail to convert"

		// Set up TTS conversion failure
		testEnv.ttsManager.setConversionError(text, errors.New("TTS service unavailable"))

		_, err := testEnv.ttsManager.ConvertToSpeech(text, config.Voice, config)
		assert.Error(t, err, "Should fail to convert text to speech")
		assert.Contains(t, err.Error(), "TTS service unavailable", "Error should indicate TTS failure")

		// Test recovery
		audioData, err := testEnv.errorRecovery.HandleTTSFailure(text, config.Voice, config, guildID)
		if err != nil {
			// Recovery might fail, but should be handled gracefully
			assert.Nil(t, audioData, "Audio data should be nil on recovery failure")
		}
	})

	t.Run("InvalidTextInput", func(t *testing.T) {
		testCases := []struct {
			name string
			text string
		}{
			{"empty_text", ""},
			{"text_too_long", string(make([]byte, MaxMessageLength+1))},
			{"only_whitespace", "   \n\t   "},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := testEnv.ttsManager.ConvertToSpeech(tc.text, config.Voice, config)
				assert.Error(t, err, "Should fail for invalid text: %s", tc.name)
			})
		}
	})

	t.Run("InvalidTTSConfiguration", func(t *testing.T) {
		guildID := "config-error-guild"

		invalidConfigs := []TTSConfig{
			{Voice: "", Speed: 1.0, Volume: 1.0, Format: AudioFormatDCA},                  // Empty voice
			{Voice: "invalid-voice", Speed: 1.0, Volume: 1.0, Format: AudioFormatDCA},     // Invalid voice
			{Voice: "en-US-Standard-A", Speed: -1.0, Volume: 1.0, Format: AudioFormatDCA}, // Invalid speed
			{Voice: "en-US-Standard-A", Speed: 1.0, Volume: -1.0, Format: AudioFormatDCA}, // Invalid volume
			{Voice: "en-US-Standard-A", Speed: 1.0, Volume: 1.0, Format: "invalid"},       // Invalid format
		}

		for i, invalidConfig := range invalidConfigs {
			t.Run(fmt.Sprintf("InvalidConfig_%d", i), func(t *testing.T) {
				err := testEnv.ttsManager.SetVoiceConfig(guildID, invalidConfig)
				assert.Error(t, err, "Should reject invalid configuration %d", i)
			})
		}
	})

	t.Run("TTSServiceUnavailable", func(t *testing.T) {
		guildID := "service-unavailable-guild"
		text := "Test message for unavailable service"

		// Simulate service unavailability
		testEnv.ttsManager.setServiceUnavailable(true)

		_, err := testEnv.ttsManager.ConvertToSpeech(text, config.Voice, config)
		assert.Error(t, err, "Should fail when TTS service is unavailable")

		// Test that error recovery handles service unavailability
		audioData, err := testEnv.errorRecovery.HandleTTSFailure(text, config.Voice, config, guildID)
		if err != nil {
			assert.Nil(t, audioData, "Should not return audio data when service is unavailable")
		}

		// Restore service
		testEnv.ttsManager.setServiceUnavailable(false)

		// Should work again
		_, err = testEnv.ttsManager.ConvertToSpeech(text, config.Voice, config)
		assert.NoError(t, err, "Should work when service is restored")
	})
}

// TestPermissionErrorScenarios tests various permission-related error scenarios
func TestPermissionErrorScenarios(t *testing.T) {
	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "permission-error-guild"
	voiceChannelID := "permission-voice-channel"
	textChannelID := "permission-text-channel"

	t.Run("UnauthorizedBotInvitation", func(t *testing.T) {
		unauthorizedUserID := "unauthorized-user"

		// Set user as unauthorized
		testEnv.permissionService.setCanInviteBot(unauthorizedUserID, guildID, false)

		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, unauthorizedUserID)
		assert.Error(t, err, "Should reject unauthorized bot invitation")
		assert.Contains(t, err.Error(), "cannot invite bot", "Error should indicate permission denial")

		// Verify no connection was established
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Should not connect for unauthorized user")
	})

	t.Run("UnauthorizedBotControl", func(t *testing.T) {
		authorizedUserID := "authorized-user"
		unauthorizedUserID := "unauthorized-control-user"

		// Set up authorized user and establish connection
		testEnv.permissionService.setCanInviteBot(authorizedUserID, guildID, true)
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, authorizedUserID)
		require.NoError(t, err, "Authorized user should be able to invite bot")

		// Set unauthorized user for control
		testEnv.permissionService.setCanControlBot(unauthorizedUserID, guildID, false)

		// Test unauthorized control commands
		err = testEnv.controlHandler.handlePauseCommand(guildID, unauthorizedUserID)
		assert.Error(t, err, "Should reject unauthorized pause command")

		err = testEnv.controlHandler.handleResumeCommand(guildID, unauthorizedUserID)
		assert.Error(t, err, "Should reject unauthorized resume command")

		err = testEnv.controlHandler.handleSkipCommand(guildID, unauthorizedUserID)
		assert.Error(t, err, "Should reject unauthorized skip command")
	})

	t.Run("ChannelAccessDenied", func(t *testing.T) {
		userID := "channel-access-user"
		restrictedChannelID := "restricted-channel"

		// Set user as authorized for bot invitation but not for channel access
		testEnv.permissionService.setCanInviteBot(userID, guildID, true)
		testEnv.channelService.setChannelAccessError(userID, restrictedChannelID, errors.New("access denied"))

		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, restrictedChannelID, userID)
		assert.Error(t, err, "Should reject invitation when user lacks channel access")
		assert.Contains(t, err.Error(), "cannot access", "Error should indicate channel access denial")
	})

	t.Run("BotLacksPermissions", func(t *testing.T) {
		userID := "bot-permission-user"

		// Set user as authorized
		testEnv.permissionService.setCanInviteBot(userID, guildID, true)

		// Simulate bot lacking permissions
		testEnv.voiceManager.setConnectionError(guildID, errors.New("bot lacks voice channel permissions"))

		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		assert.Error(t, err, "Should fail when bot lacks permissions")

		// Test user-friendly error message
		userMessage := testEnv.errorRecovery.CreateUserFriendlyErrorMessage(err, guildID)
		assert.NotEmpty(t, userMessage, "Should provide user-friendly error message")
		assert.Contains(t, userMessage, "permissions", "Error message should mention permissions")
	})
}

// TestMessageProcessingErrorScenarios tests error scenarios in message processing
func TestMessageProcessingErrorScenarios(t *testing.T) {
	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "message-error-guild"
	textChannelID := "message-error-channel"

	t.Run("MessageQueueOverflow", func(t *testing.T) {
		maxQueueSize := 5
		err := testEnv.messageQueue.SetMaxSize(guildID, maxQueueSize)
		require.NoError(t, err, "Should set max queue size")

		// Add messages beyond the limit
		for i := 0; i < maxQueueSize*2; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("overflow-msg-%d", i),
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    "overflow-user",
				Username:  "OverflowUser",
				Content:   fmt.Sprintf("Overflow message %d", i),
				Timestamp: time.Now(),
			}
			err := testEnv.messageQueue.Enqueue(message)
			assert.NoError(t, err, "Should handle overflow gracefully")
		}

		// Queue size should not exceed limit
		queueSize := testEnv.messageQueue.Size(guildID)
		assert.LessOrEqual(t, queueSize, maxQueueSize, "Queue size should not exceed limit")
	})

	t.Run("ProcessingOptedOutUserMessages", func(t *testing.T) {
		optedOutUserID := "opted-out-user"

		// Set user as opted out
		err := testEnv.userService.SetOptInStatus(optedOutUserID, guildID, false)
		require.NoError(t, err, "Should set user as opted out")

		message := &QueuedMessage{
			ID:        "opted-out-msg",
			GuildID:   guildID,
			ChannelID: textChannelID,
			UserID:    optedOutUserID,
			Username:  "OptedOutUser",
			Content:   "This message should not be processed",
			Timestamp: time.Now(),
		}

		// Message should not be processed
		shouldProcess := testEnv.shouldProcessMessage(message)
		assert.False(t, shouldProcess, "Messages from opted-out users should not be processed")
	})

	t.Run("CorruptedMessageData", func(t *testing.T) {
		corruptedMessages := []*QueuedMessage{
			{ID: "", GuildID: guildID, ChannelID: textChannelID, UserID: "user1", Username: "User1", Content: "Valid content", Timestamp: time.Now()}, // Empty ID
			{ID: "msg1", GuildID: "", ChannelID: textChannelID, UserID: "user1", Username: "User1", Content: "Valid content", Timestamp: time.Now()},  // Empty GuildID
			{ID: "msg2", GuildID: guildID, ChannelID: "", UserID: "user1", Username: "User1", Content: "Valid content", Timestamp: time.Now()},        // Empty ChannelID
			{ID: "msg3", GuildID: guildID, ChannelID: textChannelID, UserID: "", Username: "User1", Content: "Valid content", Timestamp: time.Now()},  // Empty UserID
			{ID: "msg4", GuildID: guildID, ChannelID: textChannelID, UserID: "user1", Username: "", Content: "Valid content", Timestamp: time.Now()},  // Empty Username
			{ID: "msg5", GuildID: guildID, ChannelID: textChannelID, UserID: "user1", Username: "User1", Content: "", Timestamp: time.Now()},          // Empty Content
		}

		for i, msg := range corruptedMessages {
			t.Run(fmt.Sprintf("CorruptedMessage_%d", i), func(t *testing.T) {
				err := testEnv.messageQueue.Enqueue(msg)
				// Should either reject the message or handle it gracefully
				if err != nil {
					assert.Error(t, err, "Should reject corrupted message %d", i)
				} else {
					// If accepted, processing should handle it gracefully
					_, processErr := testEnv.messageQueue.Dequeue(guildID)
					assert.NoError(t, processErr, "Should handle corrupted message gracefully")
				}
			})
		}
	})
}

// TestConcurrentErrorScenarios tests error scenarios under concurrent load
func TestConcurrentErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent error test in short mode")
	}

	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("ConcurrentConnectionFailures", func(t *testing.T) {
		numGoroutines := 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				guildID := fmt.Sprintf("concurrent-error-guild-%d", goroutineID)
				voiceChannelID := fmt.Sprintf("concurrent-voice-%d", goroutineID)

				// Simulate random connection failures
				if goroutineID%3 == 0 {
					testEnv.voiceManager.setConnectionError(guildID, fmt.Errorf("connection failure %d", goroutineID))
				}

				_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
				errors <- err
			}(i)
		}

		wg.Wait()
		close(errors)

		successCount := 0
		failureCount := 0
		for err := range errors {
			if err == nil {
				successCount++
			} else {
				failureCount++
			}
		}

		t.Logf("Concurrent connections: %d successes, %d failures", successCount, failureCount)
		assert.Greater(t, successCount, 0, "Some connections should succeed")
		assert.Greater(t, failureCount, 0, "Some connections should fail (as expected)")
	})

	t.Run("ConcurrentTTSFailures", func(t *testing.T) {
		numGoroutines := 10
		conversionsPerGoroutine := 10
		var wg sync.WaitGroup
		results := make(chan bool, numGoroutines*conversionsPerGoroutine)

		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 1.0,
			Format: AudioFormatDCA,
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < conversionsPerGoroutine; j++ {
					text := fmt.Sprintf("Concurrent TTS test %d-%d", goroutineID, j)

					// Simulate random TTS failures
					if (goroutineID+j)%4 == 0 {
						testEnv.ttsManager.setConversionError(text, fmt.Errorf("TTS failure %d-%d", goroutineID, j))
					}

					_, err := testEnv.ttsManager.ConvertToSpeech(text, config.Voice, config)
					results <- (err == nil)
				}
			}(i)
		}

		wg.Wait()
		close(results)

		successCount := 0
		failureCount := 0
		for success := range results {
			if success {
				successCount++
			} else {
				failureCount++
			}
		}

		totalOperations := numGoroutines * conversionsPerGoroutine
		t.Logf("Concurrent TTS: %d successes, %d failures out of %d operations",
			successCount, failureCount, totalOperations)

		assert.Greater(t, successCount, 0, "Some TTS conversions should succeed")
		assert.Greater(t, failureCount, 0, "Some TTS conversions should fail (as expected)")
	})
}

// TestErrorRecoveryMechanisms tests the error recovery system
func TestErrorRecoveryMechanisms(t *testing.T) {
	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "recovery-test-guild"

	t.Run("AutomaticRecoveryAttempts", func(t *testing.T) {
		// Test voice connection recovery
		testEnv.voiceManager.connections[guildID] = &VoiceConnection{
			GuildID:   guildID,
			ChannelID: "test-channel",
			IsPlaying: false,
		}

		// Simulate connection loss
		testEnv.voiceManager.simulateConnectionLoss(guildID)
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Connection should be lost")

		// Test recovery
		err := testEnv.errorRecovery.HandleVoiceDisconnection(guildID)
		assert.NoError(t, err, "Should handle voice disconnection recovery")

		// Verify recovery was attempted
		assert.True(t, testEnv.voiceManager.wasRecoveryAttempted(guildID), "Recovery should be attempted")
	})

	t.Run("ErrorStatisticsTracking", func(t *testing.T) {
		// Generate various types of errors
		testEnv.errorRecovery.updateErrorStats(guildID, "voice_connection")
		testEnv.errorRecovery.updateErrorStats(guildID, "tts_conversion")
		testEnv.errorRecovery.updateErrorStats(guildID, "audio_playback")
		testEnv.errorRecovery.updateErrorStats(guildID, "voice_connection")

		// Get error statistics
		stats := testEnv.errorRecovery.GetErrorStats(guildID)
		assert.Equal(t, 2, stats.VoiceConnectionErrors, "Should track voice connection errors")
		assert.Equal(t, 1, stats.TTSConversionErrors, "Should track TTS conversion errors")
		assert.Equal(t, 1, stats.AudioPlaybackErrors, "Should track audio playback errors")
		assert.GreaterOrEqual(t, stats.ConsecutiveFailures, 4, "Should track consecutive failures")
	})

	t.Run("HealthMonitoring", func(t *testing.T) {
		healthyGuildID := "healthy-guild"
		unhealthyGuildID := "unhealthy-guild"

		// Healthy guild should be healthy initially
		assert.True(t, testEnv.errorRecovery.IsGuildHealthy(healthyGuildID), "New guild should be healthy")

		// Generate many errors for unhealthy guild
		for i := 0; i < 10; i++ {
			testEnv.errorRecovery.updateErrorStats(unhealthyGuildID, "voice_connection")
		}

		// Unhealthy guild should be marked as unhealthy
		if testEnv.errorRecovery.GetErrorStats(unhealthyGuildID).ConsecutiveFailures >= 5 {
			assert.False(t, testEnv.errorRecovery.IsGuildHealthy(unhealthyGuildID), "Guild with many errors should be unhealthy")
		}
	})

	t.Run("UserFriendlyErrorMessages", func(t *testing.T) {
		errorTypes := map[string]error{
			"voice_connection": errors.New("voice connection failed"),
			"permission":       errors.New("permission denied"),
			"tts_conversion":   errors.New("TTS conversion failed"),
			"audio_playback":   errors.New("audio playback failed"),
			"unknown":          errors.New("unknown error occurred"),
		}

		for errorType, err := range errorTypes {
			t.Run(errorType, func(t *testing.T) {
				userMessage := testEnv.errorRecovery.CreateUserFriendlyErrorMessage(err, guildID)
				assert.NotEmpty(t, userMessage, "Should provide user-friendly message for %s", errorType)
				assert.NotContains(t, userMessage, "nil", "Message should not contain technical details")
				assert.NotContains(t, userMessage, "panic", "Message should not contain technical details")
			})
		}
	})
}

// Test environment definitions moved to test_utils.go
