# Project Structure

## Current Organization
```
darrot/
├── .git/           # Git version control
├── .kiro/          # Kiro AI assistant configuration
│   └── steering/   # AI guidance documents
├── .vscode/        # VSCode editor settings
├── .gitignore      # Git ignore patterns for Go projects
├── LICENSE         # MIT license
└── README.md       # Project documentation
```

## Recommended Go Project Structure
When implementing the core application, follow standard Go project layout:

```
darrot/
├── cmd/
│   └── darrot/     # Main application entry point
│       └── main.go
├── internal/       # Private application code
│   ├── bot/        # Discord bot logic
│   ├── tts/        # Text-to-Speech functionality
│   └── config/     # Configuration management
├── pkg/            # Public library code (if any)
├── scripts/        # Build and deployment scripts
├── docs/           # Additional documentation
├── .env.example    # Example environment configuration
├── go.mod          # Go module definition
└── go.sum          # Go module checksums
```

## File Naming Conventions
- Use lowercase with underscores for Go files: `discord_handler.go`
- Package names should be short, lowercase, single words
- Test files should end with `_test.go`

## Code Organization
- Keep main.go minimal - delegate to internal packages
- Separate concerns: bot logic, TTS processing, configuration
- Use interfaces for testability and modularity
- Group related functionality in packages