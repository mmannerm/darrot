package tts

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func TestInMemoryMessageQueue_Enqueue(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	message := &QueuedMessage{
		ID:        "msg1",
		GuildID:   "guild1",
		ChannelID: "channel1",
		UserID:    "user1",
		Username:  "TestUser",
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	// Test successful enqueue
	err := queue.Enqueue(message)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check queue size
	size := queue.Size("guild1")
	if size != 1 {
		t.Errorf("Expected queue size 1, got %d", size)
	}

	// Test enqueue with nil message
	err = queue.Enqueue(nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}

	// Test enqueue with empty guild ID
	invalidMessage := &QueuedMessage{
		ID:      "msg2",
		GuildID: "",
		Content: "Invalid message",
	}
	err = queue.Enqueue(invalidMessage)
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryMessageQueue_Dequeue(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Test dequeue from empty queue
	message, err := queue.Dequeue("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if message != nil {
		t.Error("Expected nil message from empty queue")
	}

	// Add messages to queue
	msg1 := &QueuedMessage{
		ID:        "msg1",
		GuildID:   "guild1",
		Content:   "First message",
		Timestamp: time.Now(),
	}
	msg2 := &QueuedMessage{
		ID:        "msg2",
		GuildID:   "guild1",
		Content:   "Second message",
		Timestamp: time.Now().Add(time.Second),
	}

	queue.Enqueue(msg1)
	queue.Enqueue(msg2)

	// Test FIFO behavior
	dequeuedMsg, err := queue.Dequeue("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if dequeuedMsg == nil {
		t.Fatal("Expected message, got nil")
	}
	if dequeuedMsg.ID != "msg1" {
		t.Errorf("Expected first message ID 'msg1', got %s", dequeuedMsg.ID)
	}

	// Check remaining queue size
	size := queue.Size("guild1")
	if size != 1 {
		t.Errorf("Expected queue size 1 after dequeue, got %d", size)
	}

	// Test dequeue with empty guild ID
	_, err = queue.Dequeue("")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryMessageQueue_Clear(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Add messages to queue
	msg1 := &QueuedMessage{ID: "msg1", GuildID: "guild1", Content: "Message 1"}
	msg2 := &QueuedMessage{ID: "msg2", GuildID: "guild1", Content: "Message 2"}

	queue.Enqueue(msg1)
	queue.Enqueue(msg2)

	// Verify messages are in queue
	if queue.Size("guild1") != 2 {
		t.Error("Expected 2 messages in queue before clear")
	}

	// Clear queue
	err := queue.Clear("guild1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify queue is empty
	if queue.Size("guild1") != 0 {
		t.Error("Expected empty queue after clear")
	}

	// Test clear with empty guild ID
	err = queue.Clear("")
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryMessageQueue_SetMaxSize(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Test setting max size
	err := queue.SetMaxSize("guild1", 5)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test invalid max size
	err = queue.SetMaxSize("guild1", 0)
	if err == nil {
		t.Error("Expected error for max size 0")
	}

	err = queue.SetMaxSize("guild1", -1)
	if err == nil {
		t.Error("Expected error for negative max size")
	}

	// Test empty guild ID
	err = queue.SetMaxSize("", 5)
	if err == nil {
		t.Error("Expected error for empty guild ID")
	}
}

func TestInMemoryMessageQueue_MaxSizeEnforcement(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Set small max size
	queue.SetMaxSize("guild1", 2)

	// Add messages up to max size
	msg1 := &QueuedMessage{ID: "msg1", GuildID: "guild1", Content: "Message 1"}
	msg2 := &QueuedMessage{ID: "msg2", GuildID: "guild1", Content: "Message 2"}
	msg3 := &QueuedMessage{ID: "msg3", GuildID: "guild1", Content: "Message 3"}

	queue.Enqueue(msg1)
	queue.Enqueue(msg2)

	// Queue should be at max size
	if queue.Size("guild1") != 2 {
		t.Errorf("Expected queue size 2, got %d", queue.Size("guild1"))
	}

	// Adding another message should remove the oldest
	queue.Enqueue(msg3)

	// Queue size should still be 2
	if queue.Size("guild1") != 2 {
		t.Errorf("Expected queue size 2 after overflow, got %d", queue.Size("guild1"))
	}

	// First message should be msg2 (msg1 was removed)
	dequeuedMsg, _ := queue.Dequeue("guild1")
	if dequeuedMsg.ID != "msg2" {
		t.Errorf("Expected first message to be 'msg2', got %s", dequeuedMsg.ID)
	}
}

func TestInMemoryMessageQueue_GetMaxSize(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Test default max size
	defaultSize := queue.GetMaxSize("guild1")
	if defaultSize != 10 {
		t.Errorf("Expected default max size 10, got %d", defaultSize)
	}

	// Test custom max size
	queue.SetMaxSize("guild1", 15)
	customSize := queue.GetMaxSize("guild1")
	if customSize != 15 {
		t.Errorf("Expected custom max size 15, got %d", customSize)
	}
}

func TestInMemoryMessageQueue_MultipleGuilds(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Add messages for different guilds
	msg1 := &QueuedMessage{ID: "msg1", GuildID: "guild1", Content: "Guild 1 message"}
	msg2 := &QueuedMessage{ID: "msg2", GuildID: "guild2", Content: "Guild 2 message"}

	queue.Enqueue(msg1)
	queue.Enqueue(msg2)

	// Check that guilds have separate queues
	if queue.Size("guild1") != 1 {
		t.Errorf("Expected guild1 queue size 1, got %d", queue.Size("guild1"))
	}
	if queue.Size("guild2") != 1 {
		t.Errorf("Expected guild2 queue size 1, got %d", queue.Size("guild2"))
	}

	// Clear one guild shouldn't affect the other
	queue.Clear("guild1")
	if queue.Size("guild1") != 0 {
		t.Error("Expected guild1 queue to be empty after clear")
	}
	if queue.Size("guild2") != 1 {
		t.Error("Expected guild2 queue to still have 1 message")
	}
}

func TestInMemoryMessageQueue_ConcurrentAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	queue := NewInMemoryMessageQueue(logger)

	// Test concurrent enqueue operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			msg := &QueuedMessage{
				ID:      fmt.Sprintf("msg%d", i),
				GuildID: "guild1",
				Content: fmt.Sprintf("Message %d", i),
			}
			queue.Enqueue(msg)
		}
		done <- true
	}()

	go func() {
		for i := 100; i < 200; i++ {
			msg := &QueuedMessage{
				ID:      fmt.Sprintf("msg%d", i),
				GuildID: "guild1",
				Content: fmt.Sprintf("Message %d", i),
			}
			queue.Enqueue(msg)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Check that all messages were added (considering default max size of 10)
	size := queue.Size("guild1")
	if size != 10 {
		t.Errorf("Expected queue size 10 (max size), got %d", size)
	}
}
