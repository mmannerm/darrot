package tts

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// mockChannelService implements ChannelService for testing
type mockChannelService struct {
	pairedChannels map[string]bool // textChannelID -> isPaired
}

func newMockChannelService() *mockChannelService {
	return &mockChannelService{
		pairedChannels: make(map[string]bool),
	}
}

func (m *mockChannelService) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	return nil
}

func (m *mockChannelService) CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error {
	return nil
}

func (m *mockChannelService) RemovePairing(guildID, voiceChannelID string) error {
	return nil
}

func (m *mockChannelService) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	return nil, nil
}

func (m *mockChannelService) ValidateChannelAccess(userID, channelID string) error {
	return nil
}

func (m *mockChannelService) IsChannelPaired(guildID, textChannelID string) bool {
	return m.pairedChannels[textChannelID]
}

func (m *mockChannelService) setPaired(textChannelID string, paired bool) {
	m.pairedChannels[textChannelID] = paired
}

// mockUserService implements UserService for testing
type mockUserService struct {
	optedInUsers map[string]bool // "userID:guildID" -> optedIn
}

func newMockUserService() *mockUserService {
	return &mockUserService{
		optedInUsers: make(map[string]bool),
	}
}

func (m *mockUserService) SetOptInStatus(userID, guildID string, optedIn bool) error {
	key := userID + ":" + guildID
	m.optedInUsers[key] = optedIn
	return nil
}

func (m *mockUserService) IsOptedIn(userID, guildID string) (bool, error) {
	key := userID + ":" + guildID
	return m.optedInUsers[key], nil
}

func (m *mockUserService) GetOptedInUsers(guildID string) ([]string, error) {
	return nil, nil
}

func (m *mockUserService) AutoOptIn(userID, guildID string) error {
	return m.SetOptInStatus(userID, guildID, true)
}

func (m *mockUserService) setOptedIn(userID, guildID string, optedIn bool) {
	key := userID + ":" + guildID
	m.optedInUsers[key] = optedIn
}

// mockMessageQueue implements MessageQueue for testing
type mockMessageQueue struct {
	messages []QueuedMessage
}

func newMockMessageQueue() *mockMessageQueue {
	return &mockMessageQueue{
		messages: make([]QueuedMessage, 0),
	}
}

func (m *mockMessageQueue) Enqueue(message *QueuedMessage) error {
	m.messages = append(m.messages, *message)
	return nil
}

func (m *mockMessageQueue) Dequeue(guildID string) (*QueuedMessage, error) {
	return nil, nil
}

func (m *mockMessageQueue) Clear(guildID string) error {
	return nil
}

func (m *mockMessageQueue) Size(guildID string) int {
	return len(m.messages)
}

func (m *mockMessageQueue) SetMaxSize(guildID string, size int) error {
	return nil
}

func (m *mockMessageQueue) SkipNext(guildID string) (*QueuedMessage, error) {
	return nil, nil
}

func (m *mockMessageQueue) getMessages() []QueuedMessage {
	return m.messages
}

func (m *mockMessageQueue) reset() {
	m.messages = make([]QueuedMessage, 0)
}

func TestMessageMonitor_handleMessageCreate(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	channelService := newMockChannelService()
	userService := newMockUserService()
	messageQueue := newMockMessageQueue()

	// Create a mock Discord session (we don't need a real connection for this test)
	session := &discordgo.Session{}

	monitor := NewMessageMonitor(session, channelService, userService, messageQueue, logger)

	tests := []struct {
		name            string
		message         *discordgo.MessageCreate
		channelPaired   bool
		userOptedIn     bool
		expectQueued    bool
		expectedContent string
	}{
		{
			name: "Valid message from opted-in user in paired channel",
			message: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg1",
					Content:   "Hello world!",
					GuildID:   "guild1",
					ChannelID: "channel1",
					Author: &discordgo.User{
						ID:       "user1",
						Username: "TestUser",
						Bot:      false,
					},
				},
			},
			channelPaired:   true,
			userOptedIn:     true,
			expectQueued:    true,
			expectedContent: "TestUser says: Hello world!",
		},
		{
			name: "Message from bot should be ignored",
			message: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg2",
					Content:   "Bot message",
					GuildID:   "guild1",
					ChannelID: "channel1",
					Author: &discordgo.User{
						ID:       "bot1",
						Username: "BotUser",
						Bot:      true,
					},
				},
			},
			channelPaired: true,
			userOptedIn:   true,
			expectQueued:  false,
		},
		{
			name: "Message from unpaired channel should be ignored",
			message: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg3",
					Content:   "Unpaired channel message",
					GuildID:   "guild1",
					ChannelID: "channel2",
					Author: &discordgo.User{
						ID:       "user1",
						Username: "TestUser",
						Bot:      false,
					},
				},
			},
			channelPaired: false,
			userOptedIn:   true,
			expectQueued:  false,
		},
		{
			name: "Message from opted-out user should be ignored",
			message: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg4",
					Content:   "Opted out user message",
					GuildID:   "guild1",
					ChannelID: "channel1",
					Author: &discordgo.User{
						ID:       "user2",
						Username: "OptedOutUser",
						Bot:      false,
					},
				},
			},
			channelPaired: true,
			userOptedIn:   false,
			expectQueued:  false,
		},
		{
			name: "Empty message should be ignored",
			message: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID:        "msg5",
					Content:   "   ",
					GuildID:   "guild1",
					ChannelID: "channel1",
					Author: &discordgo.User{
						ID:       "user1",
						Username: "TestUser",
						Bot:      false,
					},
				},
			},
			channelPaired: true,
			userOptedIn:   true,
			expectQueued:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset message queue
			messageQueue.reset()

			// Set up mock services
			channelService.setPaired(tt.message.ChannelID, tt.channelPaired)
			userService.setOptedIn(tt.message.Author.ID, tt.message.GuildID, tt.userOptedIn)

			// Process the message
			monitor.handleMessageCreate(session, tt.message)

			// Check if message was queued
			messages := messageQueue.getMessages()
			if tt.expectQueued {
				if len(messages) != 1 {
					t.Errorf("Expected 1 message to be queued, got %d", len(messages))
					return
				}

				queuedMsg := messages[0]
				if queuedMsg.ID != tt.message.ID {
					t.Errorf("Expected message ID %s, got %s", tt.message.ID, queuedMsg.ID)
				}
				if queuedMsg.GuildID != tt.message.GuildID {
					t.Errorf("Expected guild ID %s, got %s", tt.message.GuildID, queuedMsg.GuildID)
				}
				if queuedMsg.ChannelID != tt.message.ChannelID {
					t.Errorf("Expected channel ID %s, got %s", tt.message.ChannelID, queuedMsg.ChannelID)
				}
				if queuedMsg.UserID != tt.message.Author.ID {
					t.Errorf("Expected user ID %s, got %s", tt.message.Author.ID, queuedMsg.UserID)
				}
				if queuedMsg.Username != tt.message.Author.Username {
					t.Errorf("Expected username %s, got %s", tt.message.Author.Username, queuedMsg.Username)
				}
				if queuedMsg.Content != tt.expectedContent {
					t.Errorf("Expected content %s, got %s", tt.expectedContent, queuedMsg.Content)
				}
			} else {
				if len(messages) != 0 {
					t.Errorf("Expected no messages to be queued, got %d", len(messages))
				}
			}
		})
	}
}

func TestMessageMonitor_preprocessMessage(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	session := &discordgo.Session{}

	monitor := NewMessageMonitor(session, newMockChannelService(), newMockUserService(), newMockMessageQueue(), logger)

	tests := []struct {
		name     string
		content  string
		username string
		expected string
	}{
		{
			name:     "Simple message",
			content:  "Hello world!",
			username: "TestUser",
			expected: "TestUser says: Hello world!",
		},
		{
			name:     "Message with custom emoji",
			content:  "Hello <:custom:123456789>!",
			username: "TestUser",
			expected: "TestUser says: Hello custom emoji!",
		},
		{
			name:     "Message with animated emoji",
			content:  "Hello <a:animated:123456789>!",
			username: "TestUser",
			expected: "TestUser says: Hello animated emoji!",
		},
		{
			name:     "Message with extra whitespace",
			content:  "  Hello world!  ",
			username: "TestUser",
			expected: "TestUser says: Hello world!",
		},
		{
			name:     "Long message should be truncated",
			content:  strings.Repeat("a", 600),
			username: "TestUser",
			expected: "", // We'll check this separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.preprocessMessage(tt.content, tt.username)

			if tt.name == "Long message should be truncated" {
				// Special handling for truncation test
				if !strings.HasPrefix(result, "TestUser says: ") {
					t.Errorf("Expected result to start with 'TestUser says: ', got %s", result)
				}
				if !strings.HasSuffix(result, "...") {
					t.Errorf("Expected result to end with '...', got %s", result)
				}
				if len(result) != 500 {
					t.Errorf("Expected result length to be 500, got %d", len(result))
				}
			} else if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMessageMonitor_handleEmojis(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	session := &discordgo.Session{}

	monitor := NewMessageMonitor(session, newMockChannelService(), newMockUserService(), newMockMessageQueue(), logger)

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Custom emoji replacement",
			content:  "Hello <:smile:123456789>!",
			expected: "Hello smile emoji!",
		},
		{
			name:     "Animated emoji replacement",
			content:  "Hello <a:dance:123456789>!",
			expected: "Hello dance emoji!",
		},
		{
			name:     "Multiple custom emojis",
			content:  "Hello <:smile:123> and <:wave:456>!",
			expected: "Hello smile emoji and wave emoji!",
		},
		{
			name:     "No emojis",
			content:  "Hello world!",
			expected: "Hello world!",
		},
		{
			name:     "Mixed content with emojis",
			content:  "Check this out <:cool:789> very nice!",
			expected: "Check this out cool emoji very nice!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.handleEmojis(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMessageMonitor_IsMonitoring(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	// Test with session
	session := &discordgo.Session{}
	monitor := NewMessageMonitor(session, newMockChannelService(), newMockUserService(), newMockMessageQueue(), logger)

	if !monitor.IsMonitoring() {
		t.Error("Expected IsMonitoring to return true when session is set")
	}

	// Test without session
	monitor.session = nil
	if monitor.IsMonitoring() {
		t.Error("Expected IsMonitoring to return false when session is nil")
	}
}
