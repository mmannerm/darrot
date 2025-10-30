package acceptance

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Interaction represents a Discord interaction
type Interaction struct {
	ID      string                 `json:"id"`
	Type    int                    `json:"type"`
	Data    map[string]interface{} `json:"data"`
	User    interface{}            `json:"user"`
	GuildID string                 `json:"guild_id"`
}

// Message represents a Discord message
type Message struct {
	ID        string      `json:"id"`
	Content   string      `json:"content"`
	Author    interface{} `json:"author"`
	ChannelID string      `json:"channel_id"`
	Timestamp time.Time   `json:"timestamp"`
}

// ChannelType represents Discord channel types
type ChannelType int

const (
	ChannelTypeText  ChannelType = 0
	ChannelTypeVoice ChannelType = 2
)

// mockServer implements the MockDiscordAPI interface
type mockServer struct {
	config       *TestConfig
	server       *http.Server
	router       *mux.Router
	guilds       map[string]*mockGuild
	users        map[string]*mockUser
	channels     map[string]*mockChannel
	interactions []Interaction
	running      bool
	mu           sync.RWMutex
}

// mockUser implements the MockUser interface
type mockUser struct {
	id            string
	username      string
	discriminator string
	bot           bool
	server        *mockServer
}

// mockGuild implements the MockGuild interface
type mockGuild struct {
	id       string
	name     string
	channels map[string]*mockChannel
	members  map[string]*mockUser
	server   *mockServer
}

// mockChannel implements the MockChannel interface
type mockChannel struct {
	id          string
	name        string
	channelType ChannelType
	guildID     string
	messages    []Message
	server      *mockServer
}

// NewMockServer creates a new mock Discord server
func NewMockServer(config *TestConfig) (MockDiscordAPI, error) {
	router := mux.NewRouter()

	server := &mockServer{
		config:       config,
		router:       router,
		guilds:       make(map[string]*mockGuild),
		users:        make(map[string]*mockUser),
		channels:     make(map[string]*mockChannel),
		interactions: make([]Interaction, 0),
		running:      false,
	}

	// Set up routes
	server.setupRoutes()

	// Create default test data
	server.setupTestData()

	return server, nil
}

// Start starts the mock Discord server
func (ms *mockServer) Start(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.running {
		return fmt.Errorf("mock server is already running")
	}

	address := fmt.Sprintf("%s:%d", ms.config.MockServer.Host, ms.config.MockServer.Port)

	ms.server = &http.Server{
		Addr:    address,
		Handler: ms.router,
	}

	// Start server in goroutine
	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Mock server error: %v\n", err)
		}
	}()

	ms.running = true

	// Wait for server to be ready
	if !ms.waitForReady(30 * time.Second) {
		return fmt.Errorf("mock server failed to start within timeout")
	}

	return nil
}

// Stop stops the mock Discord server
func (ms *mockServer) Stop() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.running || ms.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ms.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown mock server: %w", err)
	}

	ms.running = false
	ms.server = nil

	return nil
}

// Reset resets the mock server state
func (ms *mockServer) Reset() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.interactions = make([]Interaction, 0)
	ms.setupTestData()

	return nil
}

// IsHealthy checks if the mock server is healthy
func (ms *mockServer) IsHealthy() bool {
	if !ms.running {
		return false
	}

	// Try to make a health check request
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(ms.GetBaseURL() + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetBaseURL returns the base URL of the mock server
func (ms *mockServer) GetBaseURL() string {
	return fmt.Sprintf("http://%s:%d", ms.config.MockServer.Host, ms.config.MockServer.Port)
}

// GetGatewayURL returns the WebSocket gateway URL
func (ms *mockServer) GetGatewayURL() string {
	return fmt.Sprintf("ws://%s:%d/gateway", ms.config.MockServer.Host, ms.config.MockServer.GatewayPort)
}

// SimulateUser creates or returns a simulated user
func (ms *mockServer) SimulateUser(userID string) MockUser {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if user, exists := ms.users[userID]; exists {
		return user
	}

	user := &mockUser{
		id:            userID,
		username:      fmt.Sprintf("user_%s", userID),
		discriminator: "1234",
		bot:           false,
		server:        ms,
	}

	ms.users[userID] = user
	return user
}

// CreateGuild creates a new mock guild
func (ms *mockServer) CreateGuild(guildID string) MockGuild {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if guild, exists := ms.guilds[guildID]; exists {
		return guild
	}

	guild := &mockGuild{
		id:       guildID,
		name:     fmt.Sprintf("Test Guild %s", guildID),
		channels: make(map[string]*mockChannel),
		members:  make(map[string]*mockUser),
		server:   ms,
	}

	ms.guilds[guildID] = guild
	return guild
}

// GetInteractions returns all recorded interactions
func (ms *mockServer) GetInteractions() []Interaction {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	interactions := make([]Interaction, len(ms.interactions))
	copy(interactions, ms.interactions)
	return interactions
}

// setupRoutes configures HTTP routes
func (ms *mockServer) setupRoutes() {
	// Health check
	ms.router.HandleFunc("/health", ms.healthCheck).Methods("GET")

	// API routes
	api := ms.router.PathPrefix("/api/v10").Subrouter()
	api.Use(ms.authMiddleware)

	// Guild endpoints
	api.HandleFunc("/guilds/{guildId}", ms.getGuild).Methods("GET")
	api.HandleFunc("/guilds/{guildId}/channels", ms.getGuildChannels).Methods("GET")

	// Channel endpoints
	api.HandleFunc("/channels/{channelId}", ms.getChannel).Methods("GET")
	api.HandleFunc("/channels/{channelId}/messages", ms.getChannelMessages).Methods("GET")
	api.HandleFunc("/channels/{channelId}/messages", ms.createMessage).Methods("POST")

	// Voice endpoints
	api.HandleFunc("/channels/{channelId}/voice-states/@me", ms.updateVoiceState).Methods("PATCH")

	// User endpoints
	api.HandleFunc("/users/@me", ms.getCurrentUser).Methods("GET")

	// Gateway endpoint
	ms.router.HandleFunc("/gateway", ms.handleGateway)
}

// setupTestData creates default test data
func (ms *mockServer) setupTestData() {
	// Create test guild
	guild := ms.CreateGuild("test-guild-123").(*mockGuild)

	// Create test channels
	textChannel := guild.CreateTextChannel("general").(*mockChannel)
	voiceChannel := guild.CreateVoiceChannel("General Voice").(*mockChannel)

	// Create test users
	testUser := ms.SimulateUser("test-user-456").(*mockUser)
	botUser := ms.SimulateUser("bot-user-789").(*mockUser)
	botUser.bot = true
	botUser.username = "darrot"

	// Add users to guild
	guild.AddUser(testUser)
	guild.AddUser(botUser)

	// Store channels in server map for direct access
	ms.channels[textChannel.id] = textChannel
	ms.channels[voiceChannel.id] = voiceChannel
}

// waitForReady waits for the server to become ready
func (ms *mockServer) waitForReady(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if ms.IsHealthy() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

// HTTP handlers (simplified versions of the existing mock server)
func (ms *mockServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple auth check - just verify Authorization header exists
		if auth := r.Header.Get("Authorization"); auth == "" {
			http.Error(w, `{"message": "401: Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (ms *mockServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

func (ms *mockServer) getGuild(w http.ResponseWriter, r *http.Request) {
	// Simplified implementation - return success for any guild
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "test-guild-123", "name": "Test Guild"}`)
}

func (ms *mockServer) getGuildChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `[
		{"id": "test-text-channel-456", "name": "general", "type": 0},
		{"id": "test-voice-channel-789", "name": "General Voice", "type": 2}
	]`)
}

func (ms *mockServer) getChannel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "test-text-channel-456", "name": "general", "type": 0}`)
}

func (ms *mockServer) getChannelMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `[]`)
}

func (ms *mockServer) createMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "msg-123", "content": "Test message", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

func (ms *mockServer) updateVoiceState(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (ms *mockServer) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "bot-user-789", "username": "darrot", "bot": true}`)
}

func (ms *mockServer) handleGateway(w http.ResponseWriter, r *http.Request) {
	// WebSocket gateway handling would go here
	// For now, just return a simple response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"url": "%s"}`, ms.GetGatewayURL())
}

// MockUser implementation
func (mu *mockUser) SendMessage(channelID, content string) error {
	// Simulate sending a message
	message := Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Content:   content,
		Author:    mu,
		ChannelID: channelID,
		Timestamp: time.Now(),
	}

	mu.server.mu.Lock()
	if channel, exists := mu.server.channels[channelID]; exists {
		channel.messages = append(channel.messages, message)
	}
	mu.server.mu.Unlock()

	return nil
}

func (mu *mockUser) JoinVoiceChannel(channelID string) error {
	// Simulate joining a voice channel
	return nil
}

func (mu *mockUser) LeaveVoiceChannel() error {
	// Simulate leaving a voice channel
	return nil
}

func (mu *mockUser) SendSlashCommand(command string, options map[string]interface{}) error {
	// Simulate sending a slash command
	interaction := Interaction{
		ID:      fmt.Sprintf("interaction-%d", time.Now().UnixNano()),
		Type:    2, // Application command
		Data:    map[string]interface{}{"name": command, "options": options},
		User:    mu,
		GuildID: "test-guild-123",
	}

	mu.server.mu.Lock()
	mu.server.interactions = append(mu.server.interactions, interaction)
	mu.server.mu.Unlock()

	return nil
}

func (mu *mockUser) GetID() string {
	return mu.id
}

func (mu *mockUser) GetUsername() string {
	return mu.username
}

// MockGuild implementation
func (mg *mockGuild) CreateTextChannel(name string) MockChannel {
	channelID := fmt.Sprintf("text-channel-%d", time.Now().UnixNano())

	channel := &mockChannel{
		id:          channelID,
		name:        name,
		channelType: ChannelTypeText,
		guildID:     mg.id,
		messages:    make([]Message, 0),
		server:      mg.server,
	}

	mg.channels[channelID] = channel
	mg.server.channels[channelID] = channel

	return channel
}

func (mg *mockGuild) CreateVoiceChannel(name string) MockChannel {
	channelID := fmt.Sprintf("voice-channel-%d", time.Now().UnixNano())

	channel := &mockChannel{
		id:          channelID,
		name:        name,
		channelType: ChannelTypeVoice,
		guildID:     mg.id,
		messages:    make([]Message, 0),
		server:      mg.server,
	}

	mg.channels[channelID] = channel
	mg.server.channels[channelID] = channel

	return channel
}

func (mg *mockGuild) AddUser(user MockUser) error {
	if mockUser, ok := user.(*mockUser); ok {
		mg.members[mockUser.id] = mockUser
		return nil
	}
	return fmt.Errorf("invalid user type")
}

func (mg *mockGuild) SetPermissions(userID string, permissions int64) error {
	// Simulate setting permissions
	return nil
}

func (mg *mockGuild) GetID() string {
	return mg.id
}

func (mg *mockGuild) GetChannels() []MockChannel {
	channels := make([]MockChannel, 0, len(mg.channels))
	for _, channel := range mg.channels {
		channels = append(channels, channel)
	}
	return channels
}

// MockChannel implementation
func (mc *mockChannel) GetID() string {
	return mc.id
}

func (mc *mockChannel) GetName() string {
	return mc.name
}

func (mc *mockChannel) GetType() ChannelType {
	return mc.channelType
}

func (mc *mockChannel) SendMessage(content string, userID string) error {
	message := Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Content:   content,
		ChannelID: mc.id,
		Timestamp: time.Now(),
	}

	if user, exists := mc.server.users[userID]; exists {
		message.Author = user
	}

	mc.messages = append(mc.messages, message)
	return nil
}

func (mc *mockChannel) GetMessages() []Message {
	messages := make([]Message, len(mc.messages))
	copy(messages, mc.messages)
	return messages
}
