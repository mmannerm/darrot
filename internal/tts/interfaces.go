package tts

import (
	"time"
)

// ChannelService manages voice-text channel pairings and monitoring
type ChannelService interface {
	CreatePairing(guildID, voiceChannelID, textChannelID string) error
	RemovePairing(guildID, voiceChannelID string) error
	GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error)
	ValidateChannelAccess(userID, channelID string) error
	IsChannelPaired(guildID, textChannelID string) (bool, error)
}

// UserService manages user opt-in preferences and settings
type UserService interface {
	SetOptInStatus(userID, guildID string, optedIn bool) error
	IsOptedIn(userID, guildID string) (bool, error)
	GetOptedInUsers(guildID string) ([]string, error)
	AutoOptIn(userID, guildID string) error // For bot inviters
}

// MessageQueue manages queuing and processing of text messages for TTS conversion
type MessageQueue interface {
	Enqueue(message *QueuedMessage) error
	Dequeue(guildID string) (*QueuedMessage, error)
	Clear(guildID string) error
	Size(guildID string) int
	SetMaxSize(guildID string, size int) error
}

// ChannelPairing represents a voice-text channel pairing
type ChannelPairing struct {
	GuildID        string
	VoiceChannelID string
	TextChannelID  string
	CreatedBy      string
	CreatedAt      time.Time
}

// QueuedMessage represents a message queued for TTS processing
type QueuedMessage struct {
	ID        string
	GuildID   string
	ChannelID string
	UserID    string
	Username  string
	Content   string
	Timestamp time.Time
}
