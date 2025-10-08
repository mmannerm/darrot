package tts

import (
	"fmt"
	"time"
)

// UserServiceImpl implements the UserService interface for managing user opt-in preferences
type UserServiceImpl struct {
	storage *StorageService
}

// NewUserService creates a new UserService instance
func NewUserService(storage *StorageService) *UserServiceImpl {
	return &UserServiceImpl{
		storage: storage,
	}
}

// SetOptInStatus sets the opt-in status for a user in a specific guild
func (u *UserServiceImpl) SetOptInStatus(userID, guildID string, optedIn bool) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	// Load existing preferences or create default ones
	prefs, err := u.storage.LoadUserPreferences(userID, guildID)
	if err != nil {
		// If loading fails, create default preferences
		defaultPrefs := DefaultUserPreferences(userID, guildID)
		prefs = &defaultPrefs
	}

	// Update opt-in status
	prefs.OptedIn = optedIn
	prefs.UpdatedAt = time.Now()

	// Save updated preferences
	if err := u.storage.SaveUserPreferences(*prefs); err != nil {
		return fmt.Errorf("failed to save user preferences: %w", err)
	}

	return nil
}

// IsOptedIn checks if a user has opted in for TTS in a specific guild
func (u *UserServiceImpl) IsOptedIn(userID, guildID string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("user ID cannot be empty")
	}
	if guildID == "" {
		return false, fmt.Errorf("guild ID cannot be empty")
	}

	prefs, err := u.storage.LoadUserPreferences(userID, guildID)
	if err != nil {
		// If preferences don't exist, user is not opted in by default
		return false, nil
	}

	return prefs.OptedIn, nil
}

// GetOptedInUsers returns a list of all users who have opted in for TTS in a specific guild
func (u *UserServiceImpl) GetOptedInUsers(guildID string) ([]string, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}

	optedInUsers, err := u.storage.ListOptedInUsers(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list opted-in users: %w", err)
	}

	return optedInUsers, nil
}

// AutoOptIn automatically opts in a user who invites the bot to a voice channel
// This implements requirement 6.1: users who invite the bot are automatically opted-in
func (u *UserServiceImpl) AutoOptIn(userID, guildID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	// Check if user is already opted in
	isOptedIn, err := u.IsOptedIn(userID, guildID)
	if err != nil {
		return fmt.Errorf("failed to check opt-in status: %w", err)
	}

	// If already opted in, no need to do anything
	if isOptedIn {
		return nil
	}

	// Auto opt-in the user
	if err := u.SetOptInStatus(userID, guildID, true); err != nil {
		return fmt.Errorf("failed to auto opt-in user: %w", err)
	}

	return nil
}

// GetUserPreferences returns the full TTS preferences for a user in a specific guild
func (u *UserServiceImpl) GetUserPreferences(userID, guildID string) (*UserTTSPreferences, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}

	prefs, err := u.storage.LoadUserPreferences(userID, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user preferences: %w", err)
	}

	return prefs, nil
}

// UpdateUserSettings updates the TTS settings for a user in a specific guild
func (u *UserServiceImpl) UpdateUserSettings(userID, guildID string, settings UserTTSSettings) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	// Load existing preferences or create default ones
	prefs, err := u.storage.LoadUserPreferences(userID, guildID)
	if err != nil {
		// If loading fails, create default preferences
		defaultPrefs := DefaultUserPreferences(userID, guildID)
		prefs = &defaultPrefs
	}

	// Update settings
	prefs.Settings = settings
	prefs.UpdatedAt = time.Now()

	// Validate the updated preferences
	if err := ValidateUserPreferences(*prefs); err != nil {
		return fmt.Errorf("invalid user settings: %w", err)
	}

	// Save updated preferences
	if err := u.storage.SaveUserPreferences(*prefs); err != nil {
		return fmt.Errorf("failed to save user preferences: %w", err)
	}

	return nil
}
