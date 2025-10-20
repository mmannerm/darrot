# Configuration Guide

This document provides comprehensive information about configuring the darrot Discord TTS bot using the new Cobra/Viper architecture.

## Configuration Methods

darrot supports multiple configuration methods with the following precedence order (highest to lowest):

1. **CLI flags** - Command-line arguments (e.g., `--discord-token`)
2. **Environment variables** - With `DRT_` prefix (e.g., `DRT_DISCORD_TOKEN`)
3. **Configuration files** - YAML, JSON, or TOML format
4. **Default values** - Built-in sensible defaults

## Configuration Files

### Supported Formats

darrot automatically searches for configuration files in the following locations and formats:

#### Search Locations
1. `./darrot.yaml` (current directory)
2. `./darrot.json` (current directory)
3. `./darrot.toml` (current directory)
4. `~/.darrot.yaml` (user home directory)
5. `~/.darrot.json` (user home directory)
6. `~/.darrot.toml` (user home directory)
7. `/etc/darrot/config.yaml` (system-wide, Linux/macOS)
8. `%APPDATA%\darrot\config.yaml` (Windows)

#### Custom Configuration File
You can specify a custom configuration file using the `--config` flag:

```bash
./darrot start --config /path/to/my-config.yaml
```

### Example Configuration Files

#### YAML Format (Recommended)
```yaml
# darrot.yaml
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

#### JSON Format
```json
{
  "discord_token": "your_bot_token_here",
  "log_level": "INFO",
  "tts": {
    "default_voice": "en-US-Standard-A",
    "default_speed": 1.0,
    "default_volume": 1.0,
    "max_queue_size": 10,
    "max_message_length": 500
  },
  "cli": {
    "enable_colors": true,
    "completion_shell": "bash"
  }
}
```

#### TOML Format
```toml
discord_token = "your_bot_token_here"
log_level = "INFO"

[tts]
default_voice = "en-US-Standard-A"
default_speed = 1.0
default_volume = 1.0
max_queue_size = 10
max_message_length = 500

[cli]
enable_colors = true
completion_shell = "bash"
```

## Environment Variables

All environment variables must use the `DRT_` prefix. This change was made to support the new CLI architecture and avoid conflicts with other applications.

### Core Configuration
- `DRT_DISCORD_TOKEN` - Discord bot token (required)
- `DRT_LOG_LEVEL` - Logging level (DEBUG, INFO, WARN, ERROR)

### Google Cloud TTS Authentication (Optional)
Use standard Google Cloud SDK authentication instead of configuration options:
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account JSON file
- Or use `gcloud auth application-default login` for development

### TTS Configuration
- `DRT_TTS_DEFAULT_VOICE` - Default TTS voice
- `DRT_TTS_DEFAULT_SPEED` - Speech speed (0.25-4.0)
- `DRT_TTS_DEFAULT_VOLUME` - Speech volume (0.0-2.0)
- `DRT_TTS_MAX_QUEUE_SIZE` - Maximum queue size (1-100)
- `DRT_TTS_MAX_MESSAGE_LENGTH` - Maximum message length (1-2000)

### Example .env File
```bash
# .env
DRT_DISCORD_TOKEN=your_bot_token_here
DRT_LOG_LEVEL=INFO
DRT_TTS_DEFAULT_VOICE=en-US-Standard-A
DRT_TTS_DEFAULT_SPEED=1.0
```

## CLI Flags

All configuration options are available as CLI flags for the `start` command:

### Core Flags
```bash
--discord-token string              Discord bot token
--config string                     Configuration file path
--log-level string                  Log level (DEBUG, INFO, WARN, ERROR)
```

### TTS Flags
```bash
--tts-default-voice string          Default TTS voice
--tts-default-speed float           Speech speed (0.25-4.0)
--tts-default-volume float          Speech volume (0.0-2.0)
--tts-max-queue-size int            Maximum queue size (1-100)
--tts-max-message-length int        Maximum message length (1-2000)
```

### Example Usage
```bash
# Start with CLI flags
./darrot start --discord-token "your_token" --log-level DEBUG

# Start with configuration file
./darrot start --config darrot.yaml

# Mix configuration file with CLI overrides
./darrot start --config darrot.yaml --log-level DEBUG --tts-default-speed 1.2
```

## Configuration Management Commands

### Validate Configuration
Check your configuration without starting the bot:

```bash
./darrot config validate
```

This command will:
- Load configuration from all sources
- Validate all values and ranges
- Report any errors or missing required values
- Show which configuration sources are being used

### Show Effective Configuration
Display the final configuration that will be used:

```bash
# Human-readable format
./darrot config show

# JSON format
./darrot config show --format json
```

This command shows:
- All configuration values
- The source of each value (default, file, env, flag)
- Masked sensitive values (tokens are hidden)

### Create Configuration File
Generate a configuration file from current settings:

```bash
# Create in default location (darrot.yaml)
./darrot config create

# Create in specific location
./darrot config create --output /path/to/config.yaml

# Create with current environment variables
DRT_DISCORD_TOKEN=your_token ./darrot config create --output my-config.yaml
```

## Configuration Options Reference

### Required Options

| Option | Type | Description | Environment Variable | CLI Flag |
|--------|------|-------------|---------------------|----------|
| `discord_token` | string | Discord bot token | `DRT_DISCORD_TOKEN` | `--discord-token` |

### Optional Options

| Option | Type | Default | Description | Environment Variable | CLI Flag |
|--------|------|---------|-------------|---------------------|----------|
| `log_level` | string | INFO | Logging level | `DRT_LOG_LEVEL` | `--log-level` |

### TTS Options

| Option | Type | Default | Range | Description | Environment Variable | CLI Flag |
|--------|------|---------|-------|-------------|---------------------|----------|
| `tts.default_voice` | string | en-US-Standard-A | - | Default TTS voice | `DRT_TTS_DEFAULT_VOICE` | `--tts-default-voice` |
| `tts.default_speed` | float | 1.0 | 0.25-4.0 | Speech speed | `DRT_TTS_DEFAULT_SPEED` | `--tts-default-speed` |
| `tts.default_volume` | float | 1.0 | 0.0-2.0 | Speech volume | `DRT_TTS_DEFAULT_VOLUME` | `--tts-default-volume` |
| `tts.max_queue_size` | int | 10 | 1-100 | Max queue size | `DRT_TTS_MAX_QUEUE_SIZE` | `--tts-max-queue-size` |
| `tts.max_message_length` | int | 500 | 1-2000 | Max message length | `DRT_TTS_MAX_MESSAGE_LENGTH` | `--tts-max-message-length` |

### CLI Options

| Option | Type | Default | Description | Environment Variable | CLI Flag |
|--------|------|---------|-------------|---------------------|----------|
| `cli.enable_colors` | bool | true | Enable colored output | `DRT_CLI_ENABLE_COLORS` | - |
| `cli.completion_shell` | string | bash | Default completion shell | `DRT_CLI_COMPLETION_SHELL` | - |

## Migration from Old Configuration

### Environment Variables Migration

If you have existing environment variables without the `DRT_` prefix, you can migrate them:

```bash
# Automated migration script
sed -i 's/^DISCORD_TOKEN=/DRT_DISCORD_TOKEN=/' .env
sed -i 's/^LOG_LEVEL=/DRT_LOG_LEVEL=/' .env

sed -i 's/^TTS_/DRT_TTS_/' .env
```

### Command Migration

Old command format:
```bash
./darrot  # Direct execution
```

New command format:
```bash
./darrot start  # Use start subcommand
```

## Google Cloud TTS Authentication

darrot uses the standard Google Cloud SDK authentication methods instead of configuration file options. This follows Google Cloud best practices and provides better security.

### Authentication Methods

#### Method 1: Service Account Key (Recommended for Production)
```bash
# Set the environment variable
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
./darrot start
```

#### Method 2: Application Default Credentials (Development)
```bash
# Authenticate with your Google account
gcloud auth application-default login
./darrot start
```

#### Method 3: Container/VM Metadata (Cloud Deployment)
When running on Google Cloud Platform (GCE, GKE, Cloud Run, etc.), authentication is automatic through metadata service.

### Setup Instructions

1. **Enable the Text-to-Speech API**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Enable the [Text-to-Speech API](https://console.cloud.google.com/apis/library/texttospeech.googleapis.com)

2. **Create Service Account (for production)**
   ```bash
   # Create service account
   gcloud iam service-accounts create darrot-tts \
     --description="Service account for darrot TTS bot" \
     --display-name="Darrot TTS"
   
   # Grant Text-to-Speech permissions
   gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
     --member="serviceAccount:darrot-tts@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/cloudtts.user"
   
   # Create and download key
   gcloud iam service-accounts keys create darrot-tts-key.json \
     --iam-account=darrot-tts@YOUR_PROJECT_ID.iam.gserviceaccount.com
   ```

3. **Set Authentication**
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/darrot-tts-key.json
   ```

### Container Deployment
```dockerfile
# In your Dockerfile or container environment
ENV GOOGLE_APPLICATION_CREDENTIALS=/app/credentials/gcp-key.json
COPY gcp-key.json /app/credentials/gcp-key.json
```

## Configuration Examples

### Development Environment
```yaml
# darrot-dev.yaml
discord_token: "dev_bot_token"
log_level: "DEBUG"

tts:
  max_queue_size: 5
  max_message_length: 200
  default_speed: 1.2

cli:
  enable_colors: true
```

### Production Environment
```yaml
# darrot-prod.yaml
discord_token: "prod_bot_token"
log_level: "WARN"

tts:
  default_voice: "en-US-Neural2-A"
  max_queue_size: 20
  max_message_length: 1000
  default_speed: 1.0
  default_volume: 0.9
```

```bash
# Set Google Cloud authentication
export GOOGLE_APPLICATION_CREDENTIALS=/etc/darrot/gcp-credentials.json
./darrot start --config darrot-prod.yaml
```

### High-Performance Setup
```yaml
# darrot-performance.yaml
discord_token: "your_token"
log_level: "ERROR"

tts:
  max_queue_size: 50
  max_message_length: 1500
  default_speed: 1.3
```

## Troubleshooting Configuration

### Common Issues

1. **Configuration not loading**
   ```bash
   # Check which config file is being used
   ./darrot config show
   
   # Validate configuration
   ./darrot config validate
   ```

2. **Environment variables not working**
   ```bash
   # Verify environment variables are set with DRT_ prefix
   env | grep DRT_
   
   # Test with explicit config
   ./darrot start --discord-token "your_token"
   ```

3. **Invalid configuration values**
   ```bash
   # Validate will show specific errors
   ./darrot config validate
   
   # Example output:
   # Error: tts.default_speed: 5.0 is not valid (must be between 0.25 and 4.0)
   ```

### Debug Configuration Loading

Enable debug logging to see configuration loading details:

```bash
./darrot start --log-level DEBUG
```

This will show:
- Which configuration files are found and loaded
- Environment variable mappings
- Final configuration values and their sources
- Any validation errors or warnings

## Security Considerations

### Sensitive Values

- **Never commit tokens to version control**
- **Use environment variables or secure config files for tokens**
- **The `config show` command masks sensitive values**
- **Configuration files should have restricted permissions (600)**

### Best Practices

1. **Use environment variables for sensitive data**:
   ```bash
   export DRT_DISCORD_TOKEN="your_secret_token"
   ./darrot start --config darrot.yaml
   ```

2. **Separate configuration by environment**:
   ```bash
   ./darrot start --config config/production.yaml
   ./darrot start --config config/development.yaml
   ```

3. **Validate configuration in CI/CD**:
   ```bash
   ./darrot config validate --config config/production.yaml
   ```

4. **Use configuration files for non-sensitive settings**:
   ```yaml
   # darrot.yaml (safe to commit)
   log_level: "INFO"
   tts:
     default_voice: "en-US-Standard-A"
     default_speed: 1.0
   # Token provided via environment variable
   ```