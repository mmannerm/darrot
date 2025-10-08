# Testing Guide

This document describes the testing strategy and how to run tests for the darrot Discord TTS bot.

## Test Structure

The project uses a comprehensive testing approach with multiple test types:

### Unit Tests
- **Location**: `internal/*/test.go` files
- **Purpose**: Test individual components in isolation
- **Coverage**: Configuration, command handlers, bot logic, command routing

### Integration Tests
- **Location**: `internal/bot/integration_test.go`
- **Purpose**: Test complete Discord command flow end-to-end
- **Requirements**: Discord bot token for real API testing

## Running Tests

### Quick Test (Unit Tests Only)
```bash
go test ./... -short
```

### All Tests (Including Integration)
```bash
# Set your Discord test bot token
export DISCORD_TEST_TOKEN="your_test_bot_token_here"

# Run all tests
go test ./...
```

### Integration Tests Only
```bash
export DISCORD_TEST_TOKEN="your_test_bot_token_here"
go test ./internal/bot -v -run "TestIntegration"
```

### Using Test Scripts
```bash
# Linux/macOS
./scripts/run-integration-tests.sh

# Windows
scripts\run-integration-tests.bat
```

## Integration Test Coverage

The integration tests verify all requirements for the Discord test command:

### Requirement Coverage

| Requirement | Test Function | Description |
|-------------|---------------|-------------|
| 1.1 | `TestIntegration_CompleteCommandFlow` | "/test" command execution and "Hello World" response |
| 1.2 | `TestIntegration_EphemeralResponseBehavior` | Ephemeral response (visible only to command user) |
| 1.3 | `TestIntegration_BotLifecycle` | Command availability when bot is online |
| 2.1, 2.2, 2.3 | `TestIntegration_CommandRegistrationWithDiscordAPI` | Slash command registration with Discord |
| 3.1 | `TestIntegration_CommandResponseTiming` | Response within 3 seconds |
| 3.2 | `TestIntegration_HelloWorldResponse` | "Hello World" response content |
| 3.3 | `TestIntegration_ErrorHandling` | User-friendly error messages |
| 4.1, 4.2, 4.3 | `TestIntegration_ErrorHandling` | Error logging and handling |

### Test Functions

1. **TestIntegration_CompleteCommandFlow**
   - Tests bot startup and shutdown
   - Verifies command registration
   - Checks running state management

2. **TestIntegration_CommandRegistrationWithDiscordAPI**
   - Tests real Discord API connection
   - Verifies command registration with Discord
   - Validates session state initialization

3. **TestIntegration_TestCommandExecution**
   - Tests command routing
   - Verifies handler execution
   - Checks error handling

4. **TestIntegration_EphemeralResponseBehavior**
   - Verifies ephemeral response flag usage
   - Tests handler interface compliance

5. **TestIntegration_ErrorHandling**
   - Tests unknown command handling
   - Verifies error message content
   - Checks empty command name handling

6. **TestIntegration_BotLifecycle**
   - Tests complete start/stop cycle
   - Verifies state management
   - Tests duplicate operation handling

7. **TestIntegration_CommandResponseTiming**
   - Verifies response time requirements
   - Tests handler performance

8. **TestIntegration_HelloWorldResponse**
   - Verifies "Hello World" response content
   - Tests command definition validation

## Setting Up Test Environment

### Discord Bot Token

1. Go to https://discord.com/developers/applications
2. Create a new application (e.g., "darrot-test-bot")
3. Navigate to the "Bot" section
4. Click "Add Bot" if not already created
5. Copy the token from the "Token" section
6. Set the environment variable:
   ```bash
   export DISCORD_TEST_TOKEN="your_token_here"
   ```

### Security Notes

- **Never commit bot tokens to version control**
- Use separate test tokens from production
- Revoke test tokens when no longer needed
- Keep tokens secure and don't share them

## Test Configuration

### Environment Variables

- `DISCORD_TEST_TOKEN`: Required for integration tests
- Tests will skip if token is not provided

### Test Behavior

- Integration tests are skipped if `DISCORD_TEST_TOKEN` is not set
- Unit tests run independently of integration tests
- Use `-short` flag to run only unit tests

## Continuous Integration

For CI/CD pipelines:

```yaml
# Example GitHub Actions configuration
- name: Run Unit Tests
  run: go test ./... -short

- name: Run Integration Tests
  env:
    DISCORD_TEST_TOKEN: ${{ secrets.DISCORD_TEST_TOKEN }}
  run: go test ./...
```

## Test Coverage

Generate test coverage reports:

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out -o coverage.html
```

## Troubleshooting

### Common Issues

1. **Tests skip with "DISCORD_TEST_TOKEN not set"**
   - Set the environment variable with a valid Discord bot token

2. **Connection errors**
   - Check internet connectivity
   - Verify bot token is valid and not expired

3. **Permission errors**
   - Ensure bot has basic permissions (no special server permissions needed)

4. **Rate limiting**
   - Wait a few minutes between test runs if hitting rate limits

### Debug Mode

Run tests with verbose output:
```bash
go test ./internal/bot -v
```

Add debug logging:
```bash
LOG_LEVEL=DEBUG go test ./internal/bot -v
```