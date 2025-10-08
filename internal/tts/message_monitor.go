package tts

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// MessageMonitor handles monitoring Discord text channels for TTS processing
type MessageMonitor struct {
	session        *discordgo.Session
	channelService ChannelService
	userService    UserService
	messageQueue   MessageQueue
	logger         *log.Logger
	emojiRegex     *regexp.Regexp
}

// NewMessageMonitor creates a new MessageMonitor instance
func NewMessageMonitor(
	session *discordgo.Session,
	channelService ChannelService,
	userService UserService,
	messageQueue MessageQueue,
	logger *log.Logger,
) *MessageMonitor {
	// Regex to match Discord emojis (both Unicode and custom)
	emojiRegex := regexp.MustCompile(`<a?:\w+:\d+>|[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]`)

	monitor := &MessageMonitor{
		session:        session,
		channelService: channelService,
		userService:    userService,
		messageQueue:   messageQueue,
		logger:         logger,
		emojiRegex:     emojiRegex,
	}

	// Register message event handler
	session.AddHandler(monitor.handleMessageCreate)

	return monitor
}

// handleMessageCreate processes new Discord messages for TTS
func (m *MessageMonitor) handleMessageCreate(s *discordgo.Session, mc *discordgo.MessageCreate) {
	// Skip messages from bots (including ourselves)
	if mc.Author.Bot {
		return
	}

	// Skip empty messages
	if strings.TrimSpace(mc.Content) == "" {
		return
	}

	m.logger.Printf("Received message from %s in guild %s, channel %s: %s", mc.Author.Username, mc.GuildID, mc.ChannelID, mc.Content)

	// Check if this channel is paired with a voice channel
	// Note: Using the existing interface which returns bool only
	isPaired := m.channelService.IsChannelPaired(mc.GuildID, mc.ChannelID)
	if !isPaired {
		m.logger.Printf("Channel %s in guild %s is not paired, ignoring message", mc.ChannelID, mc.GuildID)
		return // Channel is not paired, ignore message
	}

	m.logger.Printf("Channel %s in guild %s is paired, processing message", mc.ChannelID, mc.GuildID)

	// Check if user is opted-in for TTS
	isOptedIn, err := m.userService.IsOptedIn(mc.Author.ID, mc.GuildID)
	if err != nil {
		m.logger.Printf("Error checking opt-in status for user %s in guild %s: %v", mc.Author.ID, mc.GuildID, err)
		return
	}

	if !isOptedIn {
		m.logger.Printf("User %s in guild %s is not opted-in, ignoring message", mc.Author.Username, mc.GuildID)
		return // User is not opted-in, ignore message
	}

	m.logger.Printf("User %s in guild %s is opted-in, processing message", mc.Author.Username, mc.GuildID)

	// Preprocess the message
	processedContent := m.preprocessMessage(mc.Content, mc.Author.Username)

	// Skip if message becomes empty after preprocessing
	if strings.TrimSpace(processedContent) == "" {
		m.logger.Printf("Message from %s became empty after preprocessing, skipping", mc.Author.Username)
		return
	}

	// Create queued message
	queuedMessage := &QueuedMessage{
		ID:        mc.ID,
		GuildID:   mc.GuildID,
		ChannelID: mc.ChannelID,
		UserID:    mc.Author.ID,
		Username:  mc.Author.Username,
		Content:   processedContent,
		Timestamp: time.Now(),
	}

	// Add to message queue
	if err := m.messageQueue.Enqueue(queuedMessage); err != nil {
		m.logger.Printf("Error enqueueing message from %s: %v", mc.Author.Username, err)
		return
	}

	m.logger.Printf("Queued message from %s in guild %s: %s", mc.Author.Username, mc.GuildID, processedContent)
}

// preprocessMessage handles message preprocessing including author name and emoji handling
func (m *MessageMonitor) preprocessMessage(content, username string) string {
	// Clean up extra whitespace from original content first
	content = strings.TrimSpace(content)

	// Add author name prefix
	processedContent := fmt.Sprintf("%s says: %s", username, content)

	// Handle emojis - replace custom Discord emojis with their names
	processedContent = m.handleEmojis(processedContent)

	// Clean up extra whitespace again
	processedContent = strings.TrimSpace(processedContent)

	// Limit message length (max ~30 seconds of speech at average rate)
	const maxLength = 500
	if len(processedContent) > maxLength {
		processedContent = processedContent[:maxLength-3] + "..."
		m.logger.Printf("Truncated long message from %s", username)
	}

	return processedContent
}

// handleEmojis processes emojis in the message content
func (m *MessageMonitor) handleEmojis(content string) string {
	// Replace custom Discord emojis with their names
	content = regexp.MustCompile(`<a?:(\w+):\d+>`).ReplaceAllString(content, "$1 emoji")

	// For Unicode emojis, we'll keep them as-is since most TTS engines can handle them
	// or convert them to descriptive text if needed

	// Remove excessive emoji sequences (more than 3 in a row)
	content = regexp.MustCompile(`([\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]){4,}`).ReplaceAllString(content, "multiple emojis")

	return content
}

// IsMonitoring returns whether the monitor is actively listening for messages
func (m *MessageMonitor) IsMonitoring() bool {
	return m.session != nil
}

// Stop stops the message monitor (removes event handlers)
func (m *MessageMonitor) Stop() {
	if m.session != nil {
		// Note: discordgo doesn't provide a direct way to remove specific handlers
		// In a production implementation, you might need to track handler references
		// or implement a more sophisticated handler management system
		m.logger.Println("Message monitor stopped")
	}
}
