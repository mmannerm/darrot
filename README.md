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

### Configuration

darrot supports multiple configuration methods with the following precedence order:
1. **CLI flags** (highest priority)
2. **Environment variables** (with `DRT_` prefix)
3. **Configuration files** (YAML, JSON, or TOML)
4. **Default values** (lowest priority)

#### Quick Setup with Environment Variables

Set environment variables directly or use a configuration file:

```bash
# Set environment variables with DRT_ prefix
export DRT_DISCORD_TOKEN="your_actual_bot_token_here"
export DRT_LOG_LEVEL="INFO"
export DRT_TTS_DEFAULT_VOICE="en-US-Standard-A"
export DRT_TTS_DEFAULT_SPEED="1.0"

# Start the bot
./darrot start
```

#### Configuration File Setup

Create a configuration file in YAML, JSON, or TOML format:

```bash
# Generate a sample configuration file
./darrot config create --output darrot-config.yaml

# Or create manually
cat > darrot-config.yaml << EOF
discord_token: "your_bot_token_here"
log_level: "INFO"
tts:
  default_voice: "en-US-Standard-A"
  default_speed: 1.0
  default_volume: 1.0
  max_queue_size: 10
  max_message_length: 500
EOF
```

#### Configuration Management Commands

```bash
# Validate your configuration
./darrot config validate

# View effective configuration
./darrot config show

# View configuration in JSON format
./darrot config show --format json

# Create configuration file from current settings
./darrot config create --output my-config.yaml
```

## Deployment Options

### Option 1: Container Deployment (Recommended)

The easiest way to run darrot is using containers with Podman or Docker:

```bash
# Quick start with Podman
# Create configuration file or set environment variables
echo 'discord_token: "your_bot_token_here"' > darrot-config.yaml
podman build --pull -t darrot:latest .
podman run -d --name darrot-bot -v ./darrot-config.yaml:/app/darrot-config.yaml:ro -v ./data:/app/data:Z darrot:latest
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

# Run the bot with default configuration
./darrot start

# Run with custom configuration file
./darrot start --config /path/to/config.yaml

# Run with CLI flags
./darrot start --discord-token "your_token" --log-level DEBUG
```

## Usage

### CLI Commands

darrot provides a modern CLI interface with the following commands:

#### Main Commands
```bash
# Start the Discord TTS bot
./darrot start

# Display version information
./darrot version

# Show help for all commands
./darrot --help

# Show help for specific command
./darrot start --help
```

#### Configuration Management
```bash
# Validate configuration without starting the bot
./darrot config validate

# Display effective configuration with sources
./darrot config show

# Display configuration in JSON format
./darrot config show --format json

# Create configuration file from current settings
./darrot config create

# Save configuration to specific location
./darrot config create --output /path/to/config.yaml
```

#### Shell Completion
```bash
# Generate bash completion
./darrot completion bash > /etc/bash_completion.d/darrot

# Generate zsh completion
./darrot completion zsh > ~/.zsh/completions/_darrot


```

#### CLI Flags for Start Command
```bash
# Core configuration
./darrot start --discord-token "your_token"
./darrot start --config /path/to/config.yaml
./darrot start --log-level DEBUG

# TTS configuration
./darrot start --tts-default-voice "en-US-Neural2-A"
./darrot start --tts-default-speed 1.2
./darrot start --tts-default-volume 0.8
./darrot start --tts-max-queue-size 15
./darrot start --tts-max-message-length 600

# Google Cloud TTS (optional) - use standard Google Cloud authentication
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
./darrot start
```

### Discord Bot Commands

Once the bot is running and invited to your server:

- `/test` - Verify bot connectivity with "Hello World" response
- `/tts-join` - Join a voice channel and start TTS monitoring
- `/tts-leave` - Leave the voice channel and stop TTS
- `/tts-config` - Configure TTS settings (voice, speed, volume)
- `/tts-opt-in` - Enable TTS reading for your messages
- `/tts-opt-out` - Disable TTS reading for your messages

### Getting Started

1. **Configure the bot**
   ```bash
   # Option 1: Use environment variables
   export DRT_DISCORD_TOKEN="your_bot_token"
   ./darrot start
   
   # Option 2: Use configuration file
   ./darrot config create --output darrot-config.yaml
   # Edit darrot-config.yaml with your settings
   ./darrot start --config darrot-config.yaml
   
   # Option 3: Use CLI flags
   ./darrot start --discord-token "your_token" --log-level INFO
   ```

2. Invite the bot to your Discord server with appropriate permissions
3. Join a voice channel
4. Use `/tts-join` to start the TTS service
5. Type messages in the linked text channel to hear them spoken
6. Use `/tts-config` to customize voice settings

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


```

For detailed testing information, see [docs/testing.md](docs/testing.md).

#### Container Acceptance Tests
```bash
# Test container build and functionality
./test-container.sh



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
- Verify the bot token is correct in your configuration
- Use `./darrot config show` to check your effective configuration
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

# Verify environment variables (note DRT_ prefix)
podman exec darrot-bot env | grep DRT_

# Test configuration validation
podman run --rm -v ./darrot-config.yaml:/app/darrot-config.yaml:ro darrot:latest config validate

# Test with debug logging
podman run -d --name darrot-debug -v ./darrot-config.yaml:/app/darrot-config.yaml:ro -e DRT_LOG_LEVEL=DEBUG darrot:latest
```

**Permission errors with container volumes:**
```bash
# Fix data directory ownership
sudo chown -R 1001:1001 ./data
```

### Getting Help

- Use `./darrot config validate` to check configuration issues
- Use `./darrot config show` to see effective configuration and sources
- Check the [Container Documentation](CONTAINER.md) for deployment issues
- Review logs with `DRT_LOG_LEVEL=DEBUG` for detailed troubleshooting
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

## Migration Guide

### Environment Variable Changes

**Important**: Environment variables now require the `DRT_` prefix. Update your existing configuration:

#### Before (Old Format)
```bash
DISCORD_TOKEN=your_token
LOG_LEVEL=INFO
TTS_DEFAULT_VOICE=en-US-Standard-A
```

#### After (New Format)
```bash
DRT_DISCORD_TOKEN=your_token
DRT_LOG_LEVEL=INFO
DRT_TTS_DEFAULT_VOICE=en-US-Standard-A
```

#### Migration from Environment Variables

If you were using environment variables, you can continue using them with the DRT_ prefix, or migrate to configuration files:

```bash
# Option 1: Continue using environment variables with DRT_ prefix
export DRT_DISCORD_TOKEN="$DISCORD_TOKEN"
export DRT_LOG_LEVEL="$LOG_LEVEL"

# Option 2: Create a configuration file
./darrot config create --output darrot-config.yaml
```

### Command Changes

#### Before (Old Format)
```bash
./darrot  # Direct execution
```

#### After (New Format)
```bash
./darrot start  # Use start subcommand
```

## Configuration Reference

### Environment Variables (DRT_ Prefix Required)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DRT_DISCORD_TOKEN` | Yes | - | Discord bot token from Developer Portal |
| `DRT_LOG_LEVEL` | No | INFO | Logging level (DEBUG, INFO, WARN, ERROR) |
| `DRT_TTS_DEFAULT_VOICE` | No | en-US-Standard-A | Default TTS voice selection |
| `DRT_TTS_DEFAULT_SPEED` | No | 1.0 | Speech speed (0.25-4.0) |
| `DRT_TTS_DEFAULT_VOLUME` | No | 1.0 | Speech volume (0.0-2.0) |
| `DRT_TTS_MAX_QUEUE_SIZE` | No | 10 | Maximum messages in queue (1-100) |
| `DRT_TTS_MAX_MESSAGE_LENGTH` | No | 500 | Maximum message length for TTS (1-2000) |

### Configuration File Options

Configuration files support nested structure for better organization:

```yaml
# darrot-config.yaml example
discord_token: "your_bot_token_here"
log_level: "INFO"

tts:
  default_voice: "en-US-Standard-A"
  default_speed: 1.0
  default_volume: 1.0
  max_queue_size: 10
  max_message_length: 500

cli:
  enable_colors: true
  completion_shell: "bash"
```

### CLI Flag Reference

All configuration options are available as CLI flags:

```bash
# Core flags
--discord-token string              Discord bot token
--config string                     Configuration file path
--log-level string                  Log level (DEBUG, INFO, WARN, ERROR)

# TTS flags
--tts-default-voice string               Default TTS voice
--tts-default-speed float                Speech speed (0.25-4.0)
--tts-default-volume float               Speech volume (0.0-2.0)
--tts-max-queue-size int                 Maximum queue size (1-100)
--tts-max-message-length int             Maximum message length (1-2000)
```

### Google Cloud TTS Setup (Optional)

For enhanced neural voices and better TTS quality, configure Google Cloud Text-to-Speech:

1. **Create a Google Cloud project and enable the Text-to-Speech API**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select an existing one
   - Enable the [Text-to-Speech API](https://console.cloud.google.com/apis/library/texttospeech.googleapis.com)

2. **Set up authentication using one of these methods:**

   **Option A: Service Account (Recommended for production)**
   ```bash
   # Create service account and download JSON key
   # Set the environment variable
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/service-account-key.json
   ./darrot start
   ```

   **Option B: Application Default Credentials (For development)**
   ```bash
   # Install Google Cloud CLI and authenticate
   gcloud auth application-default login
   ./darrot start
   ```

3. **The bot will automatically use Google Cloud TTS when credentials are available**
   - Enhanced neural voices (e.g., en-US-Neural2-A, en-US-Neural2-C)
   - Better audio quality and more natural speech
   - Supports multiple languages and voice styles

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