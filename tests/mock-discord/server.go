package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// MockDiscordServer represents the mock Discord API server
type MockDiscordServer struct {
	guilds       map[string]*Guild
	users        map[string]*User
	channels     map[string]*Channel
	interactions []Interaction
	mu           sync.RWMutex
}

// Guild represents a Discord guild (server)
type Guild struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Channels map[string]*Channel `json:"channels"`
	Members  map[string]*User    `json:"members"`
}

// Channel represents a Discord channel
type Channel struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     ChannelType `json:"type"`
	GuildID  string      `json:"guild_id"`
	Messages []Message   `json:"messages,omitempty"`
}

// ChannelType represents Discord channel types
type ChannelType int

const (
	ChannelTypeText  ChannelType = 0
	ChannelTypeVoice ChannelType = 2
)

// User represents a Discord user
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Bot           bool   `json:"bot"`
}

// Message represents a Discord message
type Message struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Author    *User     `json:"author"`
	ChannelID string    `json:"channel_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Interaction represents a Discord interaction (slash command)
type Interaction struct {
	ID      string                 `json:"id"`
	Type    int                    `json:"type"`
	Data    map[string]interface{} `json:"data"`
	User    *User                  `json:"user"`
	GuildID string                 `json:"guild_id"`
}

// NewMockDiscordServer creates a new mock Discord server
func NewMockDiscordServer() *MockDiscordServer {
	server := &MockDiscordServer{
		guilds:       make(map[string]*Guild),
		users:        make(map[string]*User),
		channels:     make(map[string]*Channel),
		interactions: make([]Interaction, 0),
	}

	// Create default test data
	server.setupTestData()

	return server
}

// setupTestData creates default guilds, channels, and users for testing
func (s *MockDiscordServer) setupTestData() {
	// Create test guild
	guild := &Guild{
		ID:       "test-guild-123",
		Name:     "Test Guild",
		Channels: make(map[string]*Channel),
		Members:  make(map[string]*User),
	}

	// Create test channels
	textChannel := &Channel{
		ID:       "test-text-channel-456",
		Name:     "general",
		Type:     ChannelTypeText,
		GuildID:  guild.ID,
		Messages: make([]Message, 0),
	}

	voiceChannel := &Channel{
		ID:      "test-voice-channel-789",
		Name:    "General Voice",
		Type:    ChannelTypeVoice,
		GuildID: guild.ID,
	}

	// Create test user
	testUser := &User{
		ID:            "test-user-456",
		Username:      "testuser",
		Discriminator: "1234",
		Bot:           false,
	}

	// Create bot user
	botUser := &User{
		ID:            "bot-user-789",
		Username:      "darrot",
		Discriminator: "0000",
		Bot:           true,
	}

	// Add to collections
	guild.Channels[textChannel.ID] = textChannel
	guild.Channels[voiceChannel.ID] = voiceChannel
	guild.Members[testUser.ID] = testUser
	guild.Members[botUser.ID] = botUser

	s.guilds[guild.ID] = guild
	s.channels[textChannel.ID] = textChannel
	s.channels[voiceChannel.ID] = voiceChannel
	s.users[testUser.ID] = testUser
	s.users[botUser.ID] = botUser
}

// SetupRoutes configures the HTTP routes for the mock Discord API
func (s *MockDiscordServer) SetupRoutes(router *mux.Router) {
	// API v10 routes
	api := router.PathPrefix("/api/v10").Subrouter()

	// Authentication middleware
	api.Use(s.authMiddleware)

	// Guild endpoints
	api.HandleFunc("/guilds/{guildId}", s.getGuild).Methods("GET")
	api.HandleFunc("/guilds/{guildId}/channels", s.getGuildChannels).Methods("GET")
	api.HandleFunc("/guilds/{guildId}/members/{userId}", s.getGuildMember).Methods("GET")

	// Channel endpoints
	api.HandleFunc("/channels/{channelId}", s.getChannel).Methods("GET")
	api.HandleFunc("/channels/{channelId}/messages", s.getChannelMessages).Methods("GET")
	api.HandleFunc("/channels/{channelId}/messages", s.createMessage).Methods("POST")

	// Voice endpoints
	api.HandleFunc("/channels/{channelId}/voice-states/@me", s.updateVoiceState).Methods("PATCH")

	// User endpoints
	api.HandleFunc("/users/@me", s.getCurrentUser).Methods("GET")
	api.HandleFunc("/users/{userId}", s.getUser).Methods("GET")

	// Interaction endpoints
	api.HandleFunc("/interactions/{interactionId}/{interactionToken}/callback", s.interactionCallback).Methods("POST")

	// Health check endpoint
	router.HandleFunc("/health", s.healthCheck).Methods("GET")
}

// authMiddleware simulates Discord API authentication
func (s *MockDiscordServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bot ") {
			http.Error(w, `{"message": "401: Unauthorized", "code": 0}`, http.StatusUnauthorized)
			return
		}

		// Extract token (in real implementation, validate against known tokens)
		token := strings.TrimPrefix(auth, "Bot ")
		if token == "" {
			http.Error(w, `{"message": "401: Unauthorized", "code": 0}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getGuild handles GET /guilds/{guildId}
func (s *MockDiscordServer) getGuild(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	guildID := vars["guildId"]

	s.mu.RLock()
	guild, exists := s.guilds[guildID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, `{"message": "Unknown Guild", "code": 10004}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(guild)
}

// getGuildChannels handles GET /guilds/{guildId}/channels
func (s *MockDiscordServer) getGuildChannels(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	guildID := vars["guildId"]

	s.mu.RLock()
	guild, exists := s.guilds[guildID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, `{"message": "Unknown Guild", "code": 10004}`, http.StatusNotFound)
		return
	}

	channels := make([]*Channel, 0, len(guild.Channels))
	for _, channel := range guild.Channels {
		channels = append(channels, channel)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// getGuildMember handles GET /guilds/{guildId}/members/{userId}
func (s *MockDiscordServer) getGuildMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	guildID := vars["guildId"]
	userID := vars["userId"]

	s.mu.RLock()
	guild, guildExists := s.guilds[guildID]
	user, userExists := s.users[userID]
	s.mu.RUnlock()

	if !guildExists {
		http.Error(w, `{"message": "Unknown Guild", "code": 10004}`, http.StatusNotFound)
		return
	}

	if !userExists {
		http.Error(w, `{"message": "Unknown User", "code": 10013}`, http.StatusNotFound)
		return
	}

	// Check if user is member of guild
	if _, isMember := guild.Members[userID]; !isMember {
		http.Error(w, `{"message": "Unknown Member", "code": 10007}`, http.StatusNotFound)
		return
	}

	member := map[string]interface{}{
		"user":      user,
		"nick":      nil,
		"roles":     []string{},
		"joined_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(member)
}

// getChannel handles GET /channels/{channelId}
func (s *MockDiscordServer) getChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

	s.mu.RLock()
	channel, exists := s.channels[channelID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, `{"message": "Unknown Channel", "code": 10003}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channel)
}

// getChannelMessages handles GET /channels/{channelId}/messages
func (s *MockDiscordServer) getChannelMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

	s.mu.RLock()
	channel, exists := s.channels[channelID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, `{"message": "Unknown Channel", "code": 10003}`, http.StatusNotFound)
		return
	}

	// Parse query parameters
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Return recent messages (up to limit)
	messages := channel.Messages
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// createMessage handles POST /channels/{channelId}/messages
func (s *MockDiscordServer) createMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

	s.mu.Lock()
	channel, exists := s.channels[channelID]
	s.mu.Unlock()

	if !exists {
		http.Error(w, `{"message": "Unknown Channel", "code": 10003}`, http.StatusNotFound)
		return
	}

	var messageData struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&messageData); err != nil {
		http.Error(w, `{"message": "Invalid JSON", "code": 50109}`, http.StatusBadRequest)
		return
	}

	// Create new message
	message := Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Content:   messageData.Content,
		Author:    s.users["bot-user-789"], // Bot user
		ChannelID: channelID,
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	channel.Messages = append(channel.Messages, message)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// updateVoiceState handles PATCH /channels/{channelId}/voice-states/@me
func (s *MockDiscordServer) updateVoiceState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channelId"]

	s.mu.RLock()
	channel, exists := s.channels[channelID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, `{"message": "Unknown Channel", "code": 10003}`, http.StatusNotFound)
		return
	}

	if channel.Type != ChannelTypeVoice {
		http.Error(w, `{"message": "Cannot join non-voice channel", "code": 40032}`, http.StatusBadRequest)
		return
	}

	// Simulate successful voice state update
	w.WriteHeader(http.StatusNoContent)
}

// getCurrentUser handles GET /users/@me
func (s *MockDiscordServer) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Return bot user as current user
	s.mu.RLock()
	botUser := s.users["bot-user-789"]
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(botUser)
}

// getUser handles GET /users/{userId}
func (s *MockDiscordServer) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	s.mu.RLock()
	user, exists := s.users[userID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, `{"message": "Unknown User", "code": 10013}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// interactionCallback handles POST /interactions/{interactionId}/{interactionToken}/callback
func (s *MockDiscordServer) interactionCallback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	interactionID := vars["interactionId"]

	var callback struct {
		Type int                    `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		http.Error(w, `{"message": "Invalid JSON", "code": 50109}`, http.StatusBadRequest)
		return
	}

	// Store interaction for testing purposes
	interaction := Interaction{
		ID:   interactionID,
		Type: callback.Type,
		Data: callback.Data,
	}

	s.mu.Lock()
	s.interactions = append(s.interactions, interaction)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// healthCheck handles GET /health
func (s *MockDiscordServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"guilds":    len(s.guilds),
		"users":     len(s.users),
		"channels":  len(s.channels),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetInteractions returns all recorded interactions for testing
func (s *MockDiscordServer) GetInteractions() []Interaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	interactions := make([]Interaction, len(s.interactions))
	copy(interactions, s.interactions)
	return interactions
}

// Reset clears all recorded interactions and resets test data
func (s *MockDiscordServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interactions = make([]Interaction, 0)
	s.setupTestData()
}
