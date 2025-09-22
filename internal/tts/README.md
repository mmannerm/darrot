# TTS Package - Message Monitoring and Processing

This package implements the message monitoring and processing functionality for the Discord TTS Voice feature as specified in task 13.

## Components Implemented

### 1. MessageMonitor (`message_monitor.go`)
The core component that handles monitoring Discord text channels for TTS processing.

**Key Features:**
- Monitors Discord message events for paired text channels
- Filters messages from opted-in users only
- Preprocesses messages (adds author name, handles emojis, truncates long messages)
- Integrates with MessageQueue for TTS processing pipeline
- Ignores bot messages and empty messages

**Message Preprocessing:**
- Adds "Username says: " prefix to messages
- Converts Discord custom emojis (`<:name:id>`) to readable text ("name emoji")
- Handles animated emojis (`<a:name:id>`)
- Truncates messages longer than 500 characters
- Cleans up extra whitespace

### 2. MessageQueue (`message_queue.go`)
In-memory implementation of message queuing for TTS processing.

**Key Features:**
- FIFO queue per guild
- Configurable maximum queue size (default: 10)
- Automatic overflow handling (removes oldest messages)
- Thread-safe operations with mutex protection
- Support for multiple guilds simultaneously

### 3. ChannelService (`channel_service.go`)
Manages voice-text channel pairings and monitoring.

**Key Features:**
- Create and remove voice-text channel pairings
- Validate channel access permissions
- Check if text channels are paired with voice channels
- Prevent duplicate text channel pairings
- Thread-safe operations

### 4. UserService (`user_service.go`)
Manages user opt-in preferences for TTS functionality.

**Key Features:**
- Set and check user opt-in status per guild
- Auto opt-in for bot inviters
- Get lists of opted-in users per guild
- Clear guild data when needed
- Thread-safe operations

### 5. Interfaces (`interfaces.go`)
Defines the contracts for all TTS services and data structures.

**Key Interfaces:**
- `ChannelService`: Channel pairing management
- `UserService`: User preference management  
- `MessageQueue`: Message queuing operations

**Key Data Structures:**
- `ChannelPairing`: Represents voice-text channel relationships
- `QueuedMessage`: Represents messages queued for TTS processing

## Requirements Addressed

This implementation addresses the following requirements from the specification:

- **Requirement 2.1**: Automatic monitoring of paired text channels for new messages from opted-in users
- **Requirement 2.2**: Text-to-speech conversion with author name inclusion
- **Requirement 2.4**: Filtering messages based on user opt-in status
- **Requirement 2.5**: Handling of emoji and special character content

## Testing

Comprehensive unit tests are provided for all components:

- `message_monitor_test.go`: Tests message filtering, preprocessing, and emoji handling
- `message_queue_test.go`: Tests queue operations, size limits, and concurrent access
- `channel_service_test.go`: Tests channel pairing management and validation
- `user_service_test.go`: Tests user opt-in management and guild isolation

All tests include:
- Happy path scenarios
- Error handling
- Edge cases
- Concurrent access patterns
- Input validation

## Usage Example

```go
// Create services
logger := log.New(os.Stdout, "[TTS] ", log.LstdFlags)
channelService := NewInMemoryChannelService(logger)
userService := NewInMemoryUserService(logger)
messageQueue := NewInMemoryMessageQueue(logger)

// Create message monitor
monitor := NewMessageMonitor(session, channelService, userService, messageQueue, logger)

// The monitor automatically handles Discord message events
// Messages from opted-in users in paired channels will be queued for TTS processing
```

## Integration Points

This package integrates with:
- Discord API via `github.com/bwmarrin/discordgo`
- Existing bot command system
- Future TTS processing pipeline
- Voice connection management

## Thread Safety

All components are designed to be thread-safe and can handle concurrent operations from multiple Discord events and command handlers.