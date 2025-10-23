package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GatewayServer handles Discord Gateway WebSocket connections
type GatewayServer struct {
	clients    map[*websocket.Conn]*GatewayClient
	broadcast  chan GatewayEvent
	register   chan *GatewayClient
	unregister chan *GatewayClient
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
}

// GatewayClient represents a connected Discord bot client
type GatewayClient struct {
	conn          *websocket.Conn
	send          chan GatewayEvent
	sessionID     string
	heartbeat     *time.Ticker
	lastHeartbeat time.Time
	authenticated bool
}

// GatewayEvent represents a Discord Gateway event
type GatewayEvent struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
	S  *int        `json:"s,omitempty"`
	T  *string     `json:"t,omitempty"`
}

// Gateway opcodes
const (
	OpDispatch            = 0
	OpHeartbeat           = 1
	OpIdentify            = 2
	OpPresenceUpdate      = 3
	OpVoiceStateUpdate    = 4
	OpResume              = 6
	OpReconnect           = 7
	OpRequestGuildMembers = 8
	OpInvalidSession      = 9
	OpHello               = 10
	OpHeartbeatAck        = 11
)

// NewGatewayServer creates a new Gateway server
func NewGatewayServer() *GatewayServer {
	return &GatewayServer{
		clients:    make(map[*websocket.Conn]*GatewayClient),
		broadcast:  make(chan GatewayEvent, 256),
		register:   make(chan *GatewayClient),
		unregister: make(chan *GatewayClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for testing
			},
		},
	}
}

// Start begins the Gateway server event loop
func (gs *GatewayServer) Start(ctx context.Context) {
	go gs.run(ctx)
}

// run handles the main Gateway server event loop
func (gs *GatewayServer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-gs.register:
			gs.mu.Lock()
			gs.clients[client.conn] = client
			gs.mu.Unlock()

			// Send Hello event
			hello := GatewayEvent{
				Op: OpHello,
				D: map[string]interface{}{
					"heartbeat_interval": 41250, // 41.25 seconds
				},
			}

			select {
			case client.send <- hello:
			default:
				gs.closeClient(client)
			}

		case client := <-gs.unregister:
			gs.closeClient(client)

		case event := <-gs.broadcast:
			gs.mu.RLock()
			for _, client := range gs.clients {
				if client.authenticated {
					select {
					case client.send <- event:
					default:
						gs.closeClient(client)
					}
				}
			}
			gs.mu.RUnlock()
		}
	}
}

// closeClient safely closes a client connection
func (gs *GatewayServer) closeClient(client *GatewayClient) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, ok := gs.clients[client.conn]; ok {
		delete(gs.clients, client.conn)
		close(client.send)
		client.conn.Close()
		if client.heartbeat != nil {
			client.heartbeat.Stop()
		}
	}
}

// HandleWebSocket handles WebSocket upgrade and client management
func (gs *GatewayServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := gs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &GatewayClient{
		conn:          conn,
		send:          make(chan GatewayEvent, 256),
		sessionID:     generateSessionID(),
		lastHeartbeat: time.Now(),
		authenticated: false,
	}

	gs.register <- client

	// Start client goroutines
	go gs.writePump(client)
	go gs.readPump(client)
}

// readPump handles incoming messages from the client
func (gs *GatewayServer) readPump(client *GatewayClient) {
	defer func() {
		gs.unregister <- client
	}()

	client.conn.SetReadLimit(4096)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var event GatewayEvent
		err := client.conn.ReadJSON(&event)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		gs.handleClientEvent(client, event)
	}
}

// writePump handles outgoing messages to the client
func (gs *GatewayServer) writePump(client *GatewayClient) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case event, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteJSON(event); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientEvent processes events received from the client
func (gs *GatewayServer) handleClientEvent(client *GatewayClient, event GatewayEvent) {
	switch event.Op {
	case OpIdentify:
		gs.handleIdentify(client, event)
	case OpHeartbeat:
		gs.handleHeartbeat(client, event)
	case OpVoiceStateUpdate:
		gs.handleVoiceStateUpdate(client, event)
	case OpPresenceUpdate:
		gs.handlePresenceUpdate(client, event)
	default:
		log.Printf("Unknown opcode received: %d", event.Op)
	}
}

// handleIdentify processes client identification
func (gs *GatewayServer) handleIdentify(client *GatewayClient, event GatewayEvent) {
	identifyData, ok := event.D.(map[string]interface{})
	if !ok {
		gs.sendInvalidSession(client)
		return
	}

	token, ok := identifyData["token"].(string)
	if !ok || token == "" {
		gs.sendInvalidSession(client)
		return
	}

	// Simulate successful authentication
	client.authenticated = true

	// Start heartbeat timer
	client.heartbeat = time.NewTicker(41250 * time.Millisecond)
	go gs.heartbeatLoop(client)

	// Send Ready event
	ready := GatewayEvent{
		Op: OpDispatch,
		T:  stringPtr("READY"),
		S:  intPtr(1),
		D: map[string]interface{}{
			"v": 10,
			"user": map[string]interface{}{
				"id":            "bot-user-789",
				"username":      "darrot",
				"discriminator": "0000",
				"bot":           true,
			},
			"guilds": []map[string]interface{}{
				{
					"id":          "test-guild-123",
					"unavailable": false,
				},
			},
			"session_id": client.sessionID,
		},
	}

	select {
	case client.send <- ready:
	default:
		gs.closeClient(client)
	}

	// Send Guild Create event
	guildCreate := GatewayEvent{
		Op: OpDispatch,
		T:  stringPtr("GUILD_CREATE"),
		S:  intPtr(2),
		D: map[string]interface{}{
			"id":   "test-guild-123",
			"name": "Test Guild",
			"channels": []map[string]interface{}{
				{
					"id":       "test-text-channel-456",
					"name":     "general",
					"type":     0,
					"guild_id": "test-guild-123",
				},
				{
					"id":       "test-voice-channel-789",
					"name":     "General Voice",
					"type":     2,
					"guild_id": "test-guild-123",
				},
			},
			"members": []map[string]interface{}{
				{
					"user": map[string]interface{}{
						"id":            "test-user-456",
						"username":      "testuser",
						"discriminator": "1234",
						"bot":           false,
					},
				},
			},
		},
	}

	select {
	case client.send <- guildCreate:
	default:
		gs.closeClient(client)
	}
}

// handleHeartbeat processes heartbeat from client
func (gs *GatewayServer) handleHeartbeat(client *GatewayClient, event GatewayEvent) {
	client.lastHeartbeat = time.Now()

	// Send heartbeat ACK
	ack := GatewayEvent{
		Op: OpHeartbeatAck,
	}

	select {
	case client.send <- ack:
	default:
		gs.closeClient(client)
	}
}

// handleVoiceStateUpdate processes voice state updates
func (gs *GatewayServer) handleVoiceStateUpdate(client *GatewayClient, event GatewayEvent) {
	voiceData, ok := event.D.(map[string]interface{})
	if !ok {
		return
	}

	// Simulate voice state update event
	voiceStateUpdate := GatewayEvent{
		Op: OpDispatch,
		T:  stringPtr("VOICE_STATE_UPDATE"),
		S:  intPtr(3),
		D: map[string]interface{}{
			"guild_id":   voiceData["guild_id"],
			"channel_id": voiceData["channel_id"],
			"user_id":    "bot-user-789",
			"session_id": client.sessionID,
			"deaf":       false,
			"mute":       false,
			"self_deaf":  false,
			"self_mute":  false,
		},
	}

	select {
	case client.send <- voiceStateUpdate:
	default:
		gs.closeClient(client)
	}

	// If joining a voice channel, send voice server update
	if channelID, ok := voiceData["channel_id"].(string); ok && channelID != "" {
		voiceServerUpdate := GatewayEvent{
			Op: OpDispatch,
			T:  stringPtr("VOICE_SERVER_UPDATE"),
			S:  intPtr(4),
			D: map[string]interface{}{
				"token":    "mock-voice-token-123",
				"guild_id": voiceData["guild_id"],
				"endpoint": "mock-voice.discord.gg:80",
			},
		}

		select {
		case client.send <- voiceServerUpdate:
		default:
			gs.closeClient(client)
		}
	}
}

// handlePresenceUpdate processes presence updates
func (gs *GatewayServer) handlePresenceUpdate(client *GatewayClient, event GatewayEvent) {
	// Acknowledge presence update (no response needed)
	log.Printf("Presence update received from client %s", client.sessionID)
}

// sendInvalidSession sends an invalid session event
func (gs *GatewayServer) sendInvalidSession(client *GatewayClient) {
	invalidSession := GatewayEvent{
		Op: OpInvalidSession,
		D:  false, // Not resumable
	}

	select {
	case client.send <- invalidSession:
	default:
		gs.closeClient(client)
	}
}

// heartbeatLoop monitors client heartbeat
func (gs *GatewayServer) heartbeatLoop(client *GatewayClient) {
	for range client.heartbeat.C {
		// Check if client is still alive (heartbeat within last 60 seconds)
		if time.Since(client.lastHeartbeat) > 60*time.Second {
			log.Printf("Client %s heartbeat timeout", client.sessionID)
			gs.unregister <- client
			return
		}
	}
}

// BroadcastEvent sends an event to all authenticated clients
func (gs *GatewayServer) BroadcastEvent(event GatewayEvent) {
	select {
	case gs.broadcast <- event:
	default:
		log.Printf("Broadcast channel full, dropping event")
	}
}

// SimulateMessageCreate simulates a MESSAGE_CREATE event
func (gs *GatewayServer) SimulateMessageCreate(channelID, content, userID string) {
	event := GatewayEvent{
		Op: OpDispatch,
		T:  stringPtr("MESSAGE_CREATE"),
		S:  intPtr(5),
		D: map[string]interface{}{
			"id":         generateMessageID(),
			"channel_id": channelID,
			"content":    content,
			"author": map[string]interface{}{
				"id":            userID,
				"username":      "testuser",
				"discriminator": "1234",
				"bot":           false,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	gs.BroadcastEvent(event)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func generateSessionID() string {
	return "mock-session-" + time.Now().Format("20060102150405")
}

func generateMessageID() string {
	return "msg-" + time.Now().Format("20060102150405") + "-" + time.Now().Format("000")
}
