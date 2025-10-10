package tts

import (
	"os"
	"testing"
)

func TestUserService_SetOptInStatus(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	tests := []struct {
		name    string
		userID  string
		guildID string
		optedIn bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid opt-in",
			userID:  "user123",
			guildID: "guild456",
			optedIn: true,
			wantErr: false,
		},
		{
			name:    "valid opt-out",
			userID:  "user123",
			guildID: "guild456",
			optedIn: false,
			wantErr: false,
		},
		{
			name:    "empty user ID",
			userID:  "",
			guildID: "guild456",
			optedIn: true,
			wantErr: true,
			errMsg:  "user ID cannot be empty",
		},
		{
			name:    "empty guild ID",
			userID:  "user123",
			guildID: "",
			optedIn: true,
			wantErr: true,
			errMsg:  "guild ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := userService.SetOptInStatus(tt.userID, tt.guildID, tt.optedIn)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetOptInStatus() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("SetOptInStatus() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("SetOptInStatus() unexpected error = %v", err)
				return
			}

			// Verify the opt-in status was set correctly
			isOptedIn, err := userService.IsOptedIn(tt.userID, tt.guildID)
			if err != nil {
				t.Errorf("IsOptedIn() error = %v", err)
				return
			}

			if isOptedIn != tt.optedIn {
				t.Errorf("IsOptedIn() = %v, want %v", isOptedIn, tt.optedIn)
			}
		})
	}
}

func TestUserService_IsOptedIn(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	tests := []struct {
		name    string
		userID  string
		guildID string
		setup   func() // Function to set up test data
		want    bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "user not opted in (no preferences file)",
			userID:  "user123",
			guildID: "guild456",
			setup:   func() {}, // No setup needed
			want:    false,
			wantErr: false,
		},
		{
			name:    "user opted in",
			userID:  "user123",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name:    "user opted out",
			userID:  "user123",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", false); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			want:    false,
			wantErr: false,
		},
		{
			name:    "empty user ID",
			userID:  "",
			guildID: "guild456",
			setup:   func() {},
			want:    false,
			wantErr: true,
			errMsg:  "user ID cannot be empty",
		},
		{
			name:    "empty guild ID",
			userID:  "user123",
			guildID: "",
			setup:   func() {},
			want:    false,
			wantErr: true,
			errMsg:  "guild ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing test data
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			// Run setup
			tt.setup()

			got, err := userService.IsOptedIn(tt.userID, tt.guildID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("IsOptedIn() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("IsOptedIn() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("IsOptedIn() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("IsOptedIn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserService_GetOptedInUsers(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	tests := []struct {
		name    string
		guildID string
		setup   func() // Function to set up test data
		want    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no opted-in users",
			guildID: "guild456",
			setup:   func() {}, // No setup needed
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "single opted-in user",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			want:    []string{"user123"},
			wantErr: false,
		},
		{
			name:    "multiple opted-in users",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
				if err := userService.SetOptInStatus("user456", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
				if err := userService.SetOptInStatus("user789", "guild456", false); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				} // This one should not be included
			},
			want:    []string{"user123", "user456"},
			wantErr: false,
		},
		{
			name:    "users from different guilds",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
				if err := userService.SetOptInStatus("user456", "guild789", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				} // Different guild, should not be included
			},
			want:    []string{"user123"},
			wantErr: false,
		},
		{
			name:    "empty guild ID",
			guildID: "",
			setup:   func() {},
			want:    nil,
			wantErr: true,
			errMsg:  "guild ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing test data
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			// Run setup
			tt.setup()

			got, err := userService.GetOptedInUsers(tt.guildID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetOptedInUsers() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("GetOptedInUsers() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("GetOptedInUsers() unexpected error = %v", err)
				return
			}

			// Compare slices (order doesn't matter for this test)
			if len(got) != len(tt.want) {
				t.Errorf("GetOptedInUsers() length = %v, want %v", len(got), len(tt.want))
				return
			}

			// Create maps for easier comparison
			gotMap := make(map[string]bool)
			for _, user := range got {
				gotMap[user] = true
			}

			wantMap := make(map[string]bool)
			for _, user := range tt.want {
				wantMap[user] = true
			}

			for user := range wantMap {
				if !gotMap[user] {
					t.Errorf("GetOptedInUsers() missing user %v", user)
				}
			}

			for user := range gotMap {
				if !wantMap[user] {
					t.Errorf("GetOptedInUsers() unexpected user %v", user)
				}
			}
		})
	}
}

func TestUserService_AutoOptIn(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	tests := []struct {
		name    string
		userID  string
		guildID string
		setup   func() // Function to set up test data
		wantErr bool
		errMsg  string
		checkFn func(t *testing.T) // Function to verify the result
	}{
		{
			name:    "auto opt-in new user",
			userID:  "user123",
			guildID: "guild456",
			setup:   func() {}, // No setup needed
			wantErr: false,
			checkFn: func(t *testing.T) {
				isOptedIn, err := userService.IsOptedIn("user123", "guild456")
				if err != nil {
					t.Errorf("IsOptedIn() error = %v", err)
				}
				if !isOptedIn {
					t.Errorf("User should be opted in after AutoOptIn()")
				}
			},
		},
		{
			name:    "auto opt-in already opted-in user",
			userID:  "user123",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			wantErr: false,
			checkFn: func(t *testing.T) {
				isOptedIn, err := userService.IsOptedIn("user123", "guild456")
				if err != nil {
					t.Errorf("IsOptedIn() error = %v", err)
				}
				if !isOptedIn {
					t.Errorf("User should remain opted in after AutoOptIn()")
				}
			},
		},
		{
			name:    "auto opt-in previously opted-out user",
			userID:  "user123",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", false); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			wantErr: false,
			checkFn: func(t *testing.T) {
				isOptedIn, err := userService.IsOptedIn("user123", "guild456")
				if err != nil {
					t.Errorf("IsOptedIn() error = %v", err)
				}
				if !isOptedIn {
					t.Errorf("User should be opted in after AutoOptIn()")
				}
			},
		},
		{
			name:    "empty user ID",
			userID:  "",
			guildID: "guild456",
			setup:   func() {},
			wantErr: true,
			errMsg:  "user ID cannot be empty",
			checkFn: func(t *testing.T) {},
		},
		{
			name:    "empty guild ID",
			userID:  "user123",
			guildID: "",
			setup:   func() {},
			wantErr: true,
			errMsg:  "guild ID cannot be empty",
			checkFn: func(t *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing test data
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			// Run setup
			tt.setup()

			err := userService.AutoOptIn(tt.userID, tt.guildID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AutoOptIn() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("AutoOptIn() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("AutoOptIn() unexpected error = %v", err)
				return
			}

			// Run additional checks
			tt.checkFn(t)
		})
	}
}

func TestUserService_GetUserPreferences(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	tests := []struct {
		name    string
		userID  string
		guildID string
		setup   func() // Function to set up test data
		wantErr bool
		errMsg  string
		checkFn func(t *testing.T, prefs *UserTTSPreferences) // Function to verify the result
	}{
		{
			name:    "get default preferences",
			userID:  "user123",
			guildID: "guild456",
			setup:   func() {}, // No setup needed
			wantErr: false,
			checkFn: func(t *testing.T, prefs *UserTTSPreferences) {
				if prefs.UserID != "user123" {
					t.Errorf("UserID = %v, want %v", prefs.UserID, "user123")
				}
				if prefs.GuildID != "guild456" {
					t.Errorf("GuildID = %v, want %v", prefs.GuildID, "guild456")
				}
				if prefs.OptedIn != false {
					t.Errorf("OptedIn = %v, want %v", prefs.OptedIn, false)
				}
			},
		},
		{
			name:    "get existing preferences",
			userID:  "user123",
			guildID: "guild456",
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			wantErr: false,
			checkFn: func(t *testing.T, prefs *UserTTSPreferences) {
				if prefs.UserID != "user123" {
					t.Errorf("UserID = %v, want %v", prefs.UserID, "user123")
				}
				if prefs.GuildID != "guild456" {
					t.Errorf("GuildID = %v, want %v", prefs.GuildID, "guild456")
				}
				if prefs.OptedIn != true {
					t.Errorf("OptedIn = %v, want %v", prefs.OptedIn, true)
				}
			},
		},
		{
			name:    "empty user ID",
			userID:  "",
			guildID: "guild456",
			setup:   func() {},
			wantErr: true,
			errMsg:  "user ID cannot be empty",
			checkFn: func(t *testing.T, prefs *UserTTSPreferences) {},
		},
		{
			name:    "empty guild ID",
			userID:  "user123",
			guildID: "",
			setup:   func() {},
			wantErr: true,
			errMsg:  "guild ID cannot be empty",
			checkFn: func(t *testing.T, prefs *UserTTSPreferences) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing test data
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			// Run setup
			tt.setup()

			prefs, err := userService.GetUserPreferences(tt.userID, tt.guildID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetUserPreferences() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("GetUserPreferences() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("GetUserPreferences() unexpected error = %v", err)
				return
			}

			// Run additional checks
			tt.checkFn(t, prefs)
		})
	}
}

func TestUserService_UpdateUserSettings(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	tests := []struct {
		name     string
		userID   string
		guildID  string
		settings UserTTSSettings
		setup    func() // Function to set up test data
		wantErr  bool
		errMsg   string
		checkFn  func(t *testing.T) // Function to verify the result
	}{
		{
			name:    "update settings for new user",
			userID:  "user123",
			guildID: "guild456",
			settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-B",
				SpeedModifier:  1.5,
			},
			setup:   func() {},
			wantErr: false,
			checkFn: func(t *testing.T) {
				prefs, err := userService.GetUserPreferences("user123", "guild456")
				if err != nil {
					t.Errorf("GetUserPreferences() error = %v", err)
				}
				if prefs.Settings.PreferredVoice != "en-US-Standard-B" {
					t.Errorf("PreferredVoice = %v, want %v", prefs.Settings.PreferredVoice, "en-US-Standard-B")
				}
				if prefs.Settings.SpeedModifier != 1.5 {
					t.Errorf("SpeedModifier = %v, want %v", prefs.Settings.SpeedModifier, 1.5)
				}
			},
		},
		{
			name:    "update settings for existing user",
			userID:  "user123",
			guildID: "guild456",
			settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-C",
				SpeedModifier:  0.8,
			},
			setup: func() {
				if err := userService.SetOptInStatus("user123", "guild456", true); err != nil {
					t.Fatalf("Failed to set opt-in status: %v", err)
				}
			},
			wantErr: false,
			checkFn: func(t *testing.T) {
				prefs, err := userService.GetUserPreferences("user123", "guild456")
				if err != nil {
					t.Errorf("GetUserPreferences() error = %v", err)
				}
				if prefs.Settings.PreferredVoice != "en-US-Standard-C" {
					t.Errorf("PreferredVoice = %v, want %v", prefs.Settings.PreferredVoice, "en-US-Standard-C")
				}
				if prefs.Settings.SpeedModifier != 0.8 {
					t.Errorf("SpeedModifier = %v, want %v", prefs.Settings.SpeedModifier, 0.8)
				}
				// Opt-in status should be preserved
				if !prefs.OptedIn {
					t.Errorf("OptedIn should be preserved as true")
				}
			},
		},
		{
			name:    "invalid speed modifier (too low)",
			userID:  "user123",
			guildID: "guild456",
			settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-A",
				SpeedModifier:  0.1, // Too low
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "invalid user settings: speed modifier must be between 0.25 and 4.0",
			checkFn: func(t *testing.T) {},
		},
		{
			name:    "invalid speed modifier (too high)",
			userID:  "user123",
			guildID: "guild456",
			settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-A",
				SpeedModifier:  5.0, // Too high
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "invalid user settings: speed modifier must be between 0.25 and 4.0",
			checkFn: func(t *testing.T) {},
		},
		{
			name:    "empty preferred voice",
			userID:  "user123",
			guildID: "guild456",
			settings: UserTTSSettings{
				PreferredVoice: "", // Empty
				SpeedModifier:  1.0,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "invalid user settings: preferred voice is required",
			checkFn: func(t *testing.T) {},
		},
		{
			name:    "empty user ID",
			userID:  "",
			guildID: "guild456",
			settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-A",
				SpeedModifier:  1.0,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "user ID cannot be empty",
			checkFn: func(t *testing.T) {},
		},
		{
			name:    "empty guild ID",
			userID:  "user123",
			guildID: "",
			settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-A",
				SpeedModifier:  1.0,
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "guild ID cannot be empty",
			checkFn: func(t *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing test data
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			// Run setup
			tt.setup()

			err := userService.UpdateUserSettings(tt.userID, tt.guildID, tt.settings)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateUserSettings() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("UpdateUserSettings() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateUserSettings() unexpected error = %v", err)
				return
			}

			// Run additional checks
			tt.checkFn(t)
		})
	}
}

func TestUserService_Integration(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "tts_user_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create storage service and user service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userService := NewUserService(storage)

	// Test complete user workflow
	t.Run("complete user workflow", func(t *testing.T) {
		userID := "user123"
		guildID := "guild456"

		// 1. Check initial state (should not be opted in)
		isOptedIn, err := userService.IsOptedIn(userID, guildID)
		if err != nil {
			t.Errorf("IsOptedIn() error = %v", err)
		}
		if isOptedIn {
			t.Errorf("User should not be opted in initially")
		}

		// 2. Auto opt-in user (simulating bot invitation)
		err = userService.AutoOptIn(userID, guildID)
		if err != nil {
			t.Errorf("AutoOptIn() error = %v", err)
		}

		// 3. Verify user is now opted in
		isOptedIn, err = userService.IsOptedIn(userID, guildID)
		if err != nil {
			t.Errorf("IsOptedIn() error = %v", err)
		}
		if !isOptedIn {
			t.Errorf("User should be opted in after AutoOptIn()")
		}

		// 4. Update user settings
		newSettings := UserTTSSettings{
			PreferredVoice: "en-US-Standard-B",
			SpeedModifier:  1.2,
		}
		err = userService.UpdateUserSettings(userID, guildID, newSettings)
		if err != nil {
			t.Errorf("UpdateUserSettings() error = %v", err)
		}

		// 5. Verify settings were updated and opt-in status preserved
		prefs, err := userService.GetUserPreferences(userID, guildID)
		if err != nil {
			t.Errorf("GetUserPreferences() error = %v", err)
		}
		if !prefs.OptedIn {
			t.Errorf("Opt-in status should be preserved")
		}
		if prefs.Settings.PreferredVoice != "en-US-Standard-B" {
			t.Errorf("PreferredVoice = %v, want %v", prefs.Settings.PreferredVoice, "en-US-Standard-B")
		}
		if prefs.Settings.SpeedModifier != 1.2 {
			t.Errorf("SpeedModifier = %v, want %v", prefs.Settings.SpeedModifier, 1.2)
		}

		// 6. Opt out user
		err = userService.SetOptInStatus(userID, guildID, false)
		if err != nil {
			t.Errorf("SetOptInStatus() error = %v", err)
		}

		// 7. Verify user is opted out but settings preserved
		prefs, err = userService.GetUserPreferences(userID, guildID)
		if err != nil {
			t.Errorf("GetUserPreferences() error = %v", err)
		}
		if prefs.OptedIn {
			t.Errorf("User should be opted out")
		}
		if prefs.Settings.PreferredVoice != "en-US-Standard-B" {
			t.Errorf("Settings should be preserved after opt-out")
		}

		// 8. Check that user is not in opted-in users list
		optedInUsers, err := userService.GetOptedInUsers(guildID)
		if err != nil {
			t.Errorf("GetOptedInUsers() error = %v", err)
		}
		for _, user := range optedInUsers {
			if user == userID {
				t.Errorf("User should not be in opted-in users list")
			}
		}
	})

	// Test multiple users in same guild
	t.Run("multiple users in same guild", func(t *testing.T) {
		guildID := "guild789"
		users := []string{"user1", "user2", "user3", "user4"}

		// Opt in first 3 users
		for i, userID := range users[:3] {
			err := userService.SetOptInStatus(userID, guildID, true)
			if err != nil {
				t.Errorf("SetOptInStatus() for user %d error = %v", i, err)
			}
		}

		// Leave user4 opted out (default)

		// Get opted-in users
		optedInUsers, err := userService.GetOptedInUsers(guildID)
		if err != nil {
			t.Errorf("GetOptedInUsers() error = %v", err)
		}

		// Should have 3 opted-in users
		if len(optedInUsers) != 3 {
			t.Errorf("Expected 3 opted-in users, got %d", len(optedInUsers))
		}

		// Verify correct users are opted in
		optedInMap := make(map[string]bool)
		for _, user := range optedInUsers {
			optedInMap[user] = true
		}

		for _, userID := range users[:3] {
			if !optedInMap[userID] {
				t.Errorf("User %s should be opted in", userID)
			}
		}

		if optedInMap["user4"] {
			t.Errorf("User4 should not be opted in")
		}
	})
}
