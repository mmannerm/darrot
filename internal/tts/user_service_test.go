package tts

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestInMemoryUserService_SetOptInStatus(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test setting opt-in status to true
	err := service.SetOptInStatus("user1", "guild1", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify opt-in status
	optedIn, err := service.IsOptedIn("user1", "guild1")
	if err != nil {
		t.Errorf("Expected no error checking opt-in status, got %v", err)
	}
	if !optedIn {
		t.Error("Expected user to be opted in")
	}

	// Test setting opt-in status to false
	err = service.SetOptInStatus("user1", "guild1", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify opt-out status
	optedIn, err = service.IsOptedIn("user1", "guild1")
	if err != nil {
		t.Errorf("Expected no error checking opt-in status, got %v", err)
	}
	if optedIn {
		t.Error("Expected user to be opted out")
	}

	// Test empty parameters
	err = service.SetOptInStatus("", "guild1", true)
	if err == nil {
		t.Error("Expected error for empty user ID")
	}

	err = service.SetOptInStatus("user1", "", true)
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryUserService_IsOptedIn(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test default opt-in status (should be false)
	optedIn, err := service.IsOptedIn("user1", "guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if optedIn {
		t.Error("Expected default opt-in status to be false")
	}

	// Set user as opted in
	service.SetOptInStatus("user1", "guild1", true)

	// Test opted-in status
	optedIn, err = service.IsOptedIn("user1", "guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !optedIn {
		t.Error("Expected user to be opted in")
	}

	// Test different guild (should be false)
	optedIn, err = service.IsOptedIn("user1", "guild2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if optedIn {
		t.Error("Expected user to not be opted in for different guild")
	}

	// Test empty parameters
	_, err = service.IsOptedIn("", "guild1")
	if err == nil {
		t.Error("Expected error for empty user ID")
	}

	_, err = service.IsOptedIn("user1", "")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryUserService_GetOptedInUsers(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test empty guild (no opted-in users)
	users, err := service.GetOptedInUsers("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Expected 0 opted-in users, got %d", len(users))
	}

	// Add some opted-in users
	service.SetOptInStatus("user1", "guild1", true)
	service.SetOptInStatus("user2", "guild1", true)
	service.SetOptInStatus("user3", "guild1", false) // opted out
	service.SetOptInStatus("user4", "guild2", true)  // different guild

	// Test getting opted-in users for guild1
	users, err = service.GetOptedInUsers("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected 2 opted-in users for guild1, got %d", len(users))
	}

	// Verify correct users are returned
	userMap := make(map[string]bool)
	for _, user := range users {
		userMap[user] = true
	}
	if !userMap["user1"] || !userMap["user2"] {
		t.Error("Expected user1 and user2 to be in opted-in users list")
	}
	if userMap["user3"] || userMap["user4"] {
		t.Error("Expected user3 and user4 to not be in opted-in users list")
	}

	// Test empty guild ID
	_, err = service.GetOptedInUsers("")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryUserService_AutoOptIn(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test auto opt-in
	err := service.AutoOptIn("user1", "guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify user is opted in
	optedIn, err := service.IsOptedIn("user1", "guild1")
	if err != nil {
		t.Errorf("Expected no error checking opt-in status, got %v", err)
	}
	if !optedIn {
		t.Error("Expected user to be auto-opted in")
	}

	// Test empty parameters
	err = service.AutoOptIn("", "guild1")
	if err == nil {
		t.Error("Expected error for empty user ID")
	}

	err = service.AutoOptIn("user1", "")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryUserService_GetOptInCount(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test empty guild
	count, err := service.GetOptInCount("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 opted-in users, got %d", count)
	}

	// Add opted-in users
	service.SetOptInStatus("user1", "guild1", true)
	service.SetOptInStatus("user2", "guild1", true)
	service.SetOptInStatus("user3", "guild1", false)

	// Test count
	count, err = service.GetOptInCount("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 opted-in users, got %d", count)
	}
}

func TestInMemoryUserService_ClearGuildData(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Add users for multiple guilds
	service.SetOptInStatus("user1", "guild1", true)
	service.SetOptInStatus("user2", "guild1", true)
	service.SetOptInStatus("user3", "guild2", true)

	// Verify initial state
	count1, _ := service.GetOptInCount("guild1")
	count2, _ := service.GetOptInCount("guild2")
	if count1 != 2 || count2 != 1 {
		t.Error("Expected initial counts to be 2 and 1")
	}

	// Clear guild1 data
	err := service.ClearGuildData("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify guild1 data is cleared
	count1, _ = service.GetOptInCount("guild1")
	if count1 != 0 {
		t.Errorf("Expected 0 opted-in users for guild1 after clear, got %d", count1)
	}

	// Verify guild2 data is unchanged
	count2, _ = service.GetOptInCount("guild2")
	if count2 != 1 {
		t.Errorf("Expected 1 opted-in user for guild2 after clear, got %d", count2)
	}

	// Test empty guild ID
	err = service.ClearGuildData("")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryUserService_MultipleGuilds(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test same user in different guilds
	service.SetOptInStatus("user1", "guild1", true)
	service.SetOptInStatus("user1", "guild2", false)

	// Verify separate opt-in status per guild
	optedIn1, _ := service.IsOptedIn("user1", "guild1")
	optedIn2, _ := service.IsOptedIn("user1", "guild2")

	if !optedIn1 {
		t.Error("Expected user1 to be opted in for guild1")
	}
	if optedIn2 {
		t.Error("Expected user1 to be opted out for guild2")
	}
}

func TestInMemoryUserService_ConcurrentAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryUserService(logger)

	// Test concurrent opt-in operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			service.SetOptInStatus(fmt.Sprintf("user%d", i), "guild1", true)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			service.IsOptedIn(fmt.Sprintf("user%d", i), "guild1")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify some users are opted in
	count, _ := service.GetOptInCount("guild1")
	if count == 0 {
		t.Error("Expected at least some users to be opted in after concurrent operations")
	}
}
