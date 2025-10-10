# Requirements Document

## Introduction

The Discord TTS Voice feature enables the darrot bot to join Discord voice channels and convert text messages from associated or specified text channels into speech, providing an audio experience for Discord server members. This feature allows users to listen to chat conversations while engaged in other activities or when they prefer audio over reading text.

**Implementation Principle:** When making changes to this system, always aim for the simplest possible end result that meets the requirements while maintaining reliability and performance.

## Requirements

### Requirement 1

**User Story:** As a Discord server member, I want to invite the bot to a voice channel with a specified text channel, so that I can hear text messages from that channel converted to speech.

#### Acceptance Criteria

1. WHEN a user sends a voice channel invite command with a specified text channel THEN the bot SHALL join the voice channel and create a pairing with the specified text channel
2. WHEN a user sends a voice channel invite command without specifying a text channel THEN the bot SHALL join the voice channel and pair it with the embedded text chat of that voice channel
3. WHEN the bot joins a voice channel THEN the bot SHALL confirm its presence and indicate which text channel is being monitored
4. IF the bot is already in a voice channel THEN the bot SHALL leave the current channel before joining the new one
5. WHEN the specified text channel is invalid or inaccessible THEN the bot SHALL provide an error message and not join the voice channel
6. WHEN the bot fails to join a voice channel THEN the bot SHALL provide an error message explaining the failure
7. WHEN multiple voice channels exist in a server THEN each voice channel SHALL have only one associated text channel
8. IF a text channel is already paired with another voice channel THEN the bot SHALL provide an error message

### Requirement 2

**User Story:** As a Discord server member, I want the bot to read messages from opted-in users in the paired text channel, so that I can hear relevant conversations without looking at text.

#### Acceptance Criteria

1. WHEN the bot creates a voice-text channel pairing THEN the bot SHALL automatically monitor the paired text channel for new messages from opted-in users
2. WHEN a new text message is posted by an opted-in user in the monitored channel THEN the bot SHALL convert the message to speech
3. WHEN converting text to speech THEN the bot SHALL include the author's name before reading the message content
4. IF a message contains only emojis or special characters THEN the bot SHALL skip the message or provide appropriate audio feedback
5. WHEN a message is posted by a user who has not opted-in THEN the bot SHALL ignore the message

### Requirement 3

**User Story:** As a Discord server member with bot invite permissions, I want to control the bot's voice output, so that I can manage the audio experience.

#### Acceptance Criteria

1. WHEN a user with bot invite permissions sends a stop command THEN the bot SHALL stop reading current messages and leave the voice channel
2. WHEN a user with bot invite permissions sends a pause command THEN the bot SHALL pause message reading but remain in the voice channel
3. WHEN a user with bot invite permissions sends a resume command THEN the bot SHALL resume reading new messages
4. WHEN a user with bot invite permissions sends a skip command THEN the bot SHALL stop the current message and proceed to the next queued message
5. WHEN a user without bot invite permissions attempts to control the bot THEN the bot SHALL deny the request with an appropriate error message

### Requirement 4

**User Story:** As a Discord server member, I want the bot to handle message queuing appropriately, so that messages are read in order without overwhelming audio output.

#### Acceptance Criteria

1. WHEN multiple messages arrive quickly THEN the bot SHALL queue messages and read them sequentially
2. WHEN a message is longer than 30 seconds of speech THEN the bot SHALL truncate the message with an indication
3. WHEN the message queue exceeds 10 messages THEN the bot SHALL skip older messages and indicate the skip
4. IF no new messages arrive for 5 minutes THEN the bot SHALL announce inactivity but remain in the channel

### Requirement 5

**User Story:** As a Discord server administrator, I want to configure TTS settings, so that I can customize the voice experience for my server.

#### Acceptance Criteria

1. WHEN an administrator sets voice speed THEN the bot SHALL use the specified speech rate for all messages
2. WHEN an administrator sets voice type THEN the bot SHALL use the specified voice for speech synthesis
3. WHEN an administrator sets volume level THEN the bot SHALL adjust audio output to the specified level
4. IF invalid settings are provided THEN the bot SHALL use default values and notify the administrator

### Requirement 6

**User Story:** As a Discord server member, I want to opt-in to having my messages read aloud, so that I have control over my privacy and participation in voice output.

#### Acceptance Criteria

1. WHEN a user invites the bot to a voice channel THEN that user SHALL be automatically opted-in for message reading
2. WHEN a user wants to opt-in to message reading THEN the user SHALL use a dedicated opt-in command
3. WHEN a user wants to opt-out of message reading THEN the user SHALL use a dedicated opt-out command
4. WHEN a user checks their opt-in status THEN the bot SHALL provide their current status for the server
5. WHEN an opted-out user posts a message THEN the bot SHALL not read that message aloud
6. IF a user has not explicitly opted-in THEN the bot SHALL not read their messages by default

### Requirement 7

**User Story:** As a Discord server administrator, I want to control which roles can invite the bot to voice channels, so that I can manage who has access to TTS functionality.

#### Acceptance Criteria

1. WHEN an administrator configures required roles for bot invites THEN only users with those roles SHALL be able to invite the bot to voice channels
2. WHEN a user without required roles attempts to invite the bot THEN the bot SHALL deny the request with an appropriate error message
3. WHEN no roles are configured THEN any server member SHALL be able to invite the bot to voice channels
4. WHEN an administrator updates role requirements THEN the changes SHALL take effect immediately for new invite attempts
5. IF an administrator role is required THEN users with administrator permissions SHALL always be able to invite the bot

### Requirement 8

**User Story:** As a Discord server member, I want the bot to verify my permissions before joining channels, so that the bot only operates in channels I have legitimate access to.

#### Acceptance Criteria

1. WHEN a user invites the bot to a voice channel THEN the bot SHALL verify the user has access to that voice channel
2. WHEN a user specifies a text channel THEN the bot SHALL verify the user has read access to that text channel
3. WHEN the user lacks access to the voice channel THEN the bot SHALL deny the request with an appropriate error message
4. WHEN the user lacks access to the text channel THEN the bot SHALL deny the request with an appropriate error message
5. WHEN the bot lacks permissions for the voice or text channels THEN the bot SHALL provide clear error messages about missing permissions

### Requirement 9

**User Story:** As a Discord server member, I want the bot to handle errors gracefully, so that the voice experience remains stable.

#### Acceptance Criteria

1. WHEN the bot loses connection to the voice channel THEN the bot SHALL attempt to reconnect automatically
2. WHEN TTS conversion fails THEN the bot SHALL skip the problematic message and continue with the next
3. WHEN the bot lacks permissions for voice or text channels THEN the bot SHALL provide clear error messages
4. IF the bot encounters repeated errors THEN the bot SHALL leave the voice channel and notify users of the issue

## Implementation Status

✅ **COMPLETED** - All requirements have been implemented and tested:

- **Voice Channel Management**: Bot can join/leave voice channels with proper permission validation
- **Text-to-Speech Processing**: Messages are converted to speech using Google Cloud TTS with native Opus encoding
- **User Opt-in System**: Users can opt-in/out of having their messages read aloud
- **Message Queue Management**: Messages are queued and processed sequentially with overflow handling
- **Administrative Controls**: Server admins can configure TTS settings and role requirements
- **Error Recovery**: Comprehensive error handling with automatic reconnection and retry mechanisms
- **Performance Optimization**: Test suite runs 67% faster with configurable timing for different environments

The system uses a modular architecture with separate components for voice management, TTS processing, message monitoring, user preferences, and error recovery, following the principle of simplest possible implementation.