# Message Monitor Integration

This document describes the MessageMonitor component that was integrated into the existing TTS system to provide automatic message monitoring and processing functionality.

## Overview

The MessageMonitor component automatically monitors Discord text channels for new messages and processes them for TTS conversion. It integrates with the existing TTS system interfaces and provides the missing piece for automatic message detection and queuing.

## Key Features

### 🎯 **Automatic Message Detection**
- Monitors all Discord message events in real-time
- Filters messages based on channel pairing status
- Respects user opt-in preferences
- Ignores bot messages and empty content

### 🔧 **Message Processing**
- **Author Name Inclusion**: Adds "Username says: " prefix to messages
- **Emoji Handling**: Converts Discord custom emojis to readable text
- **Content Sanitization**: Cleans up whitespace and handles special characters
- **Length Limiting**: Truncates messages longer than 500 characters

### 🚀 **Integration Points**
- Uses existing `ChannelService` interface for pairing validation
- Uses existing `UserService` interface for opt-in checking
- Uses existing `MessageQueue` interface for message queuing
- Uses existing `QueuedMessage` type for data consistency

## Implementation Details

### MessageMonitor Structure
```go
type MessageMonitor struct {
    session        *discordgo.Session
    channelService ChannelService
    userService    UserService
    messageQueue   MessageQueue
    logger         *log.Logger
    emojiRegex     *regexp.Regexp
}
```

### Message Processing Flow
1. **Event Reception**: Discord message create event received
2. **Bot Filter**: Skip if message is from a bot
3. **Content Filter**: Skip if message is empty/whitespace only
4. **Channel Check**: Verify text channel is paired with voice channel
5. **User Check**: Verify user is opted-in for TTS
6. **Preprocessing**: Apply author name, emoji handling, and truncation
7. **Queuing**: Add processed message to guild's message queue

### Emoji Processing
- Custom Discord emojis (`<:name:id>`) → "name emoji"
- Animated emojis (`<a:name:id>`) → "name emoji"
- Excessive emoji sequences (4+) → "multiple emojis"
- Unicode emojis are preserved for TTS engine handling

## Integration with Existing System

The MessageMonitor seamlessly integrates with the existing TTS system:

### ✅ **Compatible Interfaces**
- `ChannelService.IsChannelPaired(guildID, textChannelID string) bool`
- `UserService.IsOptedIn(userID, guildID string) (bool, error)`
- `MessageQueue.Enqueue(message *QueuedMessage) error`

### ✅ **Existing Data Types**
- Uses existing `QueuedMessage` struct from `types.go`
- Compatible with existing `ChannelPairing` structure
- Follows established error handling patterns

### ✅ **No Breaking Changes**
- Additive functionality only
- No modifications to existing interfaces
- No changes to existing implementations

## Usage

### Initialization
```go
// Create MessageMonitor with existing services
monitor := NewMessageMonitor(
    discordSession,
    channelService,
    userService,
    messageQueue,
    logger,
)

// Monitor automatically registers Discord event handlers
// No additional setup required
```

### Lifecycle Management
```go
// Check if monitoring is active
if monitor.IsMonitoring() {
    log.Println("Message monitoring is active")
}

// Stop monitoring (cleanup)
monitor.Stop()
```

## Testing

Comprehensive test suite included:

### 📊 **Test Coverage**
- **Message Filtering**: Bot messages, empty messages, unpaired channels
- **User Filtering**: Opted-in vs opted-out users
- **Message Processing**: Author names, emoji handling, truncation
- **Integration**: Mock services for isolated testing

### 🧪 **Test Structure**
- `TestMessageMonitor_handleMessageCreate`: Core message processing logic
- `TestMessageMonitor_preprocessMessage`: Message preprocessing functionality
- `TestMessageMonitor_handleEmojis`: Emoji conversion logic
- `TestMessageMonitor_IsMonitoring`: Lifecycle management

## Requirements Fulfilled

This implementation addresses the following requirements:

- **2.1**: Automatic monitoring of paired text channels ✓
- **2.2**: Text-to-speech conversion with author name inclusion ✓
- **2.4**: Message filtering based on user opt-in status ✓
- **2.5**: Proper handling of emojis and special characters ✓

## Files Added

- `internal/tts/message_monitor.go` - Core MessageMonitor implementation
- `internal/tts/message_monitor_test.go` - Comprehensive test suite
- `internal/tts/message_monitor_README.md` - This documentation

## Performance Considerations

- **Minimal Overhead**: Only processes messages from paired channels
- **Early Filtering**: Multiple filter stages prevent unnecessary processing
- **Efficient Regex**: Pre-compiled emoji patterns for fast processing
- **Memory Efficient**: No message caching, immediate queue handoff

## Future Enhancements

Potential improvements for future iterations:

1. **Configurable Preprocessing**: Allow guilds to customize message formatting
2. **Advanced Emoji Handling**: More sophisticated emoji-to-text conversion
3. **Rate Limiting**: Per-user or per-guild message rate limiting
4. **Message Priorities**: Priority queuing for different message types
5. **Content Filtering**: Configurable content filters and word replacement

## Integration Status

✅ **Completed**: Message monitoring functionality integrated with existing TTS system  
✅ **Tested**: Full test coverage with passing integration tests  
✅ **Compatible**: Works with existing interfaces without breaking changes  
✅ **Ready**: Available for immediate use in production TTS system