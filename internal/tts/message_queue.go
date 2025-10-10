package tts

import (
	"errors"
	"sync"
	"time"
)

// DefaultInactivityTimeout is the default timeout for inactivity announcement
const DefaultInactivityTimeout = 5 * time.Minute

// MessageQueueImpl implements the MessageQueue interface
type MessageQueueImpl struct {
	mu     sync.RWMutex
	queues map[string]*guildQueue
}

// guildQueue represents a message queue for a specific guild
type guildQueue struct {
	messages       []*QueuedMessage
	maxSize        int
	lastActivity   time.Time
	inactivityFunc func(guildID string) // Callback for inactivity handling
}

// NewMessageQueue creates a new MessageQueue implementation
func NewMessageQueue() MessageQueue {
	return &MessageQueueImpl{
		queues: make(map[string]*guildQueue),
	}
}

// Enqueue adds a message to the queue for the specified guild
func (mq *MessageQueueImpl) Enqueue(message *QueuedMessage) error {
	if message == nil {
		return errors.New("message cannot be nil")
	}

	if message.GuildID == "" {
		return errors.New("guild ID cannot be empty")
	}

	if message.Content == "" {
		return errors.New("message content cannot be empty")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	// Get or create guild queue
	queue, exists := mq.queues[message.GuildID]
	if !exists {
		queue = &guildQueue{
			messages:     make([]*QueuedMessage, 0),
			maxSize:      DefaultMaxQueueSize,
			lastActivity: time.Now(),
		}
		mq.queues[message.GuildID] = queue
	}

	// Update last activity time
	queue.lastActivity = time.Now()

	// Check if queue is at max capacity (Requirement 4.3)
	if len(queue.messages) >= queue.maxSize {
		// Remove oldest message and indicate skip
		queue.messages = queue.messages[1:]

		// Log or handle the skip indication
		// In a real implementation, this would notify users about the skip
		// Queue overflow logging removed per user request
	}

	// Add new message to queue
	queue.messages = append(queue.messages, message)

	return nil
}

// Dequeue removes and returns the next message from the queue for the specified guild
func (mq *MessageQueueImpl) Dequeue(guildID string) (*QueuedMessage, error) {
	if guildID == "" {
		return nil, errors.New("guild ID cannot be empty")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	queue, exists := mq.queues[guildID]
	if !exists || len(queue.messages) == 0 {
		return nil, nil // No messages in queue
	}

	// Get first message
	message := queue.messages[0]

	// Remove from queue
	queue.messages = queue.messages[1:]

	// Update last activity time
	queue.lastActivity = time.Now()

	return message, nil
}

// Clear removes all messages from the queue for the specified guild
func (mq *MessageQueueImpl) Clear(guildID string) error {
	if guildID == "" {
		return errors.New("guild ID cannot be empty")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	queue, exists := mq.queues[guildID]
	if !exists {
		return nil // Nothing to clear
	}

	// Clear all messages
	queue.messages = queue.messages[:0]
	queue.lastActivity = time.Now()

	return nil
}

// Size returns the number of messages in the queue for the specified guild
func (mq *MessageQueueImpl) Size(guildID string) int {
	if guildID == "" {
		return 0
	}

	mq.mu.RLock()
	defer mq.mu.RUnlock()

	queue, exists := mq.queues[guildID]
	if !exists {
		return 0
	}

	return len(queue.messages)
}

// SetMaxSize sets the maximum queue size for the specified guild
func (mq *MessageQueueImpl) SetMaxSize(guildID string, size int) error {
	if guildID == "" {
		return errors.New("guild ID cannot be empty")
	}

	if size <= 0 {
		return errors.New("max size must be greater than 0")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	// Get or create guild queue
	queue, exists := mq.queues[guildID]
	if !exists {
		queue = &guildQueue{
			messages:     make([]*QueuedMessage, 0),
			maxSize:      size,
			lastActivity: time.Now(),
		}
		mq.queues[guildID] = queue
		return nil
	}

	// Update max size
	queue.maxSize = size

	// If current queue is larger than new max size, trim it
	if len(queue.messages) > size {
		// Keep the most recent messages
		startIndex := len(queue.messages) - size
		queue.messages = queue.messages[startIndex:]

		// Log or handle the skip indication
		// Queue size reduction logging removed per user request
	}

	return nil
}

// GetLastActivity returns the last activity time for the specified guild
func (mq *MessageQueueImpl) GetLastActivity(guildID string) time.Time {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	queue, exists := mq.queues[guildID]
	if !exists {
		return time.Time{}
	}

	return queue.lastActivity
}

// CheckInactivity checks if a guild has been inactive for the specified duration
// This supports Requirement 4.4 - announce inactivity after 5 minutes
func (mq *MessageQueueImpl) CheckInactivity(guildID string, timeout time.Duration) bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	queue, exists := mq.queues[guildID]
	if !exists {
		return false
	}

	return time.Since(queue.lastActivity) > timeout
}

// SetInactivityCallback sets a callback function to handle inactivity for a guild
func (mq *MessageQueueImpl) SetInactivityCallback(guildID string, callback func(string)) error {
	if guildID == "" {
		return errors.New("guild ID cannot be empty")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	// Get or create guild queue
	queue, exists := mq.queues[guildID]
	if !exists {
		queue = &guildQueue{
			messages:     make([]*QueuedMessage, 0),
			maxSize:      DefaultMaxQueueSize,
			lastActivity: time.Now(),
		}
		mq.queues[guildID] = queue
	}

	queue.inactivityFunc = callback
	return nil
}

// RemoveGuild removes all data for a specific guild
func (mq *MessageQueueImpl) RemoveGuild(guildID string) error {
	if guildID == "" {
		return errors.New("guild ID cannot be empty")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	delete(mq.queues, guildID)
	return nil
}

// SkipNext removes and returns the next message from the queue without processing it
func (mq *MessageQueueImpl) SkipNext(guildID string) (*QueuedMessage, error) {
	if guildID == "" {
		return nil, errors.New("guild ID cannot be empty")
	}

	mq.mu.Lock()
	defer mq.mu.Unlock()

	queue, exists := mq.queues[guildID]
	if !exists || len(queue.messages) == 0 {
		return nil, nil // No messages in queue to skip
	}

	// Get first message (the one being skipped)
	skippedMessage := queue.messages[0]

	// Remove from queue
	queue.messages = queue.messages[1:]

	// Update last activity time
	queue.lastActivity = time.Now()

	return skippedMessage, nil
}

// GetAllGuilds returns a list of all guild IDs that have queues
func (mq *MessageQueueImpl) GetAllGuilds() []string {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	guilds := make([]string, 0, len(mq.queues))
	for guildID := range mq.queues {
		guilds = append(guilds, guildID)
	}

	return guilds
}
