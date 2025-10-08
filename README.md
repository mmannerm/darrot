# darrot
Discord Parrot Text-to-Speech (TTS) AI app. Listens to chat channel and speaks

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

### Running the Bot

```bash
# Install dependencies
go mod tidy

# Build the application
go build -o darrot ./cmd/darrot

# Run the bot
./darrot
```

### Testing the Bot

Once the bot is running and invited to your server:
1. Type `/` in any channel where the bot has access
2. Look for the `/test` command in the autocomplete
3. Execute `/test` to verify the bot responds with "Hello World"

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

### Code Formatting
```bash
go fmt ./...
go vet ./...
```
