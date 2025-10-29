# Makefile-Based Container Testing System

## Overview

The container testing framework has been converted from shell script-based execution to a Makefile-based system with proper dependency tracking. This eliminates unnecessary container rebuilds and provides a more efficient development workflow.

## Key Features

### 1. Dependency Tracking
- Uses timestamp files (`.build-stamps/`) to track when containers were last built
- Only rebuilds containers when source files, Dockerfiles, or dependencies change
- Avoids expensive rebuilds when no changes have been made

### 2. Multiple Container Support
- **Main Application**: `darrot:test` - Built from main Dockerfile
- **Mock Discord Server**: `mock-discord:test` - Built from `tests/mock-discord/Dockerfile`
- **Acceptance Tests**: `acceptance-tests:test` - Built from `tests/acceptance/Dockerfile.test`

### 3. Container Runtime Compatibility
- Supports both Podman and Docker
- Automatic runtime detection
- Handles Podman-specific requirements (Docker API service for container-structure-test)

## Available Make Targets

### Container Building
```bash
make container-build          # Build main container (with dependency tracking)
make container-build-force    # Force rebuild main container
make acceptance-build         # Build all acceptance test images
```

### Container Testing
```bash
make container-test           # Run container structure tests
make container-test-quick     # Run quick container validation
make container-test-install   # Install container-structure-test tool
```

### Acceptance Testing
```bash
make acceptance-test          # Run all acceptance tests
make acceptance-test-core     # Run core functionality tests
make acceptance-test-concurrent # Run concurrent tests
make acceptance-test-error    # Run error resilience tests
```

### Development Workflow
```bash
make status                   # Show build status
make clean                    # Clean all build artifacts
make clean-stamps             # Force rebuild on next make
make show-deps                # Show build dependencies
```

### Combined Targets
```bash
make all                      # Run lint, test, container-test
make ci                       # Run CI pipeline locally
make test-all                 # Run unit, container, and acceptance tests
```

## Dependency Tracking Details

### Build Dependencies
The system tracks these files for container rebuilds:
- All Go source files (excluding tests): `internal/**/*.go`, `cmd/**/*.go`
- Go module files: `go.mod`, `go.sum`
- Main Dockerfile: `Dockerfile`

### Test Dependencies
- Mock Discord server: `tests/mock-discord/**/*.go`, `tests/mock-discord/Dockerfile`
- Acceptance tests: `tests/acceptance/**/*.go`, `tests/acceptance/Dockerfile.test`

### Timestamp Files
Located in `.build-stamps/`:
- `container.stamp` - Main darrot container
- `mock-discord.stamp` - Mock Discord server
- `acceptance-test.stamp` - Acceptance test runner
- `container-test-tool.stamp` - Container structure test tool

## Usage Examples

### First-time Setup
```bash
# Install container testing tool
make container-test-install

# Build and test main container
make container-build
make container-test-quick

# Run full container structure tests
make container-test
```

### Development Workflow
```bash
# Check what needs to be built
make status

# Build only what's needed
make container-build

# Run quick validation
make container-test-quick

# Run all tests
make test-all
```

### CI/CD Integration
```bash
# Run complete CI pipeline
make ci

# This runs:
# - go fmt, go vet, golangci-lint
# - go test with race detection
# - container build
# - container structure tests
```

## Performance Benefits

### Before (Shell Script)
- Always rebuilt all containers
- No dependency tracking
- Podman-compose issues with service dependencies
- Long execution times even for small changes

### After (Makefile)
- Only rebuilds when source files change
- Proper dependency tracking with timestamps
- Efficient Podman/Docker handling
- Fast incremental builds

### Example Timing
```bash
# First build (cold)
make container-build  # ~2-3 minutes

# No changes
make container-build  # ~0.1 seconds (Nothing to be done)

# After source change
make container-build  # ~30-60 seconds (incremental)
```

## Configuration

### Container Runtime
Set the container runtime:
```bash
# Use Podman (default)
make container-build

# Use Docker
CONTAINER_RUNTIME=docker make container-build
```

### Environment Variables
- `CONTAINER_RUNTIME`: `podman` or `docker` (default: `podman`)
- `VERSION`: Version string for builds (default: `dev`)
- `COMMIT`: Git commit hash (auto-detected)
- `DATE`: Build date (auto-generated)

## Troubleshooting

### Force Rebuild
If dependency tracking seems incorrect:
```bash
make clean-stamps
make container-build
```

### Check Dependencies
See what files trigger rebuilds:
```bash
make show-deps
```

### Debug Build Issues
Check current build status:
```bash
make status
```

### Container Runtime Issues
For Podman socket issues:
```bash
# Start Podman socket service
systemctl --user start podman.socket

# Or use Docker
CONTAINER_RUNTIME=docker make container-test
```

## Integration with Existing Workflow

### Replacing Old Scripts
- `scripts/run-container-tests.sh` → `make container-test`
- `tests/acceptance/run-container-tests.sh` → `make acceptance-test`
- Manual container builds → `make container-build`

### CI/CD Updates
The GitHub Actions workflows can now use:
```yaml
- name: Run container tests
  run: make container-test

- name: Run acceptance tests  
  run: make acceptance-test
```

This provides better caching and faster CI execution times.