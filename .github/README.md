# GitHub Actions Workflows

This directory contains the CI/CD workflows for the darrot Discord TTS bot.

## Workflows

### 1. Test Workflow (`test.yml`)
**Triggers:** Pull requests and pushes to `main` and `develop` branches

**Jobs:**
- **test**: Runs comprehensive test suite with coverage reporting
- **lint**: Code quality checks with golangci-lint
- **build**: Builds the application to verify compilation

**Features:**
- Go module caching for faster builds
- Coverage reporting with 80% minimum threshold
- Race condition detection
- Artifact upload for coverage reports

### 2. Release Workflow (`release.yml`)
**Triggers:** Pushes to `main` branch (excluding documentation changes)

**Jobs:**
- **test**: Full test suite validation
- **release**: Creates GitHub releases with Linux binaries

**Artifacts:**
- `darrot-linux-amd64`: Linux x86_64 binary
- `darrot-linux-arm64`: Linux ARM64 binary
- `checksums.txt`: SHA256 checksums for verification

**Features:**
- Automatic versioning based on date and commit hash
- Cross-compilation for multiple Linux architectures
- Automated release notes generation
- Binary checksums for security verification

### 3. Dependency Update Workflow (`dependency-update.yml`)
**Triggers:** Weekly schedule (Sundays at 2 AM UTC) and manual dispatch

**Features:**
- Automatic Go module updates
- Test validation before creating PR
- Automated pull request creation
- Clean dependency management with `go mod tidy`

## Requirements

### System Dependencies
All workflows install the following system dependencies:
- `libopus-dev`: Opus audio codec library
- `pkg-config`: Package configuration tool
- `gcc-aarch64-linux-gnu`: Cross-compiler for ARM64 (release only)

### Go Version
All workflows use Go 1.25.1 to match the project's `go.mod` specification.

## Code Quality

### Linting Configuration
The project uses golangci-lint with configuration in `.golangci.yml`:
- Comprehensive linter set including security, performance, and style checks
- Custom rules for test files and command packages
- Line length limit of 140 characters
- Import organization with local package preferences

### Coverage Requirements
- Minimum 80% test coverage required for all workflows
- Coverage reports generated and uploaded as artifacts
- Race condition detection enabled in all test runs

## Security

### Dependency Management
- Weekly automated dependency updates
- Dependency verification with `go mod verify`
- Automated testing of updated dependencies

### Release Security
- SHA256 checksums generated for all release binaries
- Secure token usage with `GITHUB_TOKEN`
- No sensitive information in workflow files

## Usage

### Running Tests Locally
```bash
# Install system dependencies (Ubuntu/Debian)
sudo apt-get install -y libopus-dev pkg-config

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Manual Release
Releases are automatically created when code is pushed to the `main` branch. To trigger a manual release:
1. Ensure all tests pass
2. Push to `main` branch
3. The release workflow will automatically create a new release

### Manual Dependency Update
```bash
# Trigger the dependency update workflow manually
gh workflow run dependency-update.yml
```

## Troubleshooting

### Common Issues
1. **Coverage Below 80%**: Add more tests or adjust coverage threshold
2. **Linting Failures**: Run `golangci-lint run` locally and fix issues
3. **Build Failures**: Ensure all system dependencies are installed
4. **Cross-compilation Issues**: Verify ARM64 cross-compiler installation

### Debugging
- Check workflow logs in the GitHub Actions tab
- Download coverage artifacts for detailed analysis
- Use `go mod verify` to check dependency integrity