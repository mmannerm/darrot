package tts

import (
	"fmt"
	"log"
	"sync"
)

// UserOptInStatus represents a user's opt-in status for a specific guild
type UserOptInStatus struct {
	UserID  string
	GuildID string
	OptedIn bool
}

// InMemoryUserService provides an in-memory implementation of UserService
type InMemoryUserService struct {
	optInStatus map[string]bool // "userID:guildID" -> opted in status
	mutex       sync.RWMutex
	logger      *log.Logger
}

// NewInMemoryUserService creates a new in-memory user service
func NewInMemoryUserService(logger *log.Logger) *InMemoryUserService {
	return &InMemoryUserService{
		optInStatus: make(map[string]bool),
		logger:      logger,
	}
}

// SetOptInStatus sets the opt-in status for a user in a specific guild
func (s *InMemoryUserService) SetOptInStatus(userID, guildID string, optedIn bool) error {
	if userID == "" || guildID == "" {
		return fmt.Errorf("user ID and guild ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	key := s.makeKey(userID, guildID)
	s.optInStatus[key] = optedIn

	status := "opted out"
	if optedIn {
		status = "opted in"
	}

	s.logger.Printf("User %s %s for TTS in guild %s", userID, status, guildID)
	return nil
}

// IsOptedIn checks if a user is opted-in for TTS in a specific guild
func (s *InMemoryUserService) IsOptedIn(userID, guildID string) (bool, error) {
	if userID == "" || guildID == "" {
		return false, fmt.Errorf("user ID and guild ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := s.makeKey(userID, guildID)
	optedIn, exists := s.optInStatus[key]

	// Default to false if not explicitly set
	if !exists {
		return false, nil
	}

	return optedIn, nil
}

// GetOptedInUsers returns a list of user IDs that are opted-in for TTS in a specific guild
func (s *InMemoryUserService) GetOptedInUsers(guildID string) ([]string, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var optedInUsers []string
	guildPrefix := ":" + guildID

	for key, optedIn := range s.optInStatus {
		if optedIn && len(key) > len(guildPrefix) && key[len(key)-len(guildPrefix):] == guildPrefix {
			// Extract user ID from key (format is "userID:guildID")
			userID := key[:len(key)-len(guildPrefix)]
			optedInUsers = append(optedInUsers, userID)
		}
	}

	return optedInUsers, nil
}

// AutoOptIn automatically opts in a user (typically used when they invite the bot)
func (s *InMemoryUserService) AutoOptIn(userID, guildID string) error {
	if userID == "" || guildID == "" {
		return fmt.Errorf("user ID and guild ID cannot be empty")
	}

	s.logger.Printf("Auto-opting in user %s for TTS in guild %s (bot inviter)", userID, guildID)
	return s.SetOptInStatus(userID, guildID, true)
}

// makeKey creates a composite key for storing user opt-in status
func (s *InMemoryUserService) makeKey(userID, guildID string) string {
	return userID + ":" + guildID
}

// GetOptInCount returns the number of opted-in users for a specific guild
func (s *InMemoryUserService) GetOptInCount(guildID string) (int, error) {
	optedInUsers, err := s.GetOptedInUsers(guildID)
	if err != nil {
		return 0, err
	}
	return len(optedInUsers), nil
}

// ClearGuildData removes all opt-in data for a specific guild
func (s *InMemoryUserService) ClearGuildData(guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	guildSuffix := ":" + guildID
	keysToDelete := make([]string, 0)

	// Find all keys for this guild
	for key := range s.optInStatus {
		if len(key) > len(guildSuffix) && key[len(key)-len(guildSuffix):] == guildSuffix {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Delete the keys
	for _, key := range keysToDelete {
		delete(s.optInStatus, key)
	}

	s.logger.Printf("Cleared opt-in data for %d users in guild %s", len(keysToDelete), guildID)
	return nil
}
