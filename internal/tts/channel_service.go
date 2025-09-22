package tts

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// InMemoryChannelService provides an in-memory implementation of ChannelService
type InMemoryChannelService struct {
	pairings       map[string]*ChannelPairing // voiceChannelID -> pairing
	textChannelMap map[string]string          // textChannelID -> voiceChannelID
	mutex          sync.RWMutex
	logger         *log.Logger
}

// NewInMemoryChannelService creates a new in-memory channel service
func NewInMemoryChannelService(logger *log.Logger) *InMemoryChannelService {
	return &InMemoryChannelService{
		pairings:       make(map[string]*ChannelPairing),
		textChannelMap: make(map[string]string),
		logger:         logger,
	}
}

// CreatePairing creates a new voice-text channel pairing
func (s *InMemoryChannelService) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	if guildID == "" || voiceChannelID == "" || textChannelID == "" {
		return fmt.Errorf("guild ID, voice channel ID, and text channel ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if text channel is already paired
	if existingVoiceID, exists := s.textChannelMap[textChannelID]; exists {
		return fmt.Errorf("text channel %s is already paired with voice channel %s", textChannelID, existingVoiceID)
	}

	// Remove existing pairing for this voice channel if it exists
	if existingPairing, exists := s.pairings[voiceChannelID]; exists {
		delete(s.textChannelMap, existingPairing.TextChannelID)
	}

	// Create new pairing
	pairing := &ChannelPairing{
		GuildID:        guildID,
		VoiceChannelID: voiceChannelID,
		TextChannelID:  textChannelID,
		CreatedBy:      "", // Will be set by caller
		CreatedAt:      time.Now(),
	}

	s.pairings[voiceChannelID] = pairing
	s.textChannelMap[textChannelID] = voiceChannelID

	s.logger.Printf("Created channel pairing: voice %s <-> text %s in guild %s", voiceChannelID, textChannelID, guildID)
	return nil
}

// RemovePairing removes a voice-text channel pairing
func (s *InMemoryChannelService) RemovePairing(guildID, voiceChannelID string) error {
	if guildID == "" || voiceChannelID == "" {
		return fmt.Errorf("guild ID and voice channel ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	pairing, exists := s.pairings[voiceChannelID]
	if !exists {
		return fmt.Errorf("no pairing found for voice channel %s", voiceChannelID)
	}

	if pairing.GuildID != guildID {
		return fmt.Errorf("voice channel %s does not belong to guild %s", voiceChannelID, guildID)
	}

	// Remove from both maps
	delete(s.pairings, voiceChannelID)
	delete(s.textChannelMap, pairing.TextChannelID)

	s.logger.Printf("Removed channel pairing: voice %s <-> text %s in guild %s", voiceChannelID, pairing.TextChannelID, guildID)
	return nil
}

// GetPairing retrieves a channel pairing by voice channel ID
func (s *InMemoryChannelService) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	if guildID == "" || voiceChannelID == "" {
		return nil, fmt.Errorf("guild ID and voice channel ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	pairing, exists := s.pairings[voiceChannelID]
	if !exists {
		return nil, fmt.Errorf("no pairing found for voice channel %s", voiceChannelID)
	}

	if pairing.GuildID != guildID {
		return nil, fmt.Errorf("voice channel %s does not belong to guild %s", voiceChannelID, guildID)
	}

	// Return a copy to prevent external modification
	return &ChannelPairing{
		GuildID:        pairing.GuildID,
		VoiceChannelID: pairing.VoiceChannelID,
		TextChannelID:  pairing.TextChannelID,
		CreatedBy:      pairing.CreatedBy,
		CreatedAt:      pairing.CreatedAt,
	}, nil
}

// ValidateChannelAccess validates that a user has access to a channel
// This is a simplified implementation - in production, this would check Discord permissions
func (s *InMemoryChannelService) ValidateChannelAccess(userID, channelID string) error {
	if userID == "" || channelID == "" {
		return fmt.Errorf("user ID and channel ID cannot be empty")
	}

	// For now, assume all users have access to all channels
	// In a real implementation, this would check Discord permissions via the API
	return nil
}

// IsChannelPaired checks if a text channel is paired with any voice channel
func (s *InMemoryChannelService) IsChannelPaired(guildID, textChannelID string) (bool, error) {
	if guildID == "" || textChannelID == "" {
		return false, fmt.Errorf("guild ID and text channel ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	voiceChannelID, exists := s.textChannelMap[textChannelID]
	if !exists {
		return false, nil
	}

	// Verify the pairing still exists and belongs to the correct guild
	pairing, exists := s.pairings[voiceChannelID]
	if !exists || pairing.GuildID != guildID {
		// Clean up orphaned mapping
		delete(s.textChannelMap, textChannelID)
		return false, nil
	}

	return true, nil
}

// GetPairingByTextChannel retrieves a channel pairing by text channel ID
func (s *InMemoryChannelService) GetPairingByTextChannel(guildID, textChannelID string) (*ChannelPairing, error) {
	if guildID == "" || textChannelID == "" {
		return nil, fmt.Errorf("guild ID and text channel ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	voiceChannelID, exists := s.textChannelMap[textChannelID]
	if !exists {
		return nil, fmt.Errorf("no pairing found for text channel %s", textChannelID)
	}

	pairing, exists := s.pairings[voiceChannelID]
	if !exists || pairing.GuildID != guildID {
		return nil, fmt.Errorf("pairing not found or does not belong to guild %s", guildID)
	}

	// Return a copy to prevent external modification
	return &ChannelPairing{
		GuildID:        pairing.GuildID,
		VoiceChannelID: pairing.VoiceChannelID,
		TextChannelID:  pairing.TextChannelID,
		CreatedBy:      pairing.CreatedBy,
		CreatedAt:      pairing.CreatedAt,
	}, nil
}
