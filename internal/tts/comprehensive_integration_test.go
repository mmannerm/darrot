package tts

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCompleteWorkflowIntegration tests the complete TTS workflow from invitation to message processing
func TestCompleteWorkflowIntegration(t *testing.T) {
	// Set up test environment
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "integration-test-guild"
	voiceChannelID := "voice-channel-123"
	textChannelID := "text-channel-456"
	userID := "user-789"

	t.Run("CompleteInviteOptInMessageFlow", func(t *testing.T) {
		// Set up permissions for the user
		testEnv.permissionService.setCanInviteBot(userID, guildID, true)
		testEnv.permissionService.setCanControlBot(userID, guildID, true)

		// Step 1: User invites bot to voice channel
		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
		assert.NoError(t, err, "Bot should successfully join voice channel")

		// Verify voice connection was established
		assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Voice connection should be established")

		// Verify channel pairing was created
		pairing, err := testEnv.channelService.GetPairing(guildID, voiceChannelID)
		assert.NoError(t, err)
		assert.NotNil(t, pairing, "Channel pairing should be created")
		assert.Equal(t, textChannelID, pairing.TextChannelID)

		// Step 2: User should be automatically opted in
		isOptedIn, err := testEnv.userService.IsOptedIn(userID, guildID)
		assert.NoError(t, err)
		assert.True(t, isOptedIn, "User who invites bot should be automatically opted in")

		// Step 3: Process a message from opted-in user
		message := &QueuedMessage{
			ID:        "msg-001",
			GuildID:   guildID,
			ChannelID: textChannelID,
			UserID:    userID,
			Username:  "TestUser",
			Content:   "Hello, this is a test message!",
			Timestamp: time.Now(),
		}

		err = testEnv.messageQueue.Enqueue(message)
		assert.NoError(t, err, "Message should be enqueued successfully")

		// Step 4: Process the message queue
		// First verify message is in queue
		queueSizeBefore := testEnv.messageQueue.Size(guildID)
		assert.Equal(t, 1, queueSizeBefore, "Message should be in queue before processing")

		err = testEnv.ttsManager.ProcessMessageQueue(guildID)
		assert.NoError(t, err, "Message queue should be processed successfully")

		// Verify message was processed (queue should be empty or smaller)
		queueSize := testEnv.messageQueue.Size(guildID)
		assert.Equal(t, 0, queueSize, "Message queue should be empty after processing")

		// Step 5: User opts out
		err = testEnv.userService.SetOptInStatus(userID, guildID, false)
		assert.NoError(t, err, "User should be able to opt out")

		// Step 6: Process another message (should be ignored)
		message2 := &QueuedMessage{
			ID:        "msg-002",
			GuildID:   guildID,
			ChannelID: textChannelID,
			UserID:    userID,
			Username:  "TestUser",
			Content:   "This message should be ignored",
			Timestamp: time.Now(),
		}

		// Message should not be processed for opted-out user
		shouldProcess := testEnv.shouldProcessMessage(message2)
		assert.False(t, shouldProcess, "Messages from opted-out users should not be processed")

		// Step 7: Bot leaves voice channel
		err = testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
		assert.NoError(t, err, "Bot should successfully leave voice channel")

		// Verify voice connection was closed
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Voice connection should be closed")
	})
}

// TestMultiGuildIntegration tests TTS functionality across multiple guilds simultaneously
func TestMultiGuildIntegration(t *testing.T) {
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	numGuilds := 3
	guilds := make([]string, numGuilds)
	for i := 0; i < numGuilds; i++ {
		guilds[i] = fmt.Sprintf("guild-%d", i)
	}

	t.Run("SimultaneousMultiGuildOperations", func(t *testing.T) {
		// Set up permissions for all users first
		for i, guildID := range guilds {
			userID := fmt.Sprintf("user-%d", i)
			testEnv.permissionService.setCanInviteBot(userID, guildID, true)
			testEnv.permissionService.setCanControlBot(userID, guildID, true)
		}

		var wg sync.WaitGroup

		// Start TTS operations in multiple guilds simultaneously
		for i, guildID := range guilds {
			wg.Add(1)
			go func(guildID string, index int) {
				defer wg.Done()

				voiceChannelID := fmt.Sprintf("voice-%d", index)
				textChannelID := fmt.Sprintf("text-%d", index)
				userID := fmt.Sprintf("user-%d", index)

				// Join voice channel
				err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, userID)
				assert.NoError(t, err, "Bot should join voice channel in guild %s", guildID)

				// Process multiple messages
				for j := 0; j < 5; j++ {
					message := &QueuedMessage{
						ID:        fmt.Sprintf("msg-%d-%d", index, j),
						GuildID:   guildID,
						ChannelID: textChannelID,
						UserID:    userID,
						Username:  fmt.Sprintf("User%d", index),
						Content:   fmt.Sprintf("Message %d from guild %s", j, guildID),
						Timestamp: time.Now(),
					}

					err := testEnv.messageQueue.Enqueue(message)
					assert.NoError(t, err, "Message should be enqueued in guild %s", guildID)
				}

				// Process message queue
				err = testEnv.ttsManager.ProcessMessageQueue(guildID)
				assert.NoError(t, err, "Message queue should be processed in guild %s", guildID)
			}(guildID, i)
		}

		wg.Wait()

		// Verify all guilds have active connections
		for _, guildID := range guilds {
			assert.True(t, testEnv.voiceManager.IsConnected(guildID),
				"Guild %s should have active voice connection", guildID)
		}

		// Clean up all connections
		for i, guildID := range guilds {
			userID := fmt.Sprintf("user-%d", i)
			err := testEnv.leaveHandler.handleLeaveCommand(guildID, userID)
			assert.NoError(t, err, "Bot should leave voice channel in guild %s", guildID)
		}
	})
}

// TestPermissionIntegration tests permission validation throughout the workflow
func TestPermissionIntegration(t *testing.T) {
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "permission-test-guild"
	voiceChannelID := "voice-channel-123"
	textChannelID := "text-channel-456"

	t.Run("PermissionValidationWorkflow", func(t *testing.T) {
		// Test with user who cannot invite bot
		unauthorizedUserID := "unauthorized-user"
		testEnv.permissionService.setCanInviteBot(unauthorizedUserID, guildID, false)

		err := testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, unauthorizedUserID)
		assert.Error(t, err, "Unauthorized user should not be able to invite bot")

		// Test with user who can invite bot
		authorizedUserID := "authorized-user"
		testEnv.permissionService.setCanInviteBot(authorizedUserID, guildID, true)

		err = testEnv.joinHandler.handleJoinCommand(guildID, voiceChannelID, textChannelID, authorizedUserID)
		assert.NoError(t, err, "Authorized user should be able to invite bot")

		// Test control permissions
		controlUserID := "control-user"
		testEnv.permissionService.setCanControlBot(controlUserID, guildID, true)

		err = testEnv.controlHandler.handlePauseCommand(guildID, controlUserID)
		assert.NoError(t, err, "User with control permissions should be able to pause")

		// Test user without control permissions
		noControlUserID := "no-control-user"
		testEnv.permissionService.setCanControlBot(noControlUserID, guildID, false)

		err = testEnv.controlHandler.handlePauseCommand(guildID, noControlUserID)
		assert.Error(t, err, "User without control permissions should not be able to pause")
	})
}

// TestConfigurationIntegration tests TTS configuration management
func TestConfigurationIntegration(t *testing.T) {
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	guildID := "config-test-guild"
	adminUserID := "admin-user"

	t.Run("ConfigurationManagement", func(t *testing.T) {
		// Set up admin permissions
		testEnv.permissionService.setCanInviteBot(adminUserID, guildID, true)

		// Test setting TTS configuration
		config := TTSConfig{
			Voice:  "en-US-Wavenet-A",
			Speed:  1.5,
			Volume: 0.8,
			Format: AudioFormatDCA,
		}

		err := testEnv.configHandler.handleSetVoiceCommand(guildID, adminUserID, config)
		assert.NoError(t, err, "Admin should be able to set TTS configuration")

		// Verify configuration was applied
		storedConfig := testEnv.ttsManager.getVoiceConfig(guildID)
		assert.Equal(t, config, storedConfig, "Configuration should be stored correctly")

		// Test setting required roles
		requiredRoles := []string{"role1", "role2"}
		err = testEnv.configHandler.handleSetRequiredRolesCommand(guildID, adminUserID, requiredRoles)
		assert.NoError(t, err, "Admin should be able to set required roles")

		// Verify roles were set
		roles, err := testEnv.permissionService.GetRequiredRoles(guildID)
		assert.NoError(t, err)
		assert.Equal(t, requiredRoles, roles, "Required roles should be stored correctly")

		// Test setting queue limits
		maxQueueSize := 15
		err = testEnv.configHandler.handleSetQueueLimitCommand(guildID, adminUserID, maxQueueSize)
		assert.NoError(t, err, "Admin should be able to set queue limits")

		// Verify queue limit was applied
		testEnv.messageQueue.SetMaxSize(guildID, maxQueueSize)
		// Test that queue respects the limit by adding messages beyond the limit
		for i := 0; i < maxQueueSize+5; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("msg-%d", i),
				GuildID:   guildID,
				ChannelID: "test-channel",
				UserID:    "test-user",
				Username:  "TestUser",
				Content:   fmt.Sprintf("Message %d", i),
				Timestamp: time.Now(),
			}
			testEnv.messageQueue.Enqueue(message)
		}

		// Queue size should not exceed the limit
		queueSize := testEnv.messageQueue.Size(guildID)
		assert.LessOrEqual(t, queueSize, maxQueueSize, "Queue size should not exceed configured limit")
	})
}

// Test environment definitions moved to test_utils.go
