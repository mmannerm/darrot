package tts

// TTSManager handles text-to-speech conversion and audio processing
type TTSManager interface {
	ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error)
	ProcessMessageQueue(guildID string) error
	SetVoiceConfig(guildID string, config TTSConfig) error
	GetSupportedVoices() []Voice
}

// VoiceManager manages Discord voice connections and audio streaming
type VoiceManager interface {
	JoinChannel(guildID, channelID string) (*VoiceConnection, error)
	LeaveChannel(guildID string) error
	GetConnection(guildID string) (*VoiceConnection, bool)
	PlayAudio(guildID string, audioData []byte) error
	IsConnected(guildID string) bool
	PausePlayback(guildID string) error
	ResumePlayback(guildID string) error
	SkipCurrentMessage(guildID string) error
	IsPaused(guildID string) bool
	RecoverConnection(guildID string) error
	HealthCheck() map[string]error
	SetConnectionStateCallback(callback func(guildID string, connected bool))
	Cleanup() error
	GetActiveConnections() []string
}

// ChannelService manages voice-text channel pairings and monitoring
type ChannelService interface {
	CreatePairing(guildID, voiceChannelID, textChannelID string) error
	CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error
	RemovePairing(guildID, voiceChannelID string) error
	GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error)
	ValidateChannelAccess(userID, channelID string) error
	IsChannelPaired(guildID, textChannelID string) bool
}

// PermissionService handles role-based access control and user permissions
type PermissionService interface {
	CanInviteBot(userID, guildID string) (bool, error)
	CanControlBot(userID, guildID string) (bool, error)
	HasChannelAccess(userID, channelID string) (bool, error)
	SetRequiredRoles(guildID string, roleIDs []string) error
	GetRequiredRoles(guildID string) ([]string, error)
}

// UserService manages user opt-in preferences and settings
type UserService interface {
	SetOptInStatus(userID, guildID string, optedIn bool) error
	IsOptedIn(userID, guildID string) (bool, error)
	GetOptedInUsers(guildID string) ([]string, error)
	AutoOptIn(userID, guildID string) error // For bot inviters
}

// MessageQueue handles queuing and processing of text messages for TTS conversion
type MessageQueue interface {
	Enqueue(message *QueuedMessage) error
	Dequeue(guildID string) (*QueuedMessage, error)
	Clear(guildID string) error
	Size(guildID string) int
	SetMaxSize(guildID string, size int) error
	SkipNext(guildID string) (*QueuedMessage, error)
}

// ConfigService manages guild TTS configuration settings
type ConfigService interface {
	GetGuildConfig(guildID string) (*GuildTTSConfig, error)
	SetGuildConfig(guildID string, config *GuildTTSConfig) error
	SetRequiredRoles(guildID string, roleIDs []string) error
	GetRequiredRoles(guildID string) ([]string, error)
	SetTTSSettings(guildID string, settings TTSConfig) error
	GetTTSSettings(guildID string) (*TTSConfig, error)
	SetMaxQueueSize(guildID string, size int) error
	GetMaxQueueSize(guildID string) (int, error)
	ValidateConfig(config *GuildTTSConfig) error
}

// TTSProcessor handles the background processing pipeline for TTS conversion and playback
type TTSProcessor interface {
	Start() error
	Stop() error
	StartGuildProcessing(guildID string) error
	StopGuildProcessing(guildID string) error
	GetProcessingStatus(guildID string) (bool, error)
	GetActiveGuilds() []string
	SkipCurrentMessage(guildID string) error
	PauseProcessing(guildID string) error
	ResumeProcessing(guildID string) error
	ClearQueue(guildID string) error
	GetQueueSize(guildID string) int
}
