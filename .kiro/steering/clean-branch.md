---
inclusion: always
---

# Clean Branch Management

## Branch Naming Convention
Use descriptive, kebab-case branch names that clearly indicate the purpose:

- **Feature branches**: `feat/feature-name` (e.g., `feat/voice-speed-control`)
- **Bug fixes**: `fix/issue-description` (e.g., `fix/audio-stuttering-arm64`)
- **Refactoring**: `refactor/component-name` (e.g., `refactor/tts-processor`)
- **Documentation**: `docs/topic` (e.g., `docs/installation-guide`)
- **CI/CD changes**: `ci/workflow-name` (e.g., `ci/automated-releases`)
- **Performance**: `perf/optimization-area` (e.g., `perf/audio-encoding`)

## Pre-Commit Checklist
Before committing any changes, ensure:

- [ ] **Tests pass**: Run `go test ./...` to verify all tests pass
- [ ] **Code formatted**: Run `go fmt ./...` to format code consistently
- [ ] **Linting clean**: Run `go vet ./...` to catch potential issues
- [ ] **Dependencies tidy**: Run `go mod tidy` to clean up module dependencies
- [ ] **Conventional commits**: Follow the conventional commits format for all commit messages
- [ ] **No sensitive data**: Verify no tokens, keys, or credentials are committed

## File Management Rules

### Never Commit These Files
- `.env` - Contains sensitive Discord tokens and API keys
- `data/` directory contents - Runtime user data and configurations
- Compiled binaries - Use releases instead of committing built executables
- `coverage.out`, `coverage.html` - Test coverage reports
- IDE-specific files outside `.vscode/` directory

### Always Include
- `.env.example` - Template for environment configuration
- `go.mod` and `go.sum` - Dependency management files
- Test files (`*_test.go`) - Comprehensive test coverage
- Documentation updates for new features

## Code Quality Standards

### Go-Specific Requirements
- **Package structure**: Follow the established `internal/` package organization
- **Error handling**: Use the centralized error recovery system in `internal/tts/error_recovery.go`
- **Interfaces**: Define interfaces for testability and modularity
- **Context usage**: Pass context through all async operations
- **Resource cleanup**: Always defer cleanup of resources (connections, files)

### Testing Requirements
- **Unit tests**: All new functions must have corresponding unit tests
- **Integration tests**: Complex features require integration test coverage
- **Error scenarios**: Test error conditions and recovery mechanisms
- **Performance tests**: Include performance benchmarks for audio processing code

## Branch Lifecycle

### Before Creating a Branch
1. **Sync with main**: `git checkout main && git pull origin main`
2. **Create from main**: Always branch from the latest main branch
3. **Clear purpose**: Ensure the branch has a single, well-defined purpose

### During Development
- **Atomic commits**: Each commit should represent a single logical change
- **Frequent commits**: Commit early and often with meaningful messages
- **Regular sync**: Periodically rebase against main to avoid conflicts
- **Clean history**: Use interactive rebase to clean up commit history before PR

### Before Pull Request
- **Rebase on main**: `git rebase main` to ensure clean integration
- **Squash if needed**: Combine related commits for cleaner history
- **Final testing**: Run full test suite including integration tests
- **Documentation**: Update README or docs if functionality changed

## Merge Strategy

### Pull Request Requirements
- **Conventional commits**: All commits must follow conventional commit format
- **Passing CI**: All GitHub Actions workflows must pass
- **Code review**: At least one approval from a maintainer
- **No merge conflicts**: Branch must be up-to-date with main
- **Documentation**: Include relevant documentation updates

### Merge Types
- **Squash merge**: Preferred for feature branches to maintain clean history
- **Merge commit**: Use for release branches or significant milestones
- **Rebase merge**: Use for small, clean commits that add value to history

## Cleanup Process

### After Merge
1. **Delete branch**: Remove both local and remote feature branches
2. **Update main**: Pull latest changes to local main branch
3. **Clean references**: Run `git remote prune origin` to clean stale references

### Regular Maintenance
- **Weekly cleanup**: Remove merged branches and update dependencies
- **Dependency updates**: Review and merge Dependabot PRs promptly
- **Release preparation**: Ensure main branch is always in releasable state

## Emergency Procedures

### Hotfix Process
1. **Create hotfix branch**: Branch from main with `hotfix/` prefix
2. **Minimal changes**: Keep changes focused on the critical issue
3. **Fast-track review**: Expedited review process for critical fixes
4. **Immediate release**: Deploy hotfix as soon as it's merged

### Rollback Strategy
- **Revert commits**: Use `git revert` for safe rollbacks
- **Emergency releases**: Have rollback release process ready
- **Communication**: Notify team immediately of any rollbacks

## Integration with CI/CD

### Automated Checks
- **Build verification**: Linux platform builds
- **Test execution**: Full test suite with coverage reporting
- **Linting**: Automated code quality checks
- **Security scanning**: Dependency vulnerability checks

### Release Automation
- **Semantic versioning**: Automatic version bumping based on conventional commits
- **Changelog generation**: Automated changelog from commit messages
- **Asset building**: Automated binary builds for Linux
- **Deployment**: Automated deployment to production environments