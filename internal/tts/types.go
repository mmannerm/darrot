package tts

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// TTSConfig holds configuration for text-to-speech conversion
type TTSConfig struct {
	Voice  string      `json:"voice"`
	Speed  float32     `json:"speed"`
	Volume float32     `json:"volume"`
	Format AudioFormat `json:"format"`
}

// AudioFormat represents the audio format for TTS output
type AudioFormat string

const (
	AudioFormatOpus AudioFormat = "opus"
	AudioFormatDCA  AudioFormat = "dca"
	AudioFormatPCM  AudioFormat = "pcm"
)

// Voice represents a TTS voice option
type Voice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Gender   string `json:"gender"`
}

// VoiceConnection represents an active Discord voice connection
type VoiceConnection struct {
	GuildID    string                     `json:"guild_id"`
	ChannelID  string                     `json:"channel_id"`
	Connection *discordgo.VoiceConnection `json:"-"`
	IsPlaying  bool                       `json:"is_playing"`
	IsPaused   bool                       `json:"is_paused"`
	Queue      *AudioQueue                `json:"-"`
}

// AudioQueue manages queued audio for playback
type AudioQueue struct {
	Items   [][]byte `json:"-"`
	MaxSize int      `json:"max_size"`
	Current int      `json:"current"`
}

// ChannelPairing represents a voice-text channel pairing
type ChannelPairing struct {
	GuildID        string    `json:"guild_id"`
	VoiceChannelID string    `json:"voice_channel_id"`
	TextChannelID  string    `json:"text_channel_id"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// QueuedMessage represents a message queued for TTS processing
type QueuedMessage struct {
	ID        string    `json:"id"`
	GuildID   string    `json:"guild_id"`
	ChannelID string    `json:"channel_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GuildTTSConfig holds TTS configuration for a specific guild
type GuildTTSConfig struct {
	GuildID       string    `json:"guild_id"`
	RequiredRoles []string  `json:"required_roles"`
	TTSSettings   TTSConfig `json:"tts_settings"`
	MaxQueueSize  int       `json:"max_queue_size"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserTTSPreferences holds user-specific TTS preferences
type UserTTSPreferences struct {
	UserID    string          `json:"user_id"`
	GuildID   string          `json:"guild_id"`
	OptedIn   bool            `json:"opted_in"`
	Settings  UserTTSSettings `json:"settings"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// UserTTSSettings holds user-specific TTS settings
type UserTTSSettings struct {
	PreferredVoice string  `json:"preferred_voice"`
	SpeedModifier  float32 `json:"speed_modifier"`
}

// ChannelPairingStorage represents stored channel pairing data
type ChannelPairingStorage struct {
	GuildID        string    `json:"guild_id"`
	VoiceChannelID string    `json:"voice_channel_id"`
	TextChannelID  string    `json:"text_channel_id"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	IsActive       bool      `json:"is_active"`
}

// VoiceSession represents an active voice session with TTS
type VoiceSession struct {
	GuildID        string                     `json:"guild_id"`
	VoiceChannelID string                     `json:"voice_channel_id"`
	TextChannelID  string                     `json:"text_channel_id"`
	Connection     *discordgo.VoiceConnection `json:"-"`
	MessageQueue   MessageQueue               `json:"-"`
	IsPlaying      bool                       `json:"is_playing"`
	IsPaused       bool                       `json:"is_paused"`
	CreatedBy      string                     `json:"created_by"`
	CreatedAt      time.Time                  `json:"created_at"`
}
