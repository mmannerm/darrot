# Darrot Discord TTS Bot - Container Deployment

This guide covers running the darrot Discord TTS bot using Podman containers.

## Prerequisites

- Podman installed on your system
- Discord bot token from [Discord Developer Portal](https://discord.com/developers/applications)
- Google Cloud TTS credentials (optional, for enhanced voices)

## Quick Start

### 1. Clone and Prepare

```bash
git clone <repository-url>
cd darrot
```

### 2. Configure Environment

```bash
# Copy the container environment template
cp container-env.example .env

# Edit .env with your Discord bot token
# DISCORD_TOKEN=your_actual_bot_token_here
```

### 3. Build and Run

```bash
# Build the container
podman build -t darrot:latest .

# Run with basic configuration
podman run -d \
  --name darrot-bot \
  --env-file .env \
  -v ./data:/app/data:Z \
  darrot:latest
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | Yes | - | Discord bot token |
| `LOG_LEVEL` | No | INFO | Logging level (DEBUG, INFO, WARN, ERROR) |
| `TTS_DEFAULT_VOICE` | No | en-US-Standard-A | Default TTS voice |
| `TTS_DEFAULT_SPEED` | No | 1.0 | Speech speed (0.25-4.0) |
| `TTS_DEFAULT_VOLUME` | No | 1.0 | Speech volume (0.0-2.0) |
| `TTS_MAX_QUEUE_SIZE` | No | 10 | Max messages in queue (1-100) |
| `TTS_MAX_MESSAGE_LENGTH` | No | 500 | Max message length (1-2000) |
| `GOOGLE_CLOUD_CREDENTIALS_PATH` | No | - | Path to GCP credentials JSON |

### Google Cloud TTS Setup (Optional)

1. Create a service account in Google Cloud Console
2. Download the credentials JSON file
3. Mount it to the container:

```bash
# Create credentials directory
mkdir -p ./credentials

# Copy your credentials file
cp /path/to/your/credentials.json ./credentials/

# Run with GCP credentials
podman run -d \
  --name darrot-bot \
  --env-file .env \
  -v ./data:/app/data:Z \
  -v ./credentials:/app/credentials:ro,Z \
  -e GOOGLE_CLOUD_CREDENTIALS_PATH=/app/credentials/credentials.json \
  darrot:latest
```

## Deployment Options

### Option 1: Direct Podman Run

```bash
# Basic deployment
podman run -d \
  --name darrot-bot \
  --restart unless-stopped \
  --env-file .env \
  -v ./data:/app/data:Z \
  darrot:latest

# With resource limits
podman run -d \
  --name darrot-bot \
  --restart unless-stopped \
  --memory=256m \
  --cpus=0.5 \
  --env-file .env \
  -v ./data:/app/data:Z \
  darrot:latest
```

### Option 2: Docker Compose (Podman Compatible)

```bash
# Start the service
podman-compose up -d

# View logs
podman-compose logs -f

# Stop the service
podman-compose down
```

### Option 3: Podman Pod

```bash
# Create a pod for the bot
podman pod create --name darrot-pod -p 8080:8080

# Run the bot in the pod
podman run -d \
  --pod darrot-pod \
  --name darrot-bot \
  --env-file .env \
  -v ./data:/app/data:Z \
  darrot:latest
```

## Container Management

### View Logs

```bash
# Follow logs
podman logs -f darrot-bot

# View recent logs
podman logs --tail 50 darrot-bot
```

### Health Monitoring

```bash
# Check container status
podman ps

# Check health status
podman inspect darrot-bot | grep -A 5 Health

# Manual health check
podman exec darrot-bot pgrep darrot
```

### Updates and Maintenance

```bash
# Stop the container
podman stop darrot-bot

# Remove old container
podman rm darrot-bot

# Rebuild with latest code
podman build -t darrot:latest .

# Start new container
podman run -d \
  --name darrot-bot \
  --env-file .env \
  -v ./data:/app/data:Z \
  darrot:latest
```

## Security Considerations

The container is configured with security best practices:

- **Non-root user**: Runs as user `darrot` (UID 1001)
- **Read-only filesystem**: Container filesystem is read-only
- **No new privileges**: Prevents privilege escalation
- **Resource limits**: Memory and CPU limits prevent resource exhaustion
- **Minimal base image**: Alpine Linux for reduced attack surface

### SELinux Compatibility

If using SELinux, volumes are mounted with the `:Z` flag for proper labeling:

```bash
-v ./data:/app/data:Z
-v ./credentials:/app/credentials:ro,Z
```

## Troubleshooting

### Common Issues

1. **Registry Resolution Error**
   ```bash
   # Error: short-name "golang:1.23-alpine" did not resolve to an alias
   
   # Solution 1: Use fully qualified names (already fixed in Dockerfile)
   # The Dockerfile now uses docker.io/golang:1.23-alpine
   
   # Solution 2: Configure registries (if needed)
   sudo mkdir -p /etc/containers
   echo 'unqualified-search-registries = ["docker.io"]' | sudo tee -a /etc/containers/registries.conf
   
   # Solution 3: Use Docker Hub explicitly
   podman build --pull -t darrot:latest .
   ```

2. **Permission Denied on Data Directory**
   ```bash
   # Fix ownership
   sudo chown -R 1001:1001 ./data
   ```

3. **Container Won't Start**
   ```bash
   # Check logs for errors
   podman logs darrot-bot
   
   # Verify environment variables
   podman exec darrot-bot env | grep DISCORD
   ```

4. **Audio Issues**
   ```bash
   # Verify Opus library is available
   podman exec darrot-bot ldd /app/darrot | grep opus
   ```

5. **Memory Issues**
   ```bash
   # Monitor resource usage
   podman stats darrot-bot
   
   # Increase memory limit
   podman update --memory=512m darrot-bot
   ```

6. **Build Issues on Different Architectures**
   ```bash
   # Force platform for cross-compilation
   podman build --platform linux/amd64 -t darrot:latest .
   
   # Or for ARM64
   podman build --platform linux/arm64 -t darrot:latest .
   ```

7. **Opus Library Build Errors**
   ```bash
   # Error: Package 'opusfile' not found
   # This indicates missing opusfile-dev package in Alpine
   
   # The Dockerfile includes both opus-dev and opusfile-dev
   # If you see this error, ensure you're using the latest Dockerfile
   
   # Manual verification in container:
   podman run --rm -it docker.io/alpine:3.19 sh
   apk add --no-cache opus-dev opusfile-dev pkgconfig
   pkg-config --exists opusfile && echo "opusfile found" || echo "opusfile missing"
   ```

8. **HEALTHCHECK Warning (Safe to Ignore)**
   ```bash
   # Warning: HEALTHCHECK is not supported for OCI image format
   # This is expected with Podman's default OCI format
   
   # The warning doesn't affect functionality
   # Health checks still work via manual commands:
   podman exec darrot-bot pgrep darrot
   
   # To use Docker format (if needed):
   podman build --format docker -t darrot:latest .
   ```

### Debug Mode

Run with debug logging:

```bash
podman run -d \
  --name darrot-bot-debug \
  --env-file .env \
  -e LOG_LEVEL=DEBUG \
  -v ./data:/app/data:Z \
  darrot:latest
```

## Production Deployment

For production environments:

1. **Use specific image tags** instead of `latest`
2. **Set up log rotation** to prevent disk space issues
3. **Monitor resource usage** and adjust limits accordingly
4. **Implement backup strategy** for the data directory
5. **Use secrets management** instead of environment files

### Systemd Service (Optional)

Create a systemd service for automatic startup:

```bash
# Generate systemd unit file
podman generate systemd --new --name darrot-bot > ~/.config/systemd/user/darrot-bot.service

# Enable and start service
systemctl --user daemon-reload
systemctl --user enable darrot-bot.service
systemctl --user start darrot-bot.service
```

## Container Registry

### Building for Multiple Architectures

```bash
# Build for AMD64 and ARM64
podman buildx build \
  --platform linux/amd64,linux/arm64 \
  -t darrot:latest \
  .
```

### Pushing to Registry

```bash
# Tag for registry
podman tag darrot:latest your-registry.com/darrot:latest

# Push to registry
podman push your-registry.com/darrot:latest
```

## Testing

### Container Acceptance Tests

The acceptance tests automatically detect and use your existing `.env` file if available, or create a minimal test configuration. They also support Google Cloud credentials from your host environment.

```bash
# Test container build and functionality
./test-container.sh

# Windows
test-container.bat

# Manual Podman test
bash tests/container/acceptance_test.sh

# Test with your actual credentials (optional)
export GOOGLE_CLOUD_CREDENTIALS_PATH=/path/to/your/credentials.json
bash tests/container/acceptance_test.sh
```

**Test Features:**
- Uses existing `.env` file from project root if available
- Falls back to minimal test configuration if no `.env` found
- Automatically mounts Google Cloud credentials from host environment
- Validates both missing and present credential scenarios
- Tests environment variable handling and defaults
- Verifies container security settings and dependencies

**Test Scenarios:**
1. Container build with Opus dependencies
2. Image properties and security configuration
3. Application startup with various credential scenarios
4. Binary functionality and version checking
5. Filesystem permissions and user context
6. Environment variable loading and defaults
7. Health check functionality
8. Resource limits compliance
9. Volume mount operations

## Support

For issues related to:
- **Container deployment**: Check this documentation and container logs
- **Bot functionality**: See the main README.md
- **Discord integration**: Verify bot permissions and token validity
- **TTS issues**: Check Google Cloud credentials and API quotas