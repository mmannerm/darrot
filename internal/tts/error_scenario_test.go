package tts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBasicErrorScenarios tests basic error handling scenarios
func TestBasicErrorScenarios(t *testing.T) {
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("VoiceConnectionBasics", func(t *testing.T) {
		guildID := "test-guild"
		voiceChannelID := "voice-channel"

		// Test successful connection
		_, err := testEnv.voiceManager.JoinChannel(guildID, voiceChannelID)
		assert.NoError(t, err, "Should connect successfully")

		// Verify connection exists
		assert.True(t, testEnv.voiceManager.IsConnected(guildID), "Should be connected")

		// Test leaving channel
		err = testEnv.voiceManager.LeaveChannel(guildID)
		assert.NoError(t, err, "Should leave successfully")

		// Verify connection is gone
		assert.False(t, testEnv.voiceManager.IsConnected(guildID), "Should not be connected after leaving")
	})

	t.Run("PermissionBasics", func(t *testing.T) {
		guildID := "permission-guild"
		userID := "test-user"

		// Test default permissions (should be false)
		canInvite, err := testEnv.permissionService.CanInviteBot(userID, guildID)
		assert.NoError(t, err)
		assert.False(t, canInvite, "Should not have invite permission by default")

		// Set permission and test
		testEnv.permissionService.setCanInviteBot(userID, guildID, true)
		canInvite, err = testEnv.permissionService.CanInviteBot(userID, guildID)
		assert.NoError(t, err)
		assert.True(t, canInvite, "Should have invite permission after setting")
	})

	t.Run("TTSBasics", func(t *testing.T) {
		text := "Hello world"
		config := DefaultTTSConfig()

		// Test TTS conversion
		audioData, err := testEnv.ttsManager.ConvertToSpeech(text, config.Voice, config)
		assert.NoError(t, err, "Should convert text to speech")
		assert.NotNil(t, audioData, "Should return audio data")
		assert.Greater(t, len(audioData), 0, "Audio data should not be empty")
	})
}
