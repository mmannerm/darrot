package tts

import (
	"fmt"
	"log"
	"sync"
)

// InMemoryMessageQueue provides an in-memory implementation of MessageQueue
type InMemoryMessageQueue struct {
	queues   map[string][]*QueuedMessage // guildID -> messages
	maxSizes map[string]int              // guildID -> max size
	mutex    sync.RWMutex
	logger   *log.Logger
}

// NewInMemoryMessageQueue creates a new in-memory message queue
func NewInMemoryMessageQueue(logger *log.Logger) *InMemoryMessageQueue {
	return &InMemoryMessageQueue{
		queues:   make(map[string][]*QueuedMessage),
		maxSizes: make(map[string]int),
		logger:   logger,
	}
}

// Enqueue adds a message to the queue for the specified guild
func (q *InMemoryMessageQueue) Enqueue(message *QueuedMessage) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	if message.GuildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Initialize queue if it doesn't exist
	if _, exists := q.queues[message.GuildID]; !exists {
		q.queues[message.GuildID] = make([]*QueuedMessage, 0)
		q.maxSizes[message.GuildID] = 10 // Default max size
	}

	maxSize := q.maxSizes[message.GuildID]

	// Check if queue is at capacity
	if len(q.queues[message.GuildID]) >= maxSize {
		// Remove oldest message to make room
		q.queues[message.GuildID] = q.queues[message.GuildID][1:]
		q.logger.Printf("Queue full for guild %s, removed oldest message", message.GuildID)
	}

	// Add new message to the end of the queue
	q.queues[message.GuildID] = append(q.queues[message.GuildID], message)

	q.logger.Printf("Enqueued message for guild %s, queue size: %d", message.GuildID, len(q.queues[message.GuildID]))
	return nil
}

// Dequeue removes and returns the next message from the queue for the specified guild
func (q *InMemoryMessageQueue) Dequeue(guildID string) (*QueuedMessage, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	queue, exists := q.queues[guildID]
	if !exists || len(queue) == 0 {
		return nil, nil // No messages in queue
	}

	// Get first message
	message := queue[0]

	// Remove first message from queue
	q.queues[guildID] = queue[1:]

	q.logger.Printf("Dequeued message for guild %s, remaining queue size: %d", guildID, len(q.queues[guildID]))
	return message, nil
}

// Clear removes all messages from the queue for the specified guild
func (q *InMemoryMessageQueue) Clear(guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	if _, exists := q.queues[guildID]; exists {
		q.queues[guildID] = make([]*QueuedMessage, 0)
		q.logger.Printf("Cleared message queue for guild %s", guildID)
	}

	return nil
}

// Size returns the number of messages in the queue for the specified guild
func (q *InMemoryMessageQueue) Size(guildID string) int {
	if guildID == "" {
		return 0
	}

	q.mutex.RLock()
	defer q.mutex.RUnlock()

	queue, exists := q.queues[guildID]
	if !exists {
		return 0
	}

	return len(queue)
}

// SetMaxSize sets the maximum queue size for the specified guild
func (q *InMemoryMessageQueue) SetMaxSize(guildID string, size int) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	if size <= 0 {
		return fmt.Errorf("max size must be greater than 0")
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.maxSizes[guildID] = size

	// Initialize queue if it doesn't exist
	if _, exists := q.queues[guildID]; !exists {
		q.queues[guildID] = make([]*QueuedMessage, 0)
	}

	// If current queue is larger than new max size, trim it
	if queue := q.queues[guildID]; len(queue) > size {
		// Keep the most recent messages
		startIndex := len(queue) - size
		q.queues[guildID] = queue[startIndex:]
		q.logger.Printf("Trimmed queue for guild %s to max size %d", guildID, size)
	}

	return nil
}

// GetMaxSize returns the maximum queue size for the specified guild
func (q *InMemoryMessageQueue) GetMaxSize(guildID string) int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if maxSize, exists := q.maxSizes[guildID]; exists {
		return maxSize
	}

	return 10 // Default max size
}
