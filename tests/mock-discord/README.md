# Mock Discord API Server

A lightweight mock implementation of Discord's REST API and Gateway for testing the darrot Discord TTS bot.

## Features

- **REST API Simulation**: Essential Discord API endpoints for bot testing
- **WebSocket Gateway**: Real-time event simulation with proper Discord Gateway protocol
- **Voice Channel Simulation**: Voice connection handling and audio stream capture
- **Audio Processing**: Opus audio packet capture and analysis for TTS validation
- **Containerized Deployment**: Docker support with health checks and orchestration

## Quick Start

### Local Development

```bash
# Install dependencies
make deps

# Run locally
make run

# Or build and run binary
make build
./mock-discord
```

### Docker Deployment

```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# Check status
make status

# View logs
make docker-logs

# Stop
make docker-stop
```

### Testing Environment

```bash
# Start test environment
make test-env-up

# Run acceptance tests
make test-acceptance

# Cleanup
make test-env-down
```

## API Endpoints

### REST API (Port 8080)

- `GET /health` - Health check endpoint
- `GET /api/v10/guilds/{guildId}` - Get guild information
- `GET /api/v10/guilds/{guildId}/channels` - Get guild channels
- `GET /api/v10/channels/{channelId}` - Get channel information
- `POST /api/v10/channels/{channelId}/messages` - Send message
- `PATCH /api/v10/channels/{channelId}/voice-states/@me` - Update voice state
- `GET /api/v10/users/@me` - Get current user (bot)

### WebSocket Gateway (Port 8080)

- `ws://localhost:8080/gateway` - Discord Gateway WebSocket connection

### Voice Server (Port 8081)

- TCP server for voice connections and audio stream capture
- `GET /voice/connections` - Get active voice connections

## Configuration

Environment variables:

- `LOG_LEVEL` - Logging level (DEBUG, INFO, WARN, ERROR)
- `MOCK_GUILD_ID` - Test guild ID (default: test-guild-123)
- `MOCK_USER_ID` - Test user ID (default: test-user-456)
- `MOCK_BOT_ID` - Bot user ID (default: bot-user-789)

## Test Data

The server creates default test data:

- **Guild**: `test-guild-123` (Test Guild)
- **Text Channel**: `test-text-channel-456` (general)
- **Voice Channel**: `test-voice-channel-789` (General Voice)
- **Test User**: `test-user-456` (testuser#1234)
- **Bot User**: `bot-user-789` (darrot#0000)

## Authentication

The server simulates Discord's bot authentication:

```
Authorization: Bot YOUR_BOT_TOKEN
```

Any non-empty token after "Bot " prefix will be accepted for testing.

## Voice Connection Simulation

### Joining Voice Channels

1. Send `PATCH /api/v10/channels/{channelId}/voice-states/@me`
2. Gateway sends `VOICE_STATE_UPDATE` event
3. Gateway sends `VOICE_SERVER_UPDATE` event with connection details
4. Bot connects to voice server on port 8081

### Audio Stream Capture

The voice server captures audio packets and provides analysis:

- Opus format validation
- Packet sequence analysis
- Audio quality metrics
- TTS content verification

## Docker Compose Services

### Development (`docker-compose.yml`)

- `mock-discord` - Main mock server
- `darrot-test` - Test bot instance (profile: testing)

### Testing (`docker-compose.test.yml`)

- `mock-discord` - Mock server for testing
- `acceptance-tests` - Automated test runner

## Health Checks

The server includes comprehensive health monitoring:

```bash
# Check server health
curl http://localhost:8080/health

# Response
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "guilds": 1,
  "users": 2,
  "channels": 2
}
```

## Development Workflow

```bash
# Full development setup
make dev

# Run tests
make test

# CI pipeline
make ci-test

# Cleanup
make clean
make clean-docker
```

## Integration with darrot

Configure darrot to use the mock server:

```yaml
# darrot-config.yaml
discord:
  api_url: "http://localhost:8080/api/v10"
  gateway_url: "ws://localhost:8080/gateway"
  voice_url: "localhost:8081"
  token: "test-bot-token-123"
```

## Troubleshooting

### Connection Issues

```bash
# Check if server is running
make health

# View logs
make docker-logs

# Restart services
make restart
```

### Port Conflicts

The test environment uses different ports:
- REST API: 18080 (instead of 8080)
- Voice: 18081 (instead of 8081)

### Audio Capture Issues

Check voice server logs for audio packet processing:

```bash
docker-compose logs -f mock-discord | grep -i voice
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST API      │    │   Gateway WS    │    │   Voice Server  │
│   Port 8080     │    │   Port 8080     │    │   Port 8081     │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • Guild mgmt    │    │ • Authentication│    │ • Voice conn    │
│ • Channel ops   │    │ • Event dispatch│    │ • Audio capture │
│ • Message API   │    │ • Heartbeat     │    │ • Format valid  │
│ • Voice states  │    │ • Reconnection  │    │ • Quality check │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Contributing

1. Make changes to the Go source files
2. Run tests: `make test`
3. Build and test locally: `make dev`
4. Test with Docker: `make ci-test`
5. Update documentation as needed

## License

MIT License - same as the main darrot project.