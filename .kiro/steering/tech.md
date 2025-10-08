# Technology Stack

## Primary Language
- **Go** - Main programming language for the Discord TTS bot

## Key Dependencies (Expected)
- Discord API library (likely discordgo or similar)
- Text-to-Speech engine integration
- Audio processing libraries for voice output

## Development Environment
- Go modules for dependency management
- Environment variables for configuration (Discord tokens, API keys)
- Cross-platform support (Windows, Linux, macOS)

## Common Commands
```bash
# Initialize Go module (when ready)
go mod init darrot

# Install dependencies
go mod tidy

# Build the application
go build -o darrot

# Run the application
go run main.go

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint code
go vet ./...
```

## Configuration
- Use `.env` file for sensitive configuration (Discord bot tokens, API keys)
- Environment variables should be documented in README
- Never commit sensitive tokens to version control