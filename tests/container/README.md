# Container Testing

This directory contains container structure tests and related testing infrastructure for the darrot Discord TTS bot.

## Overview

The container testing framework validates:
- Container structure and security compliance
- Binary functionality and dependencies
- File permissions and ownership
- Runtime environment configuration

## Files

- `structure-test.yaml` - Google Container Structure Test configuration
- `README.md` - This documentation file

## Quick Start

### Prerequisites

1. **Podman** - For building container images (Docker also supported as fallback)
2. **container-structure-test** - Google's container testing tool

### Installation

Install the container structure test tool:

```bash
# Using the provided script
make container-test-install

# Or manually
./scripts/install-container-structure-test.sh
```

### Running Tests

#### Option 1: Using Make (Recommended)

```bash
# Run all container tests
make container-test

# Build container only
make container-build

# Quick container validation
make container-test-quick
```

#### Option 2: Using Scripts Directly

```bash
# Full container structure tests
./scripts/run-container-tests.sh

# Quick validation tests
./scripts/test-container-quick.sh
```

#### Option 3: Manual Execution

```bash
# Build the container (using podman by default)
podman build -t darrot:test .

# Or with docker
docker build -t darrot:test .

# Run structure tests
container-structure-test test \
  --image darrot:test \
  --config tests/container/structure-test.yaml \
  --output json
```

## Test Categories

### Command Tests
- Binary existence and executability
- Dependency validation (Opus libraries)
- System component verification

### File Existence Tests
- Application binary permissions
- Directory structure validation
- Required system files

### Metadata Tests
- Non-root user execution
- Working directory configuration
- Port exposure validation
- Entrypoint verification

### Security Tests
- User permissions and ownership
- File system security
- Container attack surface

## CI/CD Integration

Container tests are automatically run in GitHub Actions:

1. **Build Workflow** - Includes container tests as a dependency
2. **Container Tests Workflow** - Dedicated container validation
3. **Security Scanning** - Vulnerability assessment with Trivy

### Workflow Files
- `.github/workflows/container-tests.yml` - Main container testing workflow
- `.github/workflows/build.yml` - Updated to include container tests

## Local Development

### Development Workflow

1. Make code changes
2. Run quick tests: `make container-test-quick` or `./scripts/test-container-quick.sh`
3. Run full tests: `make container-test`
4. Commit changes

### Debugging Failed Tests

1. **Check test output** - Review JSON results in `tests/results/`
2. **Interactive debugging** - Use `make container-shell` to inspect container
3. **Verbose logging** - Run scripts with `--verbose` flag
4. **Container runtime** - Set `CONTAINER_RUNTIME=docker` to use Docker instead of Podman

### Common Issues

#### Binary Not Found
- Ensure Dockerfile builds the binary correctly
- Check build context and COPY instructions

#### Permission Errors
- Verify non-root user configuration
- Check file ownership and permissions

#### Library Dependencies
- Ensure Opus libraries are installed in runtime image
- Verify pkg-config is available

#### Test Configuration
- Validate YAML syntax in `structure-test.yaml`
- Check file paths match container structure

## Configuration

### Container Runtime

The testing framework supports both Podman and Docker:

```bash
# Use Podman (default)
make container-test

# Use Docker explicitly
CONTAINER_RUNTIME=docker make container-test

# Set permanently in environment
export CONTAINER_RUNTIME=docker
```

The scripts will automatically fall back to Docker if Podman is not available.

### Test Configuration File

The `structure-test.yaml` file defines all container tests:

```yaml
schemaVersion: 2.0.0

commandTests:
  - name: "test description"
    command: "command to run"
    args: ["arg1", "arg2"]
    expectedOutput: ["expected output"]
    exitCode: 0

fileExistenceTests:
  - name: "file description"
    path: "/path/to/file"
    shouldExist: true
    permissions: "-rwxr-xr-x"

metadataTest:
  user: "darrot"
  workdir: "/app"
  exposedPorts: ["8080"]
```

### Customization

To add new tests:

1. Edit `structure-test.yaml`
2. Add test cases in appropriate sections
3. Test locally with `make container-test`
4. Commit changes

## Troubleshooting

### Tool Installation Issues

```bash
# Check if tool is installed
container-structure-test version

# Reinstall if needed
./scripts/install-container-structure-test.sh
```

### Container Runtime Issues

```bash
# Check Podman
podman version

# Or check Docker
docker version

# Clean up images (Podman)
podman system prune -f

# Clean up images (Docker)
docker system prune -f

# Switch container runtime
export CONTAINER_RUNTIME=docker  # or podman
```

### Test Failures

```bash
# Run with verbose output
./scripts/run-container-tests.sh --verbose

# Check specific test results
cat tests/results/container-structure-test-results.json | jq '.Results[] | select(.Pass == false)'
```

## Resources

- [Google Container Structure Test](https://github.com/GoogleContainerTools/container-structure-test)
- [Container Security Best Practices](https://cloud.google.com/architecture/best-practices-for-building-containers)
- [Docker Security](https://docs.docker.com/engine/security/)