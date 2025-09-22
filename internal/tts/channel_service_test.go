package tts

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestInMemoryChannelService_CreatePairing(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Test successful pairing creation
	err := service.CreatePairing("guild1", "voice1", "text1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify pairing exists
	isPaired, err := service.IsChannelPaired("guild1", "text1")
	if err != nil {
		t.Errorf("Expected no error checking pairing, got %v", err)
	}
	if !isPaired {
		t.Error("Expected channel to be paired")
	}

	// Test duplicate text channel pairing
	err = service.CreatePairing("guild1", "voice2", "text1")
	if err == nil {
		t.Error("Expected error for duplicate text channel pairing")
	}

	// Test empty parameters
	err = service.CreatePairing("", "voice1", "text1")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}

	err = service.CreatePairing("guild1", "", "text1")
	if err == nil {
		t.Error("Expected error for empty voice channel ID")
	}

	err = service.CreatePairing("guild1", "voice1", "")
	if err == nil {
		t.Error("Expected error for empty text channel ID")
	}
}

func TestInMemoryChannelService_RemovePairing(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Create a pairing first
	service.CreatePairing("guild1", "voice1", "text1")

	// Test successful removal
	err := service.RemovePairing("guild1", "voice1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify pairing is removed
	isPaired, err := service.IsChannelPaired("guild1", "text1")
	if err != nil {
		t.Errorf("Expected no error checking pairing, got %v", err)
	}
	if isPaired {
		t.Error("Expected channel to not be paired after removal")
	}

	// Test removing non-existent pairing
	err = service.RemovePairing("guild1", "voice2")
	if err == nil {
		t.Error("Expected error for non-existent pairing")
	}

	// Test empty parameters
	err = service.RemovePairing("", "voice1")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}

	err = service.RemovePairing("guild1", "")
	if err == nil {
		t.Error("Expected error for empty voice channel ID")
	}
}

func TestInMemoryChannelService_GetPairing(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Create a pairing
	service.CreatePairing("guild1", "voice1", "text1")

	// Test successful retrieval
	pairing, err := service.GetPairing("guild1", "voice1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if pairing == nil {
		t.Fatal("Expected pairing, got nil")
	}

	// Verify pairing details
	if pairing.GuildID != "guild1" {
		t.Errorf("Expected guild ID 'guild1', got %s", pairing.GuildID)
	}
	if pairing.VoiceChannelID != "voice1" {
		t.Errorf("Expected voice channel ID 'voice1', got %s", pairing.VoiceChannelID)
	}
	if pairing.TextChannelID != "text1" {
		t.Errorf("Expected text channel ID 'text1', got %s", pairing.TextChannelID)
	}
	if pairing.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Test non-existent pairing
	_, err = service.GetPairing("guild1", "voice2")
	if err == nil {
		t.Error("Expected error for non-existent pairing")
	}

	// Test wrong guild
	_, err = service.GetPairing("guild2", "voice1")
	if err == nil {
		t.Error("Expected error for wrong guild")
	}

	// Test empty parameters
	_, err = service.GetPairing("", "voice1")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}

	_, err = service.GetPairing("guild1", "")
	if err == nil {
		t.Error("Expected error for empty voice channel ID")
	}
}

func TestInMemoryChannelService_IsChannelPaired(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Test unpaired channel
	isPaired, err := service.IsChannelPaired("guild1", "text1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if isPaired {
		t.Error("Expected channel to not be paired")
	}

	// Create a pairing
	service.CreatePairing("guild1", "voice1", "text1")

	// Test paired channel
	isPaired, err = service.IsChannelPaired("guild1", "text1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !isPaired {
		t.Error("Expected channel to be paired")
	}

	// Test wrong guild
	isPaired, err = service.IsChannelPaired("guild2", "text1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if isPaired {
		t.Error("Expected channel to not be paired for different guild")
	}

	// Test empty parameters
	_, err = service.IsChannelPaired("", "text1")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}

	_, err = service.IsChannelPaired("guild1", "")
	if err == nil {
		t.Error("Expected error for empty text channel ID")
	}
}

func TestInMemoryChannelService_GetPairingByTextChannel(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Create a pairing
	service.CreatePairing("guild1", "voice1", "text1")

	// Test successful retrieval by text channel
	pairing, err := service.GetPairingByTextChannel("guild1", "text1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if pairing == nil {
		t.Fatal("Expected pairing, got nil")
	}

	// Verify pairing details
	if pairing.VoiceChannelID != "voice1" {
		t.Errorf("Expected voice channel ID 'voice1', got %s", pairing.VoiceChannelID)
	}
	if pairing.TextChannelID != "text1" {
		t.Errorf("Expected text channel ID 'text1', got %s", pairing.TextChannelID)
	}

	// Test non-existent pairing
	_, err = service.GetPairingByTextChannel("guild1", "text2")
	if err == nil {
		t.Error("Expected error for non-existent pairing")
	}

	// Test wrong guild
	_, err = service.GetPairingByTextChannel("guild2", "text1")
	if err == nil {
		t.Error("Expected error for wrong guild")
	}
}

func TestInMemoryChannelService_ValidateChannelAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Test valid parameters (simplified implementation always returns nil)
	err := service.ValidateChannelAccess("user1", "channel1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test empty parameters
	err = service.ValidateChannelAccess("", "channel1")
	if err == nil {
		t.Error("Expected error for empty user ID")
	}

	err = service.ValidateChannelAccess("user1", "")
	if err == nil {
		t.Error("Expected error for empty channel ID")
	}
}

func TestInMemoryChannelService_PairingReplacement(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Create initial pairing
	service.CreatePairing("guild1", "voice1", "text1")

	// Verify initial pairing
	isPaired, _ := service.IsChannelPaired("guild1", "text1")
	if !isPaired {
		t.Error("Expected initial pairing to exist")
	}

	// Create new pairing with same voice channel but different text channel
	err := service.CreatePairing("guild1", "voice1", "text2")
	if err != nil {
		t.Errorf("Expected no error replacing pairing, got %v", err)
	}

	// Verify old pairing is removed
	isPaired, _ = service.IsChannelPaired("guild1", "text1")
	if isPaired {
		t.Error("Expected old pairing to be removed")
	}

	// Verify new pairing exists
	isPaired, _ = service.IsChannelPaired("guild1", "text2")
	if !isPaired {
		t.Error("Expected new pairing to exist")
	}
}

func TestInMemoryChannelService_ConcurrentAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	service := NewInMemoryChannelService(logger)

	// Test concurrent pairing operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			service.CreatePairing("guild1", fmt.Sprintf("voice%d", i), fmt.Sprintf("text%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			service.IsChannelPaired("guild1", fmt.Sprintf("text%d", i))
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify some pairings exist
	isPaired, _ := service.IsChannelPaired("guild1", "text50")
	if !isPaired {
		t.Error("Expected at least some pairings to exist after concurrent operations")
	}
}
