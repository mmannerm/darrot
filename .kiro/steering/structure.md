# Project Structure

## Organizational Principle
**Always aim for the simplest possible project structure that maintains clarity and maintainability.** Avoid unnecessary nesting and complexity in the directory hierarchy.

## Current Implementation
```
darrot/
├── .git/                    # Git version control
├── .github/                 # GitHub Actions workflows (CI/CD)
├── .kiro/                   # Kiro AI assistant configuration
│   ├── specs/              # Feature specifications
│   └── steering/           # AI guidance documents
├── .vscode/                # VSCode editor settings
├── cmd/darrot/             # Main application entry point
│   └── main.go
├── internal/               # Private application code
│   ├── bot/               # Discord bot core functionality
│   │   ├── bot.go         # Main bot implementation
│   │   ├── commands.go    # Command router
│   │   ├── integration.go # Discord API integration
│   │   └── test_command.go # Test command handler
│   ├── config/            # Configuration management
│   │   └── config.go      # Configuration loading and validation
│   └── tts/               # Text-to-Speech system
│       ├── command_handlers.go      # TTS Discord commands
│       ├── config.go               # TTS configuration
│       ├── error_recovery.go       # Error handling and recovery
│       ├── message_monitor.go      # Text channel monitoring
│       ├── message_queue.go        # Message queuing system
│       ├── system.go               # TTS system integration
│       ├── tts_manager.go          # Google Cloud TTS integration
│       ├── tts_processor.go        # Message processing pipeline
│       ├── voice_manager.go        # Discord voice connections
│       └── *_test.go              # Comprehensive test suite
├── data/                   # Runtime data storage (gitignored)
├── .env                    # Environment configuration (gitignored)
├── .env.example           # Example environment configuration
├── .gitignore             # Git ignore patterns
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── LICENSE                # MIT license
└── README.md              # Project documentation
```

## Implemented Architecture Patterns

### Modular Design
- **Separation of Concerns**: Each package has a single, well-defined responsibility
- **Interface-Driven**: Components communicate through well-defined interfaces
- **Dependency Injection**: Services are injected rather than directly instantiated

### TTS System Components
```
internal/tts/
├── command_handlers.go     # Discord slash command implementations
├── config.go              # TTS configuration and validation
├── error_recovery.go      # Comprehensive error handling system
├── message_monitor.go     # Real-time text channel monitoring
├── message_queue.go       # Thread-safe message queuing
├── system.go              # System integration and initialization
├── tts_manager.go         # Google Cloud TTS integration
├── tts_processor.go       # Asynchronous message processing
├── voice_manager.go       # Discord voice connection management
└── comprehensive test suite with 100% coverage
```

## File Naming Conventions (Implemented)
- **Go Files**: Use snake_case for multi-word files: `message_monitor.go`, `error_recovery.go`
- **Package Names**: Short, lowercase, single words: `bot`, `tts`, `config`
- **Test Files**: End with `_test.go`: `voice_manager_test.go`
- **Interface Files**: Often combined with implementation for simplicity

## Code Organization Principles (Applied)
- **Minimal main.go**: Delegates to internal packages for all functionality
- **Clear Separation**: Bot logic, TTS processing, and configuration are separate
- **Interface Usage**: All major components use interfaces for testability
- **Package Cohesion**: Related functionality grouped in logical packages
- **Error Handling**: Centralized error recovery with configurable behavior
- **Testing**: Comprehensive test coverage with performance optimization

## Configuration Management
- **Environment Variables**: Sensitive configuration via `.env` file
- **JSON Storage**: Guild settings and user preferences in JSON files
- **Validation**: All configuration validated with clear error messages
- **Defaults**: Sensible defaults for all optional settings

## CI/CD Integration
- **GitHub Actions**: Automated testing and deployment workflows
- **Multi-platform**: Build verification on Windows, Linux, macOS
- **Quality Gates**: Automated testing, linting, and code coverage checks