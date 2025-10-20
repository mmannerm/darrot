# Technology Stack

## Development Principle
**Always aim for the simplest possible end result when making technical decisions.** Choose the most straightforward solution that meets requirements without unnecessary complexity.

## Primary Language
- **Go** - Main programming language for the Discord TTS bot

## Key Dependencies (Implemented)
- **Discord API**: `github.com/bwmarrin/discordgo` - Discord bot integration
- **Text-to-Speech**: Google Cloud Text-to-Speech API - High-quality neural voices
- **Audio Processing**: 
  - `layeh.com/gopus` - Native Opus encoding for Discord voice
  - `github.com/jonas747/dca` - Discord audio format support
- **Testing**: `github.com/stretchr/testify` - Comprehensive test framework

## Development Environment
- **Go Modules**: Dependency management with `go.mod` and `go.sum`
- **Environment Variables**: Configuration via `.env` file (Discord tokens, Google Cloud credentials)
- **Cross-platform**: Supports Windows, Linux, macOS
- **IDE Integration**: Optimized for development with proper module structure

## CI/CD Pipeline
- **GitHub Actions**: Automated continuous integration and deployment
  - Automated testing on pull requests
  - Multi-platform build verification
  - Automated releases and deployments
  - Code quality checks and linting

## Common Commands
```bash
# Install dependencies
go mod tidy

# Build the application
go build -o darrot ./cmd/darrot

# Run the application
go run ./cmd/darrot

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/tts

# Format code
go fmt ./...

# Lint code
go vet ./...

# Run tests with race detection
go test -race ./...
```

## Project Structure (Implemented)
```
darrot/
├── cmd/darrot/          # Main application entry point
├── internal/            # Private application code
│   ├── bot/            # Discord bot core functionality
│   ├── config/         # Configuration management
│   └── tts/            # TTS system implementation
├── .kiro/              # Kiro AI assistant configuration
├── darrot-config.yaml.example # Example YAML configuration
├── go.mod              # Go module definition
└── go.sum              # Go module checksums
```

## Configuration
- **Environment Variables**: Use `.env` file for sensitive configuration (Discord bot tokens, Google Cloud API keys)
- **JSON Storage**: Guild configurations and user preferences stored in JSON files
- **Security**: Never commit sensitive tokens to version control
- **Documentation**: All configuration options documented in README and example files

## Audio Processing Architecture
- **Native Opus Encoding**: Direct Opus codec integration for optimal Discord compatibility
- **DCA Format Support**: Discord-specific audio format for voice streaming
- **Configurable Quality**: Adjustable voice settings (speed, volume, voice selection)
- **Error Recovery**: Robust audio pipeline with automatic fallback mechanisms