package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

// VoiceServer handles Discord voice connections and audio streaming
type VoiceServer struct {
	connections  map[string]*VoiceConnection
	audioCapture *AudioCapture
	mu           sync.RWMutex
	listener     net.Listener
}

// VoiceConnection represents a voice connection from a bot
type VoiceConnection struct {
	SessionID   string
	GuildID     string
	ChannelID   string
	UserID      string
	Conn        net.Conn
	Connected   bool
	AudioStream chan []byte
	LastPacket  time.Time
}

// AudioCapture handles capturing and analyzing audio streams
type AudioCapture struct {
	captures map[string]*AudioCaptureSession
	mu       sync.RWMutex
}

// AudioCaptureSession represents an active audio capture session
type AudioCaptureSession struct {
	ChannelID     string
	StartTime     time.Time
	EndTime       *time.Time
	AudioPackets  []AudioPacket
	TotalDuration time.Duration
	PacketCount   int
}

// AudioPacket represents a captured audio packet
type AudioPacket struct {
	Timestamp time.Time
	Data      []byte
	Sequence  uint16
	SSRC      uint32
	Format    string // "opus", "pcm", etc.
}

// VoicePacket represents a Discord voice packet structure
type VoicePacket struct {
	Version     byte
	PayloadType byte
	Sequence    uint16
	Timestamp   uint32
	SSRC        uint32
	Payload     []byte
}

// NewVoiceServer creates a new voice server
func NewVoiceServer() *VoiceServer {
	return &VoiceServer{
		connections: make(map[string]*VoiceConnection),
		audioCapture: &AudioCapture{
			captures: make(map[string]*AudioCaptureSession),
		},
	}
}

// Start begins the voice server
func (vs *VoiceServer) Start(ctx context.Context, port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	vs.listener = listener
	log.Printf("Voice server listening on port %s", port)

	go vs.acceptConnections(ctx)
	return nil
}

// Stop shuts down the voice server
func (vs *VoiceServer) Stop() error {
	if vs.listener != nil {
		return vs.listener.Close()
	}
	return nil
}

// acceptConnections handles incoming voice connections
func (vs *VoiceServer) acceptConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := vs.listener.Accept()
			if err != nil {
				log.Printf("Failed to accept voice connection: %v", err)
				continue
			}

			go vs.handleVoiceConnection(ctx, conn)
		}
	}
}

// handleVoiceConnection processes a voice connection
func (vs *VoiceServer) handleVoiceConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// Simulate voice connection handshake
	voiceConn := &VoiceConnection{
		SessionID:   generateSessionID(),
		Conn:        conn,
		Connected:   true,
		AudioStream: make(chan []byte, 1000),
		LastPacket:  time.Now(),
	}

	// Read initial connection data
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Failed to read voice connection data: %v", err)
		return
	}

	// Parse connection info (simplified)
	var connectionInfo struct {
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
		UserID    string `json:"user_id"`
		Token     string `json:"token"`
	}

	if err := json.Unmarshal(buffer[:n], &connectionInfo); err != nil {
		log.Printf("Failed to parse voice connection info: %v", err)
		return
	}

	voiceConn.GuildID = connectionInfo.GuildID
	voiceConn.ChannelID = connectionInfo.ChannelID
	voiceConn.UserID = connectionInfo.UserID

	vs.mu.Lock()
	vs.connections[voiceConn.SessionID] = voiceConn
	vs.mu.Unlock()

	log.Printf("Voice connection established: %s (Channel: %s)", voiceConn.SessionID, voiceConn.ChannelID)

	// Start audio capture for this channel
	vs.audioCapture.StartCapture(voiceConn.ChannelID)

	// Handle audio packets
	vs.handleAudioStream(ctx, voiceConn)

	// Cleanup
	vs.mu.Lock()
	delete(vs.connections, voiceConn.SessionID)
	vs.mu.Unlock()

	vs.audioCapture.StopCapture(voiceConn.ChannelID)
	log.Printf("Voice connection closed: %s", voiceConn.SessionID)
}

// handleAudioStream processes incoming audio packets
func (vs *VoiceServer) handleAudioStream(ctx context.Context, conn *VoiceConnection) {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set read timeout
			conn.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			n, err := conn.Conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					log.Printf("Voice connection timeout: %s", conn.SessionID)
				} else {
					log.Printf("Voice connection read error: %v", err)
				}
				return
			}

			if n > 0 {
				conn.LastPacket = time.Now()

				// Parse voice packet
				packet := vs.parseVoicePacket(buffer[:n])
				if packet != nil {
					// Capture audio data
					vs.audioCapture.CapturePacket(conn.ChannelID, AudioPacket{
						Timestamp: time.Now(),
						Data:      packet.Payload,
						Sequence:  packet.Sequence,
						SSRC:      packet.SSRC,
						Format:    "opus", // Assume Opus format
					})
				}
			}
		}
	}
}

// parseVoicePacket parses a Discord voice packet
func (vs *VoiceServer) parseVoicePacket(data []byte) *VoicePacket {
	if len(data) < 12 {
		return nil // Invalid packet size
	}

	packet := &VoicePacket{
		Version:     (data[0] >> 6) & 0x3,
		PayloadType: data[1] & 0x7F,
		Sequence:    uint16(data[2])<<8 | uint16(data[3]),
		Timestamp:   uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]),
		SSRC:        uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11]),
		Payload:     data[12:],
	}

	return packet
}

// StartCapture begins audio capture for a channel
func (ac *AudioCapture) StartCapture(channelID string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if _, exists := ac.captures[channelID]; exists {
		return // Already capturing
	}

	session := &AudioCaptureSession{
		ChannelID:    channelID,
		StartTime:    time.Now(),
		AudioPackets: make([]AudioPacket, 0),
		PacketCount:  0,
	}

	ac.captures[channelID] = session
	log.Printf("Started audio capture for channel: %s", channelID)
}

// StopCapture ends audio capture for a channel
func (ac *AudioCapture) StopCapture(channelID string) *AudioCaptureSession {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	session, exists := ac.captures[channelID]
	if !exists {
		return nil
	}

	now := time.Now()
	session.EndTime = &now
	session.TotalDuration = now.Sub(session.StartTime)

	// Remove from active captures
	delete(ac.captures, channelID)

	log.Printf("Stopped audio capture for channel: %s (Duration: %v, Packets: %d)",
		channelID, session.TotalDuration, session.PacketCount)

	return session
}

// CapturePacket adds an audio packet to the capture session
func (ac *AudioCapture) CapturePacket(channelID string, packet AudioPacket) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	session, exists := ac.captures[channelID]
	if !exists {
		return
	}

	session.AudioPackets = append(session.AudioPackets, packet)
	session.PacketCount++
}

// GetCaptureSession returns the capture session for a channel
func (ac *AudioCapture) GetCaptureSession(channelID string) *AudioCaptureSession {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	session, exists := ac.captures[channelID]
	if !exists {
		return nil
	}

	// Return a copy
	sessionCopy := *session
	sessionCopy.AudioPackets = make([]AudioPacket, len(session.AudioPackets))
	copy(sessionCopy.AudioPackets, session.AudioPackets)

	return &sessionCopy
}

// ValidateAudioFormat checks if captured audio is in expected format
func (ac *AudioCapture) ValidateAudioFormat(channelID string, expectedFormat string) bool {
	session := ac.GetCaptureSession(channelID)
	if session == nil || len(session.AudioPackets) == 0 {
		return false
	}

	// Check if all packets match expected format
	for _, packet := range session.AudioPackets {
		if packet.Format != expectedFormat {
			return false
		}
	}

	return true
}

// AnalyzeAudioQuality performs basic audio quality analysis
func (ac *AudioCapture) AnalyzeAudioQuality(channelID string) *AudioQualityReport {
	session := ac.GetCaptureSession(channelID)
	if session == nil {
		return nil
	}

	report := &AudioQualityReport{
		ChannelID:     channelID,
		TotalPackets:  session.PacketCount,
		TotalDuration: session.TotalDuration,
		StartTime:     session.StartTime,
	}

	if session.EndTime != nil {
		report.EndTime = *session.EndTime
	}

	if len(session.AudioPackets) > 0 {
		// Calculate average packet size
		totalSize := 0
		for _, packet := range session.AudioPackets {
			totalSize += len(packet.Data)
		}
		report.AvgPacketSize = totalSize / len(session.AudioPackets)

		// Calculate packet rate
		if session.TotalDuration > 0 {
			report.PacketRate = float64(session.PacketCount) / session.TotalDuration.Seconds()
		}

		// Check for packet gaps (simplified)
		report.PacketGaps = ac.detectPacketGaps(session.AudioPackets)
	}

	return report
}

// AudioQualityReport contains audio quality analysis results
type AudioQualityReport struct {
	ChannelID     string        `json:"channel_id"`
	TotalPackets  int           `json:"total_packets"`
	TotalDuration time.Duration `json:"total_duration"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	AvgPacketSize int           `json:"avg_packet_size"`
	PacketRate    float64       `json:"packet_rate"`
	PacketGaps    int           `json:"packet_gaps"`
	Quality       string        `json:"quality"`
}

// detectPacketGaps detects missing packets in the audio stream
func (ac *AudioCapture) detectPacketGaps(packets []AudioPacket) int {
	if len(packets) < 2 {
		return 0
	}

	gaps := 0
	for i := 1; i < len(packets); i++ {
		expectedSeq := packets[i-1].Sequence + 1
		if packets[i].Sequence != expectedSeq {
			gaps++
		}
	}

	return gaps
}

// GetActiveConnections returns all active voice connections
func (vs *VoiceServer) GetActiveConnections() map[string]*VoiceConnection {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	connections := make(map[string]*VoiceConnection)
	for k, v := range vs.connections {
		connections[k] = v
	}

	return connections
}

// SimulateVoiceChannelJoin simulates a bot joining a voice channel
func (vs *VoiceServer) SimulateVoiceChannelJoin(guildID, channelID, userID string) string {
	sessionID := generateSessionID()

	voiceConn := &VoiceConnection{
		SessionID:   sessionID,
		GuildID:     guildID,
		ChannelID:   channelID,
		UserID:      userID,
		Connected:   true,
		AudioStream: make(chan []byte, 1000),
		LastPacket:  time.Now(),
	}

	vs.mu.Lock()
	vs.connections[sessionID] = voiceConn
	vs.mu.Unlock()

	// Start audio capture
	vs.audioCapture.StartCapture(channelID)

	log.Printf("Simulated voice channel join: %s (Channel: %s)", sessionID, channelID)
	return sessionID
}

// SimulateVoiceChannelLeave simulates a bot leaving a voice channel
func (vs *VoiceServer) SimulateVoiceChannelLeave(sessionID string) *AudioCaptureSession {
	vs.mu.Lock()
	conn, exists := vs.connections[sessionID]
	if exists {
		delete(vs.connections, sessionID)
	}
	vs.mu.Unlock()

	if !exists {
		return nil
	}

	// Stop audio capture and return session
	session := vs.audioCapture.StopCapture(conn.ChannelID)

	log.Printf("Simulated voice channel leave: %s", sessionID)
	return session
}
