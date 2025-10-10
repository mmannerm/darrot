package tts

import (
	"github.com/bwmarrin/discordgo"
)

// DiscordSessionWrapper wraps a discordgo.Session to implement the DiscordSession interface
type DiscordSessionWrapper struct {
	session *discordgo.Session
}

// NewDiscordSessionWrapper creates a new wrapper for a Discord session
func NewDiscordSessionWrapper(session *discordgo.Session) *DiscordSessionWrapper {
	return &DiscordSessionWrapper{
		session: session,
	}
}

// Guild retrieves guild information
func (w *DiscordSessionWrapper) Guild(guildID string) (*discordgo.Guild, error) {
	return w.session.Guild(guildID)
}

// GuildMember retrieves guild member information
func (w *DiscordSessionWrapper) GuildMember(guildID, userID string) (*discordgo.Member, error) {
	return w.session.GuildMember(guildID, userID)
}

// Channel retrieves channel information
func (w *DiscordSessionWrapper) Channel(channelID string) (*discordgo.Channel, error) {
	return w.session.Channel(channelID)
}

// UserChannelPermissions retrieves user permissions for a channel
func (w *DiscordSessionWrapper) UserChannelPermissions(userID, channelID string) (int64, error) {
	return w.session.UserChannelPermissions(userID, channelID)
}
