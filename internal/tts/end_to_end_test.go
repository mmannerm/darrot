package tts

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndVoiceConnection tests complete voice connection workflow
func TestEndToEndVoiceConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	testEnv := setupEndToEndTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "e2e-test-guild"
	voiceChannelID := "e2e-voice-channel"

	t.Run("CompleteVoiceConnectionLifecycle", func(t *testing.T) {
		// Step 1: Establish voice connection
		conn, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		require.NoError(t, err, "Should successfully join voice channel")
		require.NotNil(t, conn, "Voice connection should not be nil")

		// Verify connection properties
		assert.Equal(t, guildID, conn.GuildID)
		assert.Equal(t, voiceChannelID, conn.ChannelID)
		assert.False(t, conn.IsPlaying)

		// Step 2: Verify connection is active
		assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Voice connection should be active")

		// Step 3: Get connection details
		retrievedConn, exists := testEnv.voiceManager.GetConnection(guildID)
		assert.True(t, exists, "Connection should exist")
		assert.Equal(t, conn.GuildID, retrievedConn.GuildID)
		assert.Equal(t, conn.ChannelID, retrievedConn.ChannelID)

		// Step 4: Test audio playback
		audioData := []byte("test audio data for playback")
		err = testEnv.voiceManager.PlayAudio(guildID, audioData)
		assert.NoError(t, err, "Should successfully play audio")

		// Wait for audio to "play"
		time.Sleep(150 * time.Millisecond)

		// Step 5: Disconnect from voice channel
		err = testEnv.voiceManager.LeaveChannel(guildID)
		assert.NoError(t, err, "Should successfully leave voice channel")

		// Verify connection is closed
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Voice connection should be closed")
	})

	t.Run("VoiceConnectionErrorHandling", func(t *testing.T) {
		// Test playing audio without connection
		err := testEnv.voiceManager.PlayAudio("nonexistent-guild", []byte("test"))
		assert.Error(t, err, "Should fail to play audio without connection")

		// Test getting nonexistent connection
		_, exists := testEnv.voiceManager.GetConnection("nonexistent-guild")
		assert.False(t, exists, "Should not find nonexistent connection")

		// Test leaving nonexistent connection
		err = testEnv.voiceManager.LeaveChannel("nonexistent-guild")
		assert.NoError(t, err, "Should handle leaving nonexistent connection gracefully")
	})
}

// TestEndToEndAudioPlayback tests complete audio processing and playback
func TestEndToEndAudioPlayback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	testEnv := setupEndToEndTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "e2e-audio-guild"
	voiceChannelID := "e2e-audio-voice"
	textChannelID := "e2e-audio-text"

	t.Run("CompleteAudioProcessingPipeline", func(t *testing.T) {
		// Step 1: Set up voice connection
		_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		require.NoError(t, err, "Should join voice channel")

		// Step 2: Configure TTS settings
		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 0.8,
			Format: AudioFormatDCA,
		}
		err = testEnv.ttsManager.SetVoiceConfig(guildID, config)
		require.NoError(t, err, "Should set TTS configuration")

		// Step 3: Add messages to queue
		messages := []*QueuedMessage{
			{
				ID:        "audio-msg-1",
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    "user1",
				Username:  "TestUser1",
				Content:   "Hello, this is the first test message!",
				Timestamp: time.Now(),
			},
			{
				ID:        "audio-msg-2",
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    "user2",
				Username:  "TestUser2",
				Content:   "This is the second message for audio testing.",
				Timestamp: time.Now(),
			},
		}

		for _, msg := range messages {
			err := testEnv.messageQueue.Enqueue(msg)
			require.NoError(t, err, "Should enqueue message")
		}

		// Verify messages are queued
		queueSize := testEnv.messageQueue.Size(guildID)
		assert.Equal(t, len(messages), queueSize, "All messages should be queued")

		// Step 4: Process message queue with audio playback
		err = testEnv.processMessageQueueWithAudio(guildID)
		assert.NoError(t, err, "Should process message queue with audio")

		// Step 5: Verify queue is empty after processing
		finalQueueSize := testEnv.messageQueue.Size(guildID)
		assert.Equal(t, 0, finalQueueSize, "Queue should be empty after processing")

		// Step 6: Clean up
		err = testEnv.voiceManager.LeaveChannel(guildID)
		assert.NoError(t, err, "Should leave voice channel")
	})

	t.Run("AudioPlaybackWithDifferentFormats", func(t *testing.T) {
		formats := []AudioFormat{AudioFormatPCM, AudioFormatDCA, AudioFormatOpus}

		for _, format := range formats {
			t.Run(fmt.Sprintf("Format_%s", format), func(t *testing.T) {
				testGuildID := fmt.Sprintf("%s-%s", guildID, format)

				// Join voice channel
				_, err := testEnv.voiceManager.JoinChannel(testGuildID, voiceChannelID)
				require.NoError(t, err, "Should join voice channel")

				// Set TTS config with specific format
				config := TTSConfig{
					Voice:  "en-US-Standard-A",
					Speed:  1.0,
					Volume: 1.0,
					Format: format,
				}
				err = testEnv.ttsManager.SetVoiceConfig(testGuildID, config)
				require.NoError(t, err, "Should set TTS configuration")

				// Convert text to speech with specific format
				audioData, err := testEnv.ttsManager.ConvertToSpeech(
					"Test message for format "+string(format),
					config.Voice,
					config,
				)
				assert.NoError(t, err, "Should convert text to speech")
				assert.NotNil(t, audioData, "Audio data should not be nil")

				// Play audio
				err = testEnv.voiceManager.PlayAudio(testGuildID, audioData)
				assert.NoError(t, err, "Should play audio with format %s", format)

				// Clean up
				err = testEnv.voiceManager.LeaveChannel(testGuildID)
				assert.NoError(t, err, "Should leave voice channel")
			})
		}
	})
}

// TestEndToEndMessageProcessing tests complete message monitoring and processing
func TestEndToEndMessageProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	testEnv := setupEndToEndTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "e2e-message-guild"
	voiceChannelID := "e2e-message-voice"
	textChannelID := "e2e-message-text"

	t.Run("CompleteMessageProcessingWorkflow", func(t *testing.T) {
		// Step 1: Set up environment
		_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		require.NoError(t, err, "Should join voice channel")

		err = testEnv.channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
		require.NoError(t, err, "Should create channel pairing")

		// Step 2: Set up users with different opt-in status
		optedInUser := "opted-in-user"
		optedOutUser := "opted-out-user"

		err = testEnv.userService.SetOptInStatus(optedInUser, guildID, true)
		require.NoError(t, err, "Should opt in user")

		err = testEnv.userService.SetOptInStatus(optedOutUser, guildID, false)
		require.NoError(t, err, "Should opt out user")

		// Step 3: Simulate message monitoring
		messages := []*QueuedMessage{
			{
				ID:        "process-msg-1",
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    optedInUser,
				Username:  "OptedInUser",
				Content:   "This message should be processed",
				Timestamp: time.Now(),
			},
			{
				ID:        "process-msg-2",
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    optedOutUser,
				Username:  "OptedOutUser",
				Content:   "This message should be ignored",
				Timestamp: time.Now(),
			},
			{
				ID:        "process-msg-3",
				GuildID:   guildID,
				ChannelID: textChannelID,
				UserID:    optedInUser,
				Username:  "OptedInUser",
				Content:   "Another message that should be processed",
				Timestamp: time.Now(),
			},
		}

		// Step 4: Process messages based on opt-in status
		processedCount := 0
		for _, msg := range messages {
			shouldProcess := testEnv.shouldProcessMessage(msg)
			if shouldProcess {
				err := testEnv.messageQueue.Enqueue(msg)
				assert.NoError(t, err, "Should enqueue message from opted-in user")
				processedCount++
			}
		}

		// Should only process messages from opted-in user
		assert.Equal(t, 2, processedCount, "Should process 2 messages from opted-in user")
		assert.Equal(t, 2, testEnv.messageQueue.Size(guildID), "Queue should contain 2 messages")

		// Step 5: Process the queue
		err = testEnv.processMessageQueueWithAudio(guildID)
		assert.NoError(t, err, "Should process message queue")

		// Step 6: Verify queue is empty
		assert.Equal(t, 0, testEnv.messageQueue.Size(guildID), "Queue should be empty after processing")

		// Clean up
		err = testEnv.voiceManager.LeaveChannel(guildID)
		assert.NoError(t, err, "Should leave voice channel")
	})
}

// TestEndToEndConcurrentOperations tests concurrent operations across multiple guilds
func TestEndToEndConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	testEnv := setupEndToEndTestEnvironment(t)
	defer testEnv.cleanup()

	numGuilds := 5
	numMessagesPerGuild := 10

	t.Run("ConcurrentMultiGuildOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan error, numGuilds)

		// Start concurrent operations in multiple guilds
		for i := 0; i < numGuilds; i++ {
			wg.Add(1)
			go func(guildIndex int) {
				defer wg.Done()

				guildID := fmt.Sprintf("concurrent-guild-%d", guildIndex)
				voiceChannelID := fmt.Sprintf("concurrent-voice-%d", guildIndex)
				textChannelID := fmt.Sprintf("concurrent-text-%d", guildIndex)
				userID := fmt.Sprintf("concurrent-user-%d", guildIndex)

				// Join voice channel
				_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
				if err != nil {
					results <- fmt.Errorf("guild %d: failed to join voice channel: %v", guildIndex, err)
					return
				}

				// Set up user
				err = testEnv.userService.SetOptInStatus(userID, guildID, true)
				if err != nil {
					results <- fmt.Errorf("guild %d: failed to opt in user: %v", guildIndex, err)
					return
				}

				// Add multiple messages
				for j := 0; j < numMessagesPerGuild; j++ {
					message := &QueuedMessage{
						ID:        fmt.Sprintf("concurrent-msg-%d-%d", guildIndex, j),
						GuildID:   guildID,
						ChannelID: textChannelID,
						UserID:    userID,
						Username:  fmt.Sprintf("User%d", guildIndex),
						Content:   fmt.Sprintf("Concurrent message %d from guild %d", j, guildIndex),
						Timestamp: time.Now(),
					}

					err := testEnv.messageQueue.Enqueue(message)
					if err != nil {
						results <- fmt.Errorf("guild %d: failed to enqueue message %d: %v", guildIndex, j, err)
						return
					}
				}

				// Process messages
				err = testEnv.processMessageQueueWithAudio(guildID)
				if err != nil {
					results <- fmt.Errorf("guild %d: failed to process message queue: %v", guildIndex, err)
					return
				}

				// Verify queue is empty
				queueSize := testEnv.messageQueue.Size(guildID)
				if queueSize != 0 {
					results <- fmt.Errorf("guild %d: queue not empty after processing, size: %d", guildIndex, queueSize)
					return
				}

				// Leave voice channel
				err = testEnv.voiceManager.LeaveChannel(guildID)
				if err != nil {
					results <- fmt.Errorf("guild %d: failed to leave voice channel: %v", guildIndex, err)
					return
				}

				results <- nil
			}(i)
		}

		// Wait for all operations to complete
		wg.Wait()
		close(results)

		// Check results
		for err := range results {
			assert.NoError(t, err, "Concurrent operation should succeed")
		}
	})
}

// Test environment definitions moved to test_utils.go
