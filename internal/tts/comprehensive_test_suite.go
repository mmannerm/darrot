package tts

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensiveTTSSuite runs the complete test suite for TTS functionality
func TestComprehensiveTTSSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive test suite in short mode")
	}

	t.Run("RequirementsValidation", func(t *testing.T) {
		testAllRequirementsValidation(t)
	})

	t.Run("IntegrationWorkflows", func(t *testing.T) {
		testIntegrationWorkflows(t)
	})

	t.Run("EndToEndScenarios", func(t *testing.T) {
		testEndToEndScenarios(t)
	})

	t.Run("PerformanceValidation", func(t *testing.T) {
		testPerformanceValidation(t)
	})

	t.Run("ErrorHandlingValidation", func(t *testing.T) {
		testErrorHandlingValidation(t)
	})

	t.Run("SystemReliability", func(t *testing.T) {
		testSystemReliability(t)
	})
}

// testAllRequirementsValidation validates that all requirements are met
func testAllRequirementsValidation(t *testing.T) {
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("Requirement_1_VoiceChannelInvitation", func(t *testing.T) {
		// Requirement 1: Voice channel invitation with text channel pairing
		guildID := "req1-guild"
		voiceChannelID := "req1-voice"
		textChannelID := "req1-text"
		userID := "req1-user"

		// Test 1.1: Invite with specified text channel
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		assert.NoError(t, err, "Requirement 1.1: Should join voice channel with specified text channel")

		// Verify pairing was created
		pairing, err := testEnv.channelService.GetPairing(guildID, voiceChannelID)
		assert.NoError(t, err)
		assert.Equal(t, textChannelID, pairing.TextChannelID, "Requirement 1.1: Should pair with specified text channel")

		// Test 1.3: Bot confirms presence
		assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Requirement 1.3: Bot should confirm presence in voice channel")

		// Clean up
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
		assert.NoError(t, err)
	})

	t.Run("Requirement_2_MessageReading", func(t *testing.T) {
		// Requirement 2: Message reading from opted-in users
		guildID := "req2-guild"
		voiceChannelID := "req2-voice"
		textChannelID := "req2-text"
		userID := "req2-user"

		// Set up environment
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		require.NoError(t, err)

		// Test 2.1: Monitor paired text channel
		isOptedIn, err := testEnv.userService.IsOptedIn(userID, guildID)
		assert.NoError(t, err)
		assert.True(t, isOptedIn, "Requirement 2.1: User who invites bot should be automatically opted in")

		// Test 2.2: Convert message to speech
		message := &QueuedMessage{
			ID:        "req2-msg",
			GuildID:   guildID,
			ChannelID: textChannelID,
			UserID:    userID,
			Username:  "TestUser",
			Content:   "Test message for requirement 2",
			Timestamp: time.Now(),
		}

		err = testEnv.messageQueue.Enqueue(message)
		assert.NoError(t, err, "Requirement 2.2: Should enqueue message from opted-in user")

		// Clean up
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
		assert.NoError(t, err)
	})

	t.Run("Requirement_3_BotControl", func(t *testing.T) {
		// Requirement 3: Bot control by authorized users
		guildID := "req3-guild"
		voiceChannelID := "req3-voice"
		textChannelID := "req3-text"
		userID := "req3-user"

		// Set up environment
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		require.NoError(t, err)

		// Test 3.1: Stop command
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
		assert.NoError(t, err, "Requirement 3.1: Authorized user should be able to stop bot")

		// Verify bot left
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Requirement 3.1: Bot should leave voice channel on stop")
	})

	t.Run("Requirement_6_UserOptIn", func(t *testing.T) {
		// Requirement 6: User opt-in functionality
		guildID := "req6-guild"
		userID := "req6-user"

		// Test 6.2: Opt-in command
		err := testEnv.userService.SetOptInStatus(userID, guildID, true)
		assert.NoError(t, err, "Requirement 6.2: User should be able to opt in")

		isOptedIn, err := testEnv.userService.IsOptedIn(userID, guildID)
		assert.NoError(t, err)
		assert.True(t, isOptedIn, "Requirement 6.2: User opt-in status should be tracked")

		// Test 6.3: Opt-out command
		err = testEnv.userService.SetOptInStatus(userID, guildID, false)
		assert.NoError(t, err, "Requirement 6.3: User should be able to opt out")

		isOptedIn, err = testEnv.userService.IsOptedIn(userID, guildID)
		assert.NoError(t, err)
		assert.False(t, isOptedIn, "Requirement 6.3: User opt-out status should be tracked")
	})
}

// testIntegrationWorkflows tests complete integration workflows
func testIntegrationWorkflows(t *testing.T) {
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("CompleteUserJourney", func(t *testing.T) {
		guildID := "journey-guild"
		voiceChannelID := "journey-voice"
		textChannelID := "journey-text"
		userID := "journey-user"

		// Step 1: User joins and invites bot
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		assert.NoError(t, err, "User should be able to invite bot")

		// Step 2: Verify automatic opt-in
		isOptedIn, err := testEnv.userService.IsOptedIn(userID, guildID)
		assert.NoError(t, err)
		assert.True(t, isOptedIn, "User should be automatically opted in")

		// Step 3: Send messages and verify processing
		for i := 0; i < 3; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("journey-msg-%d", i),
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    userID,
				Username:  "JourneyUser",
				Content:   fmt.Sprintf("Journey message %d", i),
				Timestamp: time.Now(),
			}
			err := testEnv.messageQueue.Enqueue(message)
			assert.NoError(t, err, "Should enqueue message %d", i)
		}

		// Step 4: Process messages
		err = testEnv.ttsManager.ProcessMessageQueue(guildID)
		assert.NoError(t, err, "Should process message queue")

		// Step 5: User opts out
		err = testEnv.userService.SetOptInStatus(userID, guildID, false)
		assert.NoError(t, err, "User should be able to opt out")

		// Step 6: Verify opted-out messages are not processed
		optedOutMessage := &QueuedMessage{
			ID:        "opted-out-msg",
			GuildID:   guildID,
			ChannelID: textChannelID,
			UserID:    userID,
			Username:  "JourneyUser",
			Content:   "This should not be processed",
			Timestamp: time.Now(),
		}
		shouldProcess := testEnv.shouldProcessMessage(optedOutMessage)
		assert.False(t, shouldProcess, "Opted-out user messages should not be processed")

		// Step 7: User leaves
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
		assert.NoError(t, err, "User should be able to make bot leave")
	})

	t.Run("MultiUserScenario", func(t *testing.T) {
		guildID := "multi-user-guild"
		voiceChannelID := "multi-user-voice"
		textChannelID := "multi-user-text"

		users := []string{"user1", "user2", "user3"}

		// User 1 invites bot
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, users[0])
		require.NoError(t, err, "First user should invite bot")

		// Other users opt in
		for _, userID := range users[1:] {
			err := testEnv.userService.SetOptInStatus(userID, guildID, true)
			assert.NoError(t, err, "User %s should be able to opt in", userID)
		}

		// All users send messages
		for i, userID := range users {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("multi-msg-%s", userID),
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    userID,
				Username:  fmt.Sprintf("User%d", i+1),
				Content:   fmt.Sprintf("Message from user %d", i+1),
				Timestamp: time.Now(),
			}
			err := testEnv.messageQueue.Enqueue(message)
			assert.NoError(t, err, "Should enqueue message from user %s", userID)
		}

		// Process all messages
		err = testEnv.ttsManager.ProcessMessageQueue(guildID)
		assert.NoError(t, err, "Should process all messages")

		// Clean up
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, users[0])
		assert.NoError(t, err, "Should leave voice channel")
	})
}

// testEndToEndScenarios tests end-to-end scenarios
func testEndToEndScenarios(t *testing.T) {
	testEnv := setupEndToEndTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("VoiceConnectionLifecycle", func(t *testing.T) {
		guildID := "e2e-lifecycle-guild"
		voiceChannelID := "e2e-lifecycle-voice"

		// Test connection establishment
		conn, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		assert.NoError(t, err, "Should establish voice connection")
		assert.NotNil(t, conn, "Connection should not be nil")

		// Test audio playback
		audioData := []byte("test audio for lifecycle")
		err = testEnv.voiceManager.PlayAudio(guildID, audioData)
		assert.NoError(t, err, "Should play audio")

		// Test connection cleanup
		err = testEnv.voiceManager.LeaveChannel(guildID)
		assert.NoError(t, err, "Should clean up connection")
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Should not be connected after cleanup")
	})

	t.Run("MessageProcessingPipeline", func(t *testing.T) {
		guildID := "e2e-pipeline-guild"
		textChannelID := "e2e-pipeline-text"
		userID := "e2e-pipeline-user"

		// Set up user
		err := testEnv.userService.SetOptInStatus(userID, guildID, true)
		require.NoError(t, err)

		// Create and process messages
		messages := []string{
			"Hello, this is the first message",
			"This is a second message with more content",
			"Final message in the pipeline test",
		}

		for i, content := range messages {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("pipeline-msg-%d", i),
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    userID,
				Username:  "PipelineUser",
				Content:   content,
				Timestamp: time.Now(),
			}

			// Verify message should be processed
			shouldProcess := testEnv.shouldProcessMessage(message)
			assert.True(t, shouldProcess, "Message %d should be processed", i)

			// Add to queue
			err := testEnv.messageQueue.Enqueue(message)
			assert.NoError(t, err, "Should enqueue message %d", i)
		}

		// Process entire pipeline
		err = testEnv.processMessageQueueWithAudio(guildID)
		assert.NoError(t, err, "Should process entire message pipeline")

		// Verify queue is empty
		queueSize := testEnv.messageQueue.Size(guildID)
		assert.Equal(t, 0, queueSize, "Queue should be empty after processing")
	})
}

// testPerformanceValidation tests performance requirements
func testPerformanceValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance validation in short mode")
	}

	t.Run("MessageThroughput", func(t *testing.T) {
		messageQueue := NewMessageQueue()
		guildID := "perf-throughput-guild"

		// Test message throughput
		numMessages := 1000
		startTime := time.Now()

		for i := 0; i < numMessages; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("throughput-msg-%d", i),
				GuildID:   guildID,
				ChannelID: "throughput-channel",
				UserID:    "throughput-user",
				Username:  "ThroughputUser",
				Content:   fmt.Sprintf("Throughput test message %d", i),
				Timestamp: time.Now(),
			}
			err := messageQueue.Enqueue(message)
			require.NoError(t, err)
		}

		enqueueTime := time.Since(startTime)
		throughput := float64(numMessages) / enqueueTime.Seconds()

		t.Logf("Message throughput: %.2f messages/second", throughput)
		assert.Greater(t, throughput, 100.0, "Should achieve at least 100 messages/second throughput")
	})

	t.Run("TTSConversionPerformance", func(t *testing.T) {
		manager := newMockTTSManagerIntegration()
		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 1.0,
			Format: AudioFormatDCA,
		}

		// Test TTS conversion performance
		numConversions := 100
		startTime := time.Now()

		for i := 0; i < numConversions; i++ {
			text := fmt.Sprintf("Performance test message %d", i)
			_, err := manager.ConvertToSpeech(text, config.Voice, config)
			require.NoError(t, err)
		}

		conversionTime := time.Since(startTime)
		conversionRate := float64(numConversions) / conversionTime.Seconds()

		t.Logf("TTS conversion rate: %.2f conversions/second", conversionRate)
		assert.Greater(t, conversionRate, 10.0, "Should achieve at least 10 conversions/second")
	})

	t.Run("MemoryUsageValidation", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Create test environment and perform operations
		testEnv := setupIntegrationTestEnvironment(t)
		defer testEnv.cleanup()

		// Perform memory-intensive operations
		guildID := "memory-test-guild"
		for i := 0; i < 1000; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("memory-msg-%d", i),
				GuildID:   guildID,
				ChannelID: "memory-channel",
				UserID:    "memory-user",
				Username:  "MemoryUser",
				Content:   fmt.Sprintf("Memory test message %d with additional content", i),
				Timestamp: time.Now(),
			}
			testEnv.messageQueue.Enqueue(message)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryUsed := m2.Alloc - m1.Alloc
		t.Logf("Memory used: %d bytes", memoryUsed)

		// Memory usage should be reasonable (less than 10MB for 1000 messages)
		assert.Less(t, memoryUsed, uint64(10*1024*1024), "Memory usage should be reasonable")
	})
}

// testErrorHandlingValidation tests error handling requirements
func testErrorHandlingValidation(t *testing.T) {
	testEnv := setupErrorTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("VoiceConnectionErrorHandling", func(t *testing.T) {
		guildID := "error-voice-guild"

		// Test connection failure handling
		testEnv.voiceManager.setConnectionError(guildID, fmt.Errorf("connection failed"))
		_, err := testEnv.voiceManager.JoinChannel(guildID, "test-channel")
		assert.Error(t, err, "Should handle connection failure")

		// Test error recovery
		err = testEnv.errorRecovery.HandleVoiceDisconnection(guildID)
		assert.NoError(t, err, "Should handle voice disconnection recovery")
	})

	t.Run("TTSErrorHandling", func(t *testing.T) {
		guildID := "error-tts-guild"
		text := "Error test message"
		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 1.0,
			Format: AudioFormatDCA,
		}

		// Test TTS failure handling
		testEnv.ttsManager.setConversionError(text, fmt.Errorf("TTS conversion failed"))
		_, err := testEnv.ttsManager.ConvertToSpeech(text, config.Voice, config)
		assert.Error(t, err, "Should handle TTS conversion failure")

		// Test error recovery
		_, err = testEnv.errorRecovery.HandleTTSFailure(text, config.Voice, config, guildID)
		// Recovery might fail, but should be handled gracefully
		if err != nil {
			assert.NotPanics(t, func() {
				testEnv.errorRecovery.CreateUserFriendlyErrorMessage(err, guildID)
			}, "Should create user-friendly error message")
		}
	})

	t.Run("PermissionErrorHandling", func(t *testing.T) {
		guildID := "error-permission-guild"
		userID := "error-permission-user"

		// Test unauthorized access
		testEnv.permissionService.setCanInviteBot(userID, guildID, false)
		err := testEnv.joinHandler.handleJoinCommand(guildID, "voice-channel", "text-channel", userID)
		assert.Error(t, err, "Should handle permission denial")
		assert.Contains(t, err.Error(), "cannot invite bot", "Error should indicate permission issue")
	})
}

// testSystemReliability tests overall system reliability
func testSystemReliability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reliability test in short mode")
	}

	t.Run("ExtendedOperationStability", func(t *testing.T) {
		testEnv := setupIntegrationTestEnvironment(t)
		defer testEnv.cleanup()

		guildID := "reliability-guild"
		voiceChannelID := "reliability-voice"
		textChannelID := "reliability-text"
		userID := "reliability-user"

		// Establish connection
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		require.NoError(t, err, "Should establish initial connection")

		// Run extended operations
		for cycle := 0; cycle < 10; cycle++ {
			// Add batch of messages
			for i := 0; i < 50; i++ {
				message := &QueuedMessage{
					ID:        fmt.Sprintf("reliability-msg-%d-%d", cycle, i),
					GuildID:   guildID,
					ChannelID: textChannelID,
					UserID:    userID,
					Username:  "ReliabilityUser",
					Content:   fmt.Sprintf("Reliability test cycle %d message %d", cycle, i),
					Timestamp: time.Now(),
				}
				err := testEnv.messageQueue.Enqueue(message)
				assert.NoError(t, err, "Should enqueue message in cycle %d", cycle)
			}

			// Process messages
			err := testEnv.ttsManager.ProcessMessageQueue(guildID)
			assert.NoError(t, err, "Should process messages in cycle %d", cycle)

			// Verify system state
			assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Should maintain connection in cycle %d", cycle)
			assert.Equal(t, 0, testEnv.messageQueue.Size(guildID), "Queue should be empty after cycle %d", cycle)
		}

		// Clean up
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
		assert.NoError(t, err, "Should clean up after extended operations")
	})

	t.Run("ConcurrentGuildReliability", func(t *testing.T) {
		testEnv := setupIntegrationTestEnvironment(t)
		defer testEnv.cleanup()

		numGuilds := 5
		messagesPerGuild := 20

		// Set up multiple guilds concurrently
		for i := 0; i < numGuilds; i++ {
			guildID := fmt.Sprintf("concurrent-guild-%d", i)
			voiceChannelID := fmt.Sprintf("concurrent-voice-%d", i)
			textChannelID := fmt.Sprintf("concurrent-text-%d", i)
			userID := fmt.Sprintf("concurrent-user-%d", i)

			err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
			assert.NoError(t, err, "Should set up guild %d", i)

			// Add messages for each guild
			for j := 0; j < messagesPerGuild; j++ {
				message := &QueuedMessage{
					ID:        fmt.Sprintf("concurrent-msg-%d-%d", i, j),
					GuildID:   guildID,
					ChannelID: textChannelID,
					UserID:    userID,
					Username:  fmt.Sprintf("User%d", i),
					Content:   fmt.Sprintf("Concurrent message %d from guild %d", j, i),
					Timestamp: time.Now(),
				}
				err := testEnv.messageQueue.Enqueue(message)
				assert.NoError(t, err, "Should enqueue message for guild %d", i)
			}
		}

		// Process all guilds
		for i := 0; i < numGuilds; i++ {
			guildID := fmt.Sprintf("concurrent-guild-%d", i)
			err := testEnv.ttsManager.ProcessMessageQueue(guildID)
			assert.NoError(t, err, "Should process messages for guild %d", i)
		}

		// Verify all guilds are still operational
		for i := 0; i < numGuilds; i++ {
			guildID := fmt.Sprintf("concurrent-guild-%d", i)
			assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Guild %d should still be connected", i)
			assert.Equal(t, 0, testEnv.messageQueue.Size(guildID), "Guild %d queue should be empty", i)
		}

		// Clean up all guilds
		for i := 0; i < numGuilds; i++ {
			guildID := fmt.Sprintf("concurrent-guild-%d", i)
			userID := fmt.Sprintf("concurrent-user-%d", i)
			err := testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
			assert.NoError(t, err, "Should clean up guild %d", i)
		}
	})
}
