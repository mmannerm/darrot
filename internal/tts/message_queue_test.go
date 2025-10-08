package tts

import (
	"fmt"
	"testing"
	"time"
)

func TestNewMessageQueue(t *testing.T) {
	mq := NewMessageQueue()
	if mq == nil {
		t.Fatal("NewMessageQueue() returned nil")
	}

	// Verify it implements the interface
	var _ MessageQueue = mq
}

func TestMessageQueue_Enqueue(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	message := &QueuedMessage{
		ID:        "msg-1",
		GuildID:   guildID,
		ChannelID: "channel-1",
		UserID:    "user-1",
		Username:  "testuser",
		Content:   "Hello world",
		Timestamp: time.Now(),
	}

	// Test successful enqueue
	err := mq.Enqueue(message)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	// Verify message was added
	size := mq.Size(guildID)
	if size != 1 {
		t.Errorf("Expected queue size 1, got %d", size)
	}
}

func TestMessageQueue_Enqueue_ValidationErrors(t *testing.T) {
	mq := NewMessageQueue()

	tests := []struct {
		name    string
		message *QueuedMessage
		wantErr bool
	}{
		{
			name:    "nil message",
			message: nil,
			wantErr: true,
		},
		{
			name: "empty guild ID",
			message: &QueuedMessage{
				ID:      "msg-1",
				GuildID: "",
				Content: "test",
			},
			wantErr: true,
		},
		{
			name: "empty content",
			message: &QueuedMessage{
				ID:      "msg-1",
				GuildID: "guild-1",
				Content: "",
			},
			wantErr: true,
		},
		{
			name: "valid message",
			message: &QueuedMessage{
				ID:      "msg-1",
				GuildID: "guild-1",
				Content: "test content",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mq.Enqueue(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Enqueue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageQueue_Dequeue(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Test dequeue from empty queue
	msg, err := mq.Dequeue(guildID)
	if err != nil {
		t.Fatalf("Dequeue() from empty queue failed: %v", err)
	}
	if msg != nil {
		t.Errorf("Expected nil message from empty queue, got %v", msg)
	}

	// Add a message
	originalMessage := &QueuedMessage{
		ID:        "msg-1",
		GuildID:   guildID,
		ChannelID: "channel-1",
		UserID:    "user-1",
		Username:  "testuser",
		Content:   "Hello world",
		Timestamp: time.Now(),
	}

	err = mq.Enqueue(originalMessage)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	// Test successful dequeue
	dequeuedMessage, err := mq.Dequeue(guildID)
	if err != nil {
		t.Fatalf("Dequeue() failed: %v", err)
	}

	if dequeuedMessage == nil {
		t.Fatal("Dequeue() returned nil message")
	}

	if dequeuedMessage.ID != originalMessage.ID {
		t.Errorf("Expected message ID %s, got %s", originalMessage.ID, dequeuedMessage.ID)
	}

	// Verify queue is now empty
	size := mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected queue size 0 after dequeue, got %d", size)
	}
}

func TestMessageQueue_Dequeue_EmptyGuildID(t *testing.T) {
	mq := NewMessageQueue()

	_, err := mq.Dequeue("")
	if err == nil {
		t.Error("Expected error for empty guild ID, got nil")
	}
}

func TestMessageQueue_Clear(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Add multiple messages
	for i := 0; i < 5; i++ {
		message := &QueuedMessage{
			ID:      fmt.Sprintf("msg-%d", i),
			GuildID: guildID,
			Content: fmt.Sprintf("Message %d", i),
		}
		err := mq.Enqueue(message)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}
	}

	// Verify messages were added
	size := mq.Size(guildID)
	if size != 5 {
		t.Errorf("Expected queue size 5, got %d", size)
	}

	// Clear the queue
	err := mq.Clear(guildID)
	if err != nil {
		t.Fatalf("Clear() failed: %v", err)
	}

	// Verify queue is empty
	size = mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected queue size 0 after clear, got %d", size)
	}
}

func TestMessageQueue_Clear_EmptyGuildID(t *testing.T) {
	mq := NewMessageQueue()

	err := mq.Clear("")
	if err == nil {
		t.Error("Expected error for empty guild ID, got nil")
	}
}

func TestMessageQueue_Size(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Test size of non-existent queue
	size := mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected size 0 for non-existent queue, got %d", size)
	}

	// Test size with empty guild ID
	size = mq.Size("")
	if size != 0 {
		t.Errorf("Expected size 0 for empty guild ID, got %d", size)
	}

	// Add messages and test size
	for i := 0; i < 3; i++ {
		message := &QueuedMessage{
			ID:      fmt.Sprintf("msg-%d", i),
			GuildID: guildID,
			Content: fmt.Sprintf("Message %d", i),
		}
		err := mq.Enqueue(message)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}

		expectedSize := i + 1
		size = mq.Size(guildID)
		if size != expectedSize {
			t.Errorf("Expected size %d, got %d", expectedSize, size)
		}
	}
}

func TestMessageQueue_SetMaxSize(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Test setting max size for new guild
	err := mq.SetMaxSize(guildID, 5)
	if err != nil {
		t.Fatalf("SetMaxSize() failed: %v", err)
	}

	// Test validation errors
	err = mq.SetMaxSize("", 5)
	if err == nil {
		t.Error("Expected error for empty guild ID, got nil")
	}

	err = mq.SetMaxSize(guildID, 0)
	if err == nil {
		t.Error("Expected error for zero max size, got nil")
	}

	err = mq.SetMaxSize(guildID, -1)
	if err == nil {
		t.Error("Expected error for negative max size, got nil")
	}
}

// TestMessageQueue_QueueOverflow tests Requirement 4.3:
// When the message queue exceeds 10 messages, skip older messages and indicate the skip
func TestMessageQueue_QueueOverflow(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Set max size to 3 for easier testing
	err := mq.SetMaxSize(guildID, 3)
	if err != nil {
		t.Fatalf("SetMaxSize() failed: %v", err)
	}

	// Add messages up to max capacity
	for i := 0; i < 3; i++ {
		message := &QueuedMessage{
			ID:      fmt.Sprintf("msg-%d", i),
			GuildID: guildID,
			Content: fmt.Sprintf("Message %d", i),
		}
		err := mq.Enqueue(message)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}
	}

	// Verify queue is at max capacity
	size := mq.Size(guildID)
	if size != 3 {
		t.Errorf("Expected queue size 3, got %d", size)
	}

	// Add one more message to trigger overflow
	overflowMessage := &QueuedMessage{
		ID:      "msg-overflow",
		GuildID: guildID,
		Content: "Overflow message",
	}
	err = mq.Enqueue(overflowMessage)
	if err != nil {
		t.Fatalf("Enqueue() overflow failed: %v", err)
	}

	// Verify queue size is still at max
	size = mq.Size(guildID)
	if size != 3 {
		t.Errorf("Expected queue size 3 after overflow, got %d", size)
	}

	// Verify the oldest message was removed and newest was added
	firstMessage, err := mq.Dequeue(guildID)
	if err != nil {
		t.Fatalf("Dequeue() failed: %v", err)
	}

	// The first message should now be "msg-1" (msg-0 was removed due to overflow)
	if firstMessage.ID != "msg-1" {
		t.Errorf("Expected first message ID 'msg-1', got '%s'", firstMessage.ID)
	}
}

// TestMessageQueue_SequentialProcessing tests Requirement 4.1:
// When multiple messages arrive quickly, queue messages and read them sequentially
func TestMessageQueue_SequentialProcessing(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Add multiple messages quickly
	messageIDs := []string{"msg-1", "msg-2", "msg-3", "msg-4", "msg-5"}
	for _, id := range messageIDs {
		message := &QueuedMessage{
			ID:      id,
			GuildID: guildID,
			Content: "Test message",
		}
		err := mq.Enqueue(message)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}
	}

	// Verify all messages were queued
	size := mq.Size(guildID)
	if size != len(messageIDs) {
		t.Errorf("Expected queue size %d, got %d", len(messageIDs), size)
	}

	// Dequeue messages and verify they come out in order (FIFO)
	for i, expectedID := range messageIDs {
		message, err := mq.Dequeue(guildID)
		if err != nil {
			t.Fatalf("Dequeue() failed at index %d: %v", i, err)
		}

		if message == nil {
			t.Fatalf("Dequeue() returned nil at index %d", i)
		}

		if message.ID != expectedID {
			t.Errorf("Expected message ID '%s' at index %d, got '%s'", expectedID, i, message.ID)
		}
	}

	// Verify queue is empty
	size = mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected queue size 0 after processing all messages, got %d", size)
	}
}

func TestMessageQueue_MultipleGuilds(t *testing.T) {
	mq := NewMessageQueue()
	guild1 := "guild-1"
	guild2 := "guild-2"

	// Add messages to different guilds
	message1 := &QueuedMessage{
		ID:      "msg-1",
		GuildID: guild1,
		Content: "Message for guild 1",
	}
	message2 := &QueuedMessage{
		ID:      "msg-2",
		GuildID: guild2,
		Content: "Message for guild 2",
	}

	err := mq.Enqueue(message1)
	if err != nil {
		t.Fatalf("Enqueue() for guild1 failed: %v", err)
	}

	err = mq.Enqueue(message2)
	if err != nil {
		t.Fatalf("Enqueue() for guild2 failed: %v", err)
	}

	// Verify each guild has its own queue
	size1 := mq.Size(guild1)
	size2 := mq.Size(guild2)

	if size1 != 1 {
		t.Errorf("Expected guild1 queue size 1, got %d", size1)
	}

	if size2 != 1 {
		t.Errorf("Expected guild2 queue size 1, got %d", size2)
	}

	// Verify messages are isolated per guild
	msg1, err := mq.Dequeue(guild1)
	if err != nil {
		t.Fatalf("Dequeue() for guild1 failed: %v", err)
	}

	if msg1.ID != "msg-1" {
		t.Errorf("Expected message ID 'msg-1' for guild1, got '%s'", msg1.ID)
	}

	msg2, err := mq.Dequeue(guild2)
	if err != nil {
		t.Fatalf("Dequeue() for guild2 failed: %v", err)
	}

	if msg2.ID != "msg-2" {
		t.Errorf("Expected message ID 'msg-2' for guild2, got '%s'", msg2.ID)
	}
}

func TestMessageQueue_GetLastActivity(t *testing.T) {
	mq := NewMessageQueue().(*MessageQueueImpl)
	guildID := "test-guild-123"

	// Test non-existent guild
	lastActivity := mq.GetLastActivity(guildID)
	if !lastActivity.IsZero() {
		t.Errorf("Expected zero time for non-existent guild, got %v", lastActivity)
	}

	// Add a message to create the guild queue
	message := &QueuedMessage{
		ID:      "msg-1",
		GuildID: guildID,
		Content: "Test message",
	}

	beforeEnqueue := time.Now()
	err := mq.Enqueue(message)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}
	afterEnqueue := time.Now()

	// Check that last activity was updated
	lastActivity = mq.GetLastActivity(guildID)
	if lastActivity.Before(beforeEnqueue) || lastActivity.After(afterEnqueue) {
		t.Errorf("Last activity time %v not within expected range [%v, %v]",
			lastActivity, beforeEnqueue, afterEnqueue)
	}
}

func TestMessageQueue_CheckInactivity(t *testing.T) {
	mq := NewMessageQueue().(*MessageQueueImpl)
	guildID := "test-guild-123"

	// Test non-existent guild
	isInactive := mq.CheckInactivity(guildID, time.Minute)
	if isInactive {
		t.Error("Expected false for non-existent guild, got true")
	}

	// Add a message to create the guild queue
	message := &QueuedMessage{
		ID:      "msg-1",
		GuildID: guildID,
		Content: "Test message",
	}

	err := mq.Enqueue(message)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	// Check inactivity immediately (should be false)
	isInactive = mq.CheckInactivity(guildID, time.Minute)
	if isInactive {
		t.Error("Expected false for recent activity, got true")
	}

	// Wait a small amount and check inactivity with very short timeout (should be true)
	time.Sleep(time.Millisecond)
	isInactive = mq.CheckInactivity(guildID, time.Nanosecond)
	if !isInactive {
		t.Error("Expected true for expired timeout, got false")
	}
}

func TestMessageQueue_SetInactivityCallback(t *testing.T) {
	mq := NewMessageQueue().(*MessageQueueImpl)
	guildID := "test-guild-123"

	callback := func(id string) {
		// Callback implementation for testing
	}

	// Test setting callback
	err := mq.SetInactivityCallback(guildID, callback)
	if err != nil {
		t.Fatalf("SetInactivityCallback() failed: %v", err)
	}

	// Test validation error
	err = mq.SetInactivityCallback("", callback)
	if err == nil {
		t.Error("Expected error for empty guild ID, got nil")
	}

	// Verify callback was set (this is internal state, so we can't directly test it)
	// but we can verify the guild queue was created
	size := mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected size 0 for new guild queue, got %d", size)
	}
}

func TestMessageQueue_RemoveGuild(t *testing.T) {
	mq := NewMessageQueue().(*MessageQueueImpl)
	guildID := "test-guild-123"

	// Add a message to create the guild queue
	message := &QueuedMessage{
		ID:      "msg-1",
		GuildID: guildID,
		Content: "Test message",
	}

	err := mq.Enqueue(message)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	// Verify guild queue exists
	size := mq.Size(guildID)
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Remove guild
	err = mq.RemoveGuild(guildID)
	if err != nil {
		t.Fatalf("RemoveGuild() failed: %v", err)
	}

	// Verify guild queue was removed
	size = mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected size 0 after removal, got %d", size)
	}

	// Test validation error
	err = mq.RemoveGuild("")
	if err == nil {
		t.Error("Expected error for empty guild ID, got nil")
	}
}

func TestMessageQueue_GetAllGuilds(t *testing.T) {
	mq := NewMessageQueue().(*MessageQueueImpl)

	// Test empty queue
	guilds := mq.GetAllGuilds()
	if len(guilds) != 0 {
		t.Errorf("Expected 0 guilds, got %d", len(guilds))
	}

	// Add messages for different guilds
	guildIDs := []string{"guild-1", "guild-2", "guild-3"}
	for _, guildID := range guildIDs {
		message := &QueuedMessage{
			ID:      "msg-1",
			GuildID: guildID,
			Content: "Test message",
		}
		err := mq.Enqueue(message)
		if err != nil {
			t.Fatalf("Enqueue() failed for guild %s: %v", guildID, err)
		}
	}

	// Get all guilds
	guilds = mq.GetAllGuilds()
	if len(guilds) != len(guildIDs) {
		t.Errorf("Expected %d guilds, got %d", len(guildIDs), len(guilds))
	}

	// Verify all guild IDs are present
	guildMap := make(map[string]bool)
	for _, guild := range guilds {
		guildMap[guild] = true
	}

	for _, expectedGuild := range guildIDs {
		if !guildMap[expectedGuild] {
			t.Errorf("Expected guild %s not found in result", expectedGuild)
		}
	}
}

// TestMessageQueue_ConcurrentAccess tests thread safety
func TestMessageQueue_ConcurrentAccess(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Set a reasonable max size
	err := mq.SetMaxSize(guildID, 100)
	if err != nil {
		t.Fatalf("SetMaxSize() failed: %v", err)
	}

	// Run concurrent enqueue operations
	done := make(chan bool)
	numGoroutines := 10
	messagesPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < messagesPerGoroutine; j++ {
				message := &QueuedMessage{
					ID:      fmt.Sprintf("msg-%d-%d", goroutineID, j),
					GuildID: guildID,
					Content: fmt.Sprintf("Message from goroutine %d, iteration %d", goroutineID, j),
				}

				err := mq.Enqueue(message)
				if err != nil {
					t.Errorf("Enqueue() failed: %v", err)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final queue size
	finalSize := mq.Size(guildID)
	expectedSize := numGoroutines * messagesPerGoroutine
	if finalSize != expectedSize {
		t.Errorf("Expected final queue size %d, got %d", expectedSize, finalSize)
	}
}

// Tests for SkipNext method

func TestMessageQueue_SkipNext_Success(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Add multiple messages
	messages := []*QueuedMessage{
		{
			ID:       "msg-1",
			GuildID:  guildID,
			Username: "user1",
			Content:  "First message",
		},
		{
			ID:       "msg-2",
			GuildID:  guildID,
			Username: "user2",
			Content:  "Second message",
		},
		{
			ID:       "msg-3",
			GuildID:  guildID,
			Username: "user3",
			Content:  "Third message",
		},
	}

	for _, msg := range messages {
		err := mq.Enqueue(msg)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}
	}

	// Verify initial queue size
	size := mq.Size(guildID)
	if size != 3 {
		t.Errorf("Expected queue size 3, got %d", size)
	}

	// Skip the first message
	skippedMessage, err := mq.SkipNext(guildID)
	if err != nil {
		t.Fatalf("SkipNext() failed: %v", err)
	}

	// Verify the skipped message is the first one
	if skippedMessage == nil {
		t.Fatal("SkipNext() returned nil message")
	}

	if skippedMessage.ID != "msg-1" {
		t.Errorf("Expected skipped message ID 'msg-1', got '%s'", skippedMessage.ID)
	}

	if skippedMessage.Username != "user1" {
		t.Errorf("Expected skipped message username 'user1', got '%s'", skippedMessage.Username)
	}

	// Verify queue size decreased
	size = mq.Size(guildID)
	if size != 2 {
		t.Errorf("Expected queue size 2 after skip, got %d", size)
	}

	// Verify the next message in queue is now the second one
	nextMessage, err := mq.Dequeue(guildID)
	if err != nil {
		t.Fatalf("Dequeue() failed: %v", err)
	}

	if nextMessage.ID != "msg-2" {
		t.Errorf("Expected next message ID 'msg-2', got '%s'", nextMessage.ID)
	}
}

func TestMessageQueue_SkipNext_EmptyQueue(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Try to skip from empty queue
	skippedMessage, err := mq.SkipNext(guildID)
	if err != nil {
		t.Fatalf("SkipNext() from empty queue failed: %v", err)
	}

	// Should return nil message for empty queue
	if skippedMessage != nil {
		t.Errorf("Expected nil message from empty queue, got %v", skippedMessage)
	}
}

func TestMessageQueue_SkipNext_NonExistentGuild(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "non-existent-guild"

	// Try to skip from non-existent guild queue
	skippedMessage, err := mq.SkipNext(guildID)
	if err != nil {
		t.Fatalf("SkipNext() from non-existent guild failed: %v", err)
	}

	// Should return nil message for non-existent guild
	if skippedMessage != nil {
		t.Errorf("Expected nil message from non-existent guild, got %v", skippedMessage)
	}
}

func TestMessageQueue_SkipNext_EmptyGuildID(t *testing.T) {
	mq := NewMessageQueue()

	// Try to skip with empty guild ID
	_, err := mq.SkipNext("")
	if err == nil {
		t.Error("Expected error for empty guild ID, got nil")
	}

	if err.Error() != "guild ID cannot be empty" {
		t.Errorf("Expected 'guild ID cannot be empty' error, got '%s'", err.Error())
	}
}

func TestMessageQueue_SkipNext_SingleMessage(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Add single message
	message := &QueuedMessage{
		ID:       "msg-1",
		GuildID:  guildID,
		Username: "user1",
		Content:  "Only message",
	}

	err := mq.Enqueue(message)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	// Skip the only message
	skippedMessage, err := mq.SkipNext(guildID)
	if err != nil {
		t.Fatalf("SkipNext() failed: %v", err)
	}

	// Verify the message was skipped
	if skippedMessage == nil {
		t.Fatal("SkipNext() returned nil message")
	}

	if skippedMessage.ID != "msg-1" {
		t.Errorf("Expected skipped message ID 'msg-1', got '%s'", skippedMessage.ID)
	}

	// Verify queue is now empty
	size := mq.Size(guildID)
	if size != 0 {
		t.Errorf("Expected queue size 0 after skipping only message, got %d", size)
	}

	// Verify dequeue returns nil
	nextMessage, err := mq.Dequeue(guildID)
	if err != nil {
		t.Fatalf("Dequeue() failed: %v", err)
	}

	if nextMessage != nil {
		t.Errorf("Expected nil message after skipping only message, got %v", nextMessage)
	}
}

func TestMessageQueue_SkipNext_UpdatesLastActivity(t *testing.T) {
	mq := NewMessageQueue().(*MessageQueueImpl)
	guildID := "test-guild-123"

	// Add a message
	message := &QueuedMessage{
		ID:      "msg-1",
		GuildID: guildID,
		Content: "Test message",
	}

	err := mq.Enqueue(message)
	if err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	// Get initial last activity time
	initialActivity := mq.GetLastActivity(guildID)

	// Wait a small amount to ensure time difference
	time.Sleep(time.Millisecond)

	// Skip the message
	beforeSkip := time.Now()
	_, err = mq.SkipNext(guildID)
	if err != nil {
		t.Fatalf("SkipNext() failed: %v", err)
	}
	afterSkip := time.Now()

	// Verify last activity was updated
	updatedActivity := mq.GetLastActivity(guildID)

	if !updatedActivity.After(initialActivity) {
		t.Errorf("Expected last activity to be updated after skip, initial: %v, updated: %v",
			initialActivity, updatedActivity)
	}

	if updatedActivity.Before(beforeSkip) || updatedActivity.After(afterSkip) {
		t.Errorf("Last activity time %v not within expected range [%v, %v]",
			updatedActivity, beforeSkip, afterSkip)
	}
}

func TestMessageQueue_SkipNext_ConcurrentAccess(t *testing.T) {
	mq := NewMessageQueue()
	guildID := "test-guild-123"

	// Set a larger max size to accommodate all messages
	numMessages := 100
	err := mq.SetMaxSize(guildID, numMessages)
	if err != nil {
		t.Fatalf("SetMaxSize() failed: %v", err)
	}

	// Add multiple messages
	for i := 0; i < numMessages; i++ {
		message := &QueuedMessage{
			ID:      fmt.Sprintf("msg-%d", i),
			GuildID: guildID,
			Content: fmt.Sprintf("Message %d", i),
		}
		err := mq.Enqueue(message)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}
	}

	// Run concurrent skip operations
	done := make(chan bool)
	numGoroutines := 10
	skippedMessages := make(chan *QueuedMessage, numMessages)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for {
				skippedMessage, err := mq.SkipNext(guildID)
				if err != nil {
					t.Errorf("SkipNext() failed: %v", err)
					return
				}

				if skippedMessage == nil {
					// No more messages to skip
					return
				}

				skippedMessages <- skippedMessage
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	close(skippedMessages)

	// Count skipped messages
	skippedCount := 0
	skippedIDs := make(map[string]bool)
	for msg := range skippedMessages {
		skippedCount++
		if skippedIDs[msg.ID] {
			t.Errorf("Message ID %s was skipped multiple times", msg.ID)
		}
		skippedIDs[msg.ID] = true
	}

	// Verify all messages were skipped exactly once
	if skippedCount != numMessages {
		t.Errorf("Expected %d messages to be skipped, got %d", numMessages, skippedCount)
	}

	// Verify queue is empty
	finalSize := mq.Size(guildID)
	if finalSize != 0 {
		t.Errorf("Expected final queue size 0, got %d", finalSize)
	}
}
