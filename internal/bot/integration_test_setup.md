# Integration Test Setup

This document describes how to set up and run the integration tests for the Discord command flow.

## Prerequisites

1. **Discord Bot Token**: You need a test Discord bot token to run integration tests
2. **Go Environment**: Go 1.19 or later
3. **Network Access**: Tests require internet connectivity to reach Discord API

## Setting Up Test Bot Token

### Option 1: Environment Variable (Recommended)
```bash
export DISCORD_TEST_TOKEN="your_test_bot_token_here"
```

### Option 2: Test Environment File
Create a `.env.test` file in the project root:
```
DISCORD_TEST_TOKEN=your_test_bot_token_here
```

## Creating a Test Discord Bot

1. Go to https://discord.com/developers/applications
2. Click "New Application"
3. Give it a name like "darrot-test-bot"
4. Go to the "Bot" section
5. Click "Add Bot"
6. Copy the token from the "Token" section
7. **Important**: Keep this token secure and never commit it to version control

## Running Integration Tests

### Run All Tests (Including Integration)
```bash
# With environment variable
DISCORD_TEST_TOKEN="your_token" go test ./internal/bot -v

# Or if you have the token exported
go test ./internal/bot -v
```

### Run Only Integration Tests
```bash
DISCORD_TEST_TOKEN="your_token" go test ./internal/bot -v -run "TestIntegration"
```

### Run Only Unit Tests (Skip Integration)
```bash
go test ./internal/bot -v -short
```

## Test Coverage

The integration tests cover:

1. **Complete Command Flow** (`TestIntegration_CompleteCommandFlow`)
   - Bot startup and shutdown
   - Command registration
   - Running state verification

2. **Discord API Integration** (`TestIntegration_CommandRegistrationWithDiscordAPI`)
   - Real Discord API connection
   - Command registration with Discord
   - Session state verification

3. **Command Execution** (`TestIntegration_TestCommandExecution`)
   - Command routing infrastructure
   - Handler execution path validation
   - Command registration verification
   - Note: Uses mock interactions, Discord API errors are expected

4. **Response Behavior** (`TestIntegration_EphemeralResponseBehavior`)
   - Ephemeral response verification
   - Handler interface compliance

5. **Error Handling** (`TestIntegration_ErrorHandling`)
   - Unknown command handling
   - Empty command name handling
   - Error message verification

6. **Bot Lifecycle** (`TestIntegration_BotLifecycle`)
   - Start/stop functionality
   - State management
   - Duplicate operation handling

7. **Performance** (`TestIntegration_CommandResponseTiming`)
   - Response time verification (< 3 seconds requirement)
   - Handler execution timing

8. **Response Content** (`TestIntegration_HelloWorldResponse`)
   - "Hello World" response verification
   - Command definition validation

## Test Requirements Mapping

These tests verify the following requirements:

- **Requirement 1.1**: "/test" command execution and response
- **Requirement 1.2**: Ephemeral response behavior
- **Requirement 1.3**: Command availability when bot is online
- **Requirement 3.1**: Response within 3 seconds
- **Requirement 3.2**: "Hello World" response content
- **Requirement 3.3**: Error handling and user-friendly messages

## Troubleshooting

### Tests Skip with "DISCORD_TEST_TOKEN not set"
- Ensure you have set the `DISCORD_TEST_TOKEN` environment variable
- Verify the token is valid and not expired

### Connection Errors
- Check your internet connection
- Verify the bot token is correct
- Ensure the bot hasn't been deleted from Discord Developer Portal

### Permission Errors
- The test bot needs basic bot permissions
- No special server permissions are required for these tests

### Rate Limiting
- If tests fail due to rate limiting, wait a few minutes and retry
- Consider running tests less frequently during development

### Expected Test Behavior
- `TestIntegration_TestCommandExecution` may show Discord API errors when using mock interactions - this is expected
- The test validates command routing infrastructure, not actual Discord responses
- Real Discord interactions require valid interaction tokens from Discord's gateway

## Security Notes

- Never commit bot tokens to version control
- Use separate test tokens from production tokens
- Revoke test tokens when no longer needed
- Keep tokens secure and don't share them