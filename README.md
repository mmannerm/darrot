# darrot

Discord Parrot Text-to-Speech (TTS) AI app that listens to Discord chat channels and converts text messages to speech using Google Cloud TTS.

## Features

- 🎤 **Real-time TTS**: Converts Discord messages to speech in voice channels
- 🔒 **Privacy Controls**: Opt-in system for user message reading
- 🎛️ **Configurable**: Adjustable voice, speed, volume, and queue settings
- 👑 **Role-based Permissions**: Administrative controls for server management
- 🔄 **Error Recovery**: Automatic reconnection and retry mechanisms
- 🐳 **Container Ready**: Production-ready Docker/Podman deployment
- 🧪 **Comprehensive Testing**: Full test suite with 100% core coverage

## Setup

### Prerequisites
- Go 1.19 or later
- Discord Bot Token

### Discord Bot Setup

1. **Create a Discord Application**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Click "New Application" and give it a name
   - Navigate to the "Bot" section in the left sidebar

2. **Get Your Bot Token**
   - In the Bot section, click "Reset Token" to generate a new token
   - Copy the token (you'll need this for the next step)
   - **Important**: Keep this token secret and never commit it to version control

3. **Configure Bot Permissions**
   - In the Bot section, enable the following permissions:
     - `applications.commands` (for slash commands)
     - `bot` (basic bot functionality)
   - Generate an invite link from the OAuth2 > URL Generator section
   - Invite the bot to your Discord server

### Environment Configuration

1. **Copy the environment template**
   ```bash
   cp .env.example .env
   ```

2. **Edit the .env file**
   - Replace `your_discord_bot_token_here` with your actual Discord bot token
   - Optionally adjust the LOG_LEVEL (DEBUG, INFO, WARN, ERROR)

3. **Example .env file**
   ```
   DISCORD_TOKEN=your_actual_bot_token_here
   LOG_LEVEL=INFO
   ```

## Deployment Options

### Option 1: Container Deployment (Recommended)

The easiest way to run darrot is using containers with Podman or Docker:

```bash
# Quick start with Podman
cp container-env.example .env
# Edit .env with your Discord token
podman build --pull -t darrot:latest .
podman run -d --name darrot-bot --env-file .env -v ./data:/app/data:Z darrot:latest
```

**Container Features:**
- 🔒 Security hardened (non-root user, read-only filesystem)
- 📦 Minimal Alpine-based image (~50MB)
- 🚀 Multi-architecture support (AMD64, ARM64)
- 📊 Resource limits and health checks
- 🔧 Easy configuration via environment variables

For detailed container setup, configuration, and testing instructions, see [CONTAINER.md](CONTAINER.md).

### Option 2: Local Development

```bash
# Install dependencies
go mod tidy

# Build the application
go build -o darrot ./cmd/darrot

# Run the bot
./darrot
```

## Usage

### Bot Commands

Once the bot is running and invited to your server:

- `/test` - Verify bot connectivity with "Hello World" response
- `/tts-join` - Join a voice channel and start TTS monitoring
- `/tts-leave` - Leave the voice channel and stop TTS
- `/tts-config` - Configure TTS settings (voice, speed, volume)
- `/tts-opt-in` - Enable TTS reading for your messages
- `/tts-opt-out` - Disable TTS reading for your messages

### Getting Started

1. Invite the bot to your Discord server with appropriate permissions
2. Join a voice channel
3. Use `/tts-join` to start the TTS service
4. Type messages in the linked text channel to hear them spoken
5. Use `/tts-config` to customize voice settings

## Development

### Running Tests

#### Unit Tests Only
```bash
go test ./... -short
```

#### All Tests (Including Integration)
```bash
# Set test bot token for integration tests
export DISCORD_TEST_TOKEN="your_test_bot_token"
go test ./...
```

#### Using Test Scripts
```bash
# Linux/macOS
./scripts/run-integration-tests.sh

# Windows
scripts\run-integration-tests.bat
```

For detailed testing information, see [docs/testing.md](docs/testing.md).

#### Container Acceptance Tests
```bash
# Test container build and functionality
./test-container.sh

# Windows
test-container.bat

# Manual Podman test
bash tests/container/acceptance_test.sh
```

### Code Formatting
```bash
go fmt ./...
go vet ./...
```

### Building from Source
```bash
# Clone the repository
git clone <repository-url>
cd darrot

# Install dependencies
go mod tidy

# Build with version information
go build -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o darrot ./cmd/darrot

# Run locally
./darrot
```

## Troubleshooting

### Common Issues

**Bot doesn't respond to commands:**
- Verify the bot token is correct in your `.env` file
- Ensure the bot has `applications.commands` and `bot` permissions
- Check that the bot is online in your Discord server

**Audio issues or no TTS output:**
- Verify Google Cloud TTS credentials are properly configured
- Check that the bot has permission to join voice channels
- Ensure Opus audio libraries are installed (for local builds)

**Container won't start:**
```bash
# Check container logs
podman logs darrot-bot

# Verify environment variables
podman exec darrot-bot env | grep DISCORD

# Test with debug logging
podman run -d --name darrot-debug --env-file .env -e LOG_LEVEL=DEBUG darrot:latest
```

**Permission errors with container volumes:**
```bash
# Fix data directory ownership
sudo chown -R 1001:1001 ./data
```

### Getting Help

- Check the [Container Documentation](CONTAINER.md) for deployment issues
- Review logs with `LOG_LEVEL=DEBUG` for detailed troubleshooting
- Verify Discord bot permissions and token validity
- Ensure Google Cloud TTS API is enabled and credentials are valid

## Architecture

The application follows a modular design:

- **cmd/darrot**: Main application entry point
- **internal/bot**: Discord bot core functionality and command routing
- **internal/tts**: Text-to-Speech processing, voice management, and message monitoring
- **internal/config**: Configuration management and validation

Key components:
- **Message Monitor**: Real-time Discord message processing
- **Voice Manager**: Discord voice connection handling
- **TTS Manager**: Google Cloud TTS integration
- **Error Recovery**: Comprehensive error handling and retry logic

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | Yes | - | Discord bot token from Developer Portal |
| `LOG_LEVEL` | No | INFO | Logging level (DEBUG, INFO, WARN, ERROR) |
| `GOOGLE_CLOUD_CREDENTIALS_PATH` | No | - | Path to Google Cloud service account JSON |
| `TTS_DEFAULT_VOICE` | No | en-US-Standard-A | Default TTS voice selection |
| `TTS_DEFAULT_SPEED` | No | 1.0 | Speech speed (0.25-4.0) |
| `TTS_DEFAULT_VOLUME` | No | 1.0 | Speech volume (0.0-2.0) |
| `TTS_MAX_QUEUE_SIZE` | No | 10 | Maximum messages in queue (1-100) |
| `TTS_MAX_MESSAGE_LENGTH` | No | 500 | Maximum message length for TTS (1-2000) |

### Google Cloud TTS Setup (Optional)

1. Create a Google Cloud project and enable the Text-to-Speech API
2. Create a service account and download the JSON credentials
3. Set `GOOGLE_CLOUD_CREDENTIALS_PATH` to the credentials file path
4. The bot will use enhanced neural voices when configured

## Performance

- **Memory Usage**: ~50-100MB typical, 256MB container limit
- **CPU Usage**: Low, optimized for concurrent message processing
- **Audio Quality**: Native Opus encoding for optimal Discord compatibility
- **Latency**: <500ms typical TTS processing time
- **Scalability**: Supports multiple Discord servers simultaneously

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following the coding standards
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Commit using [Conventional Commits](https://www.conventionalcommits.org/) format
7. Push to your branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Maintain test coverage above 90%
- Use conventional commit messages for automated releases
- Document all public APIs and configuration options
- Test container builds before submitting PRs

## Acknowledgments

- [Discord Go](https://github.com/bwmarrin/discordgo) - Discord API library
- [Google Cloud TTS](https://cloud.google.com/text-to-speech) - Text-to-Speech service
- [Opus](https://opus-codec.org/) - Audio codec for Discord compatibility