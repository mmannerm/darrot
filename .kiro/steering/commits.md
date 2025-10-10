# Conventional Commits Standard

## Mandatory Requirement
**ALL commit messages in this project MUST follow the Conventional Commits specification.** This is not optional - it's required for the automated semantic versioning and release system to function properly.

## Commit Message Format
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Required Types
Use these commit types for proper semantic versioning:

### Version Bumping Types
- **feat**: A new feature (triggers MINOR version bump)
- **fix**: A bug fix (triggers PATCH version bump)
- **perf**: A performance improvement (triggers PATCH version bump)

### Breaking Changes
- **feat!**: New feature with breaking changes (triggers MAJOR version bump)
- **fix!**: Bug fix with breaking changes (triggers MAJOR version bump)
- **BREAKING CHANGE**: In footer (triggers MAJOR version bump)

### Non-Version Bumping Types
- **refactor**: Code refactoring without functional changes
- **docs**: Documentation changes only
- **test**: Adding or updating tests
- **build**: Changes to build system or dependencies
- **ci**: Changes to CI/CD configuration
- **chore**: Maintenance tasks, dependency updates
- **style**: Code style changes (formatting, whitespace)

## Examples

### Feature Addition (Minor Version Bump)
```
feat: add voice speed control for TTS

Allow users to adjust TTS playback speed from 0.5x to 2.0x
through new /tts-speed command.
```

### Bug Fix (Patch Version Bump)
```
fix: resolve audio stuttering on ARM64 systems

Fixed Opus encoding buffer size calculation that caused
audio artifacts on ARM64 architecture.

Closes #42
```

### Breaking Change (Major Version Bump)
```
feat!: redesign configuration file format

BREAKING CHANGE: Configuration format changed from JSON to YAML.
Users must migrate their .env files to the new format.

Migration guide: docs/migration-v2.md
```

### Refactoring (No Version Bump)
```
refactor: extract TTS processing into separate module

Moved TTS logic from bot package to dedicated tts package
for better separation of concerns.
```

### Documentation (No Version Bump)
```
docs: update installation instructions for ARM64

Added specific steps for installing Opus libraries
on ARM64 Linux distributions.
```

### CI/CD Changes (No Version Bump)
```
ci: add automated dependency updates

Configure Dependabot to create weekly PRs for
Go module updates with automated testing.
```

## Scope Guidelines
Optional scope can specify the area of change:

- **bot**: Discord bot core functionality
- **tts**: Text-to-speech processing
- **config**: Configuration management
- **audio**: Audio processing and encoding
- **ci**: CI/CD workflows
- **deps**: Dependency management

### Examples with Scope
```
feat(tts): add neural voice options
fix(bot): handle reconnection after network timeout
refactor(config): simplify environment variable loading
ci(build): optimize Go module caching
```

## Body and Footer Guidelines

### Body (Optional)
- Explain WHAT and WHY, not HOW
- Use imperative mood ("add feature" not "added feature")
- Wrap at 72 characters per line

### Footer (Optional)
- Reference issues: `Closes #123`, `Fixes #456`
- Breaking changes: `BREAKING CHANGE: description`
- Co-authors: `Co-authored-by: Name <email>`

## Validation
The release system will:
- ✅ Parse commit messages for version bumping
- ✅ Generate changelogs from commit types
- ✅ Create semantic version tags automatically
- ❌ Fail if commits don't follow the format

## Tools and Enforcement

### Local Development
Consider using commitizen for consistent commit messages:
```bash
npm install -g commitizen cz-conventional-changelog
echo '{ "path": "cz-conventional-changelog" }' > ~/.czrc
git cz  # Use instead of git commit
```

### IDE Integration
Most IDEs have Conventional Commits plugins:
- VS Code: "Conventional Commits" extension
- IntelliJ: "Git Commit Template" plugin
- Vim: "vim-conventional-commits" plugin

## Common Mistakes to Avoid

### ❌ Wrong Format
```
Added new feature for TTS
Update documentation
Fixed bug in audio processing
```

### ✅ Correct Format
```
feat: add new TTS voice selection feature
docs: update API documentation
fix: resolve audio processing buffer overflow
```

### ❌ Vague Descriptions
```
feat: improvements
fix: bug fix
refactor: changes
```

### ✅ Clear Descriptions
```
feat: add configurable TTS voice speed control
fix: resolve memory leak in audio buffer management
refactor: extract voice connection logic into separate module
```

## Release Impact

### Commit Types → Version Changes
- `feat:` → 1.0.0 → 1.1.0 (minor)
- `fix:` → 1.1.0 → 1.1.1 (patch)
- `feat!:` → 1.1.1 → 2.0.0 (major)
- `refactor:`, `docs:`, `test:`, `ci:`, `chore:` → No version change

### Changelog Generation
Commits are automatically grouped in changelogs:
- **Features** (feat)
- **Bug Fixes** (fix)
- **Performance Improvements** (perf)
- **Code Refactoring** (refactor)
- **Documentation** (docs)
- **Tests** (test)
- **Build System** (build)
- **Continuous Integration** (ci)
- **Miscellaneous** (chore)

## Enforcement Policy
- **All commits** must follow this format
- **No exceptions** for any commit type
- **PR reviews** will check commit message format
- **Automated releases** depend on proper formatting
- **Squash merges** should maintain conventional format

## Resources
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)

---

**Remember**: Consistent commit messages enable automated releases, clear changelogs, and better project maintenance. Every commit matters for the project's version history!