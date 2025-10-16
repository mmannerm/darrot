# Darrot Container Quick Reference

## Essential Commands

### Build and Run
```bash
# Build image (with pull for latest base images)
podman build --pull -t darrot:latest .

# Run with basic config
podman run -d --name darrot-bot --env-file .env -v ./data:/app/data:Z darrot:latest

# Run with resource limits
podman run -d --name darrot-bot --memory=256m --cpus=0.5 --env-file .env -v ./data:/app/data:Z darrot:latest
```

### Management
```bash
# View logs
podman logs -f darrot-bot

# Stop/start
podman stop darrot-bot
podman start darrot-bot

# Remove container
podman rm darrot-bot

# Remove image
podman rmi darrot:latest
```

### Debugging
```bash
# Shell access
podman exec -it darrot-bot /bin/sh

# Check processes
podman exec darrot-bot ps aux

# Check environment
podman exec darrot-bot env | grep DISCORD

# Debug mode
podman run -d --name darrot-debug --env-file .env -e LOG_LEVEL=DEBUG -v ./data:/app/data:Z darrot:latest
```

## Required Files

### .env (Required)
```bash
DISCORD_TOKEN=your_bot_token_here
LOG_LEVEL=INFO
```

### Optional: Google Cloud TTS
```bash
# Create credentials directory
mkdir -p ./credentials
cp /path/to/credentials.json ./credentials/

# Run with GCP
podman run -d --name darrot-bot \
  --env-file .env \
  -e GOOGLE_CLOUD_CREDENTIALS_PATH=/app/credentials/credentials.json \
  -v ./data:/app/data:Z \
  -v ./credentials:/app/credentials:ro,Z \
  darrot:latest
```

## Testing
```bash
# Run acceptance tests
./test-container.sh

# Manual test
bash tests/container/acceptance_test.sh
```

## Troubleshooting

### Common Issues
- **Registry resolution**: Use `podman build --pull -t darrot:latest .`
- **Opus build errors**: Ensure Dockerfile includes `opusfile-dev` package
- **Permission denied**: `sudo chown -R 1001:1001 ./data`
- **Container won't start**: `podman logs darrot-bot`
- **Memory issues**: Add `--memory=512m` to run command
- **SELinux issues**: Ensure `:Z` flag on volume mounts

### Registry Configuration (if needed)
```bash
# Configure unqualified registries
sudo mkdir -p /etc/containers
echo 'unqualified-search-registries = ["docker.io"]' | sudo tee -a /etc/containers/registries.conf
```

### Health Check
```bash
# Check container health
podman inspect darrot-bot | grep -A 5 Health

# Manual health check
podman exec darrot-bot pgrep darrot
```