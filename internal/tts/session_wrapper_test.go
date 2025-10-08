package tts

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestDiscordSessionWrapper_Creation(t *testing.T) {
	session := &discordgo.Session{}
	wrapper := NewDiscordSessionWrapper(session)

	assert.NotNil(t, wrapper)
	assert.Equal(t, session, wrapper.session)
}

func TestDiscordSessionWrapper_ImplementsInterface(t *testing.T) {
	session := &discordgo.Session{}
	wrapper := NewDiscordSessionWrapper(session)

	// Verify that the wrapper implements the DiscordSession interface
	var _ DiscordSession = wrapper

	assert.NotNil(t, wrapper)
}
