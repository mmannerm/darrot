---
inclusion: always
---

# Development Workflow & Code Quality

## Essential Pre-Work
**Always ensure main branch is current before starting any task.**

## Branch Naming Convention
Use kebab-case with conventional commit prefixes:
- `feat/feature-name` - New features
- `fix/issue-description` - Bug fixes  
- `refactor/component-name` - Code refactoring
- `docs/topic` - Documentation updates
- `ci/workflow-name` - CI/CD changes
- `perf/optimization-area` - Performance improvements

## Pre-Commit Requirements
**Always run these commands before committing:**
```bash
go test ./...        # All tests must pass
go fmt ./...         # Format code consistently
go vet ./...         # Lint for potential issues
go mod tidy          # Clean up dependencies
```

## File Management Rules

### Never Commit
- `darrot-config.yaml`, `darrot-config.json`, `darrot-config.toml` - May contain sensitive tokens/keys
- `data/` directory - Runtime user configurations
- Compiled binaries (`darrot.exe`, `darrot`)
- Coverage reports (`coverage.out`, `coverage.html`)

### Always Include
- Test files (`*_test.go`) for all new functionality
- Documentation updates for feature changes
- Configuration example updates for new options

## Go Code Standards

### Architecture Patterns
- **Package Organization**: Use `internal/` for private application code
- **Error Handling**: Leverage `internal/tts/error_recovery.go` for centralized error management
- **Interfaces**: Define interfaces for all major components to enable testing
- **Context Propagation**: Pass `context.Context` through all async operations
- **Resource Management**: Always `defer` cleanup of connections, files, and resources

### Testing Requirements
- **Unit Tests**: Required for all new functions and methods
- **Integration Tests**: Required for complex features involving multiple components
- **Error Scenarios**: Test error conditions and recovery mechanisms
- **Performance Tests**: Include benchmarks for audio processing code
- **Test Structure**: Use `testify/assert` and `testify/mock` for consistency

### TTS System Specific Rules
- **Audio Processing**: Use native Opus encoding via `layeh.com/gopus`
- **Discord Integration**: Follow DCA format for voice streaming
- **Configuration**: Store guild settings in JSON files under `data/`
- **Voice Management**: Handle voice connection lifecycle in `voice_manager.go`
- **Message Processing**: Use the queue system in `message_queue.go` for async processing

## Commit Message Format
**Must follow Conventional Commits specification:**
```
<type>[scope]: <description>

[optional body]
[optional footer]
```

### Version Impact
- `feat:` → Minor version bump (1.0.0 → 1.1.0)
- `fix:` → Patch version bump (1.1.0 → 1.1.1)  
- `feat!:` or `BREAKING CHANGE:` → Major version bump (1.1.1 → 2.0.0)
- `refactor:`, `docs:`, `test:`, `ci:` → No version change

## CI/CD Integration
- **GitHub Actions**: All workflows must pass before merge
- **Automated Testing**: Includes unit, integration, and performance tests
- **Semantic Releases**: Version bumping based on conventional commits
- **Linux Builds**: Verify compatibility and deployment on Linux

## Quality Gates
1. **All tests pass** with maintained coverage
2. **No linting errors** from `go vet` and `golangci-lint`
3. **Conventional commit format** for automated releases
4. **Documentation updated** for user-facing changes
5. **No sensitive data** committed to repository