# Implementation Plan

- [x] 1. Add dependencies and update project structure





  - Add Cobra and Viper dependencies to go.mod
  - Create new command structure files in cmd/darrot/
  - Update imports and module structure
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2. Implement core Cobra CLI structure





  - [x] 2.1 Create root command with global flags and help

    - Implement main root command with global flags (--config, --log-level, --help)
    - Set up command hierarchy and basic help text
    - _Requirements: 1.1, 1.4_

  - [x] 2.2 Implement version command

    - Create version subcommand that displays structured version information
    - Support both human-readable and JSON output formats
    - _Requirements: 1.2_

  - [x] 2.3 Create start command structure

    - Implement start subcommand that replaces current main functionality
    - Add all current configuration options as CLI flags
    - _Requirements: 1.3, 1.4_

- [x] 3. Integrate Viper configuration management













  - [x] 3.1 Create Viper-based configuration loader




    - Implement ConfigManager struct with Viper integration
    - Set up configuration file search paths and format support
    - _Requirements: 2.1, 2.4, 2.5_



  - [x] 3.2 Implement configuration precedence system


    - Set up precedence order: CLI flags > environment variables > config file > defaults
    - Configure Viper's AutomaticEnv with "DRT" prefix for environment variable support


    - _Requirements: 2.2, 2.3, 2.6_

  - [x] 3.3 Add default values for all configuration options


    - Define sensible defaults for all non-sensitive configuration options
    - Maintain existing default values from current implementation

    - _Requirements: 2.3, 4.4, 4.5_

- [x] 4. Implement config command group

  - [x] 4.1 Create config validate subcommand






    - Implement configuration validation without starting the bot
    - Provide clear error messages for invalid configurations
    - _Requirements: 3.1, 3.6_

  - [x] 4.2 Create config show subcommand








    - Display effective configuration with source information
    - Support multiple output formats (human-readable, JSON)
    - Mask sensitive values like tokens
    - _Requirements: 3.2, 3.3, 3.7_

  - [x] 4.3 Create config create subcommand





    - Save current effective configuration to file
    - Support custom output locations via --output flag
    - Generate properly formatted YAML configuration files
    - _Requirements: 3.4, 3.5_

- [x] 5. Enhance error handling and user experience





  - [x] 5.1 Implement comprehensive CLI error handling


    - Add helpful error messages with suggestions for invalid commands
    - Implement command and flag suggestion system
    - _Requirements: 1.5, 5.4, 5.5_

  - [x] 5.2 Add shell completion support


    - Generate bash, zsh, and PowerShell completion scripts
    - Implement completion commands for all shells
    - _Requirements: 5.1, 5.2, 5.3_

- [x] 6. Update main application integration





  - [x] 6.1 Refactor main.go to use Cobra commands


    - Replace current flag parsing with Cobra command execution
    - Maintain all existing functionality through start command
    - _Requirements: 1.3, 4.3_



  - [ ] 6.2 Update configuration loading in bot initialization
    - Modify bot initialization to use new Viper-based configuration
    - Update configuration validation to work with new structure
    - _Requirements: 2.7_

- [ ] 7. Add comprehensive testing
  - [ ]* 7.1 Write unit tests for CLI commands
    - Test all command parsing and flag handling
    - Test help text and error message generation
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [ ]* 7.2 Write configuration management tests
    - Test configuration loading from all sources
    - Test precedence rules and validation logic
    - _Requirements: 2.1, 2.2, 2.3, 2.6_

  - [ ]* 7.3 Write config command tests
    - Test validate, show, and create subcommands
    - Test output formatting and error handling
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

  - [ ]* 7.4 Write environment variable tests
    - Test new DRT_ prefixed environment variable configurations
    - Test environment variable precedence and mapping
    - _Requirements: 2.2, 2.6_

- [x] 8. Update documentation and examples





  - [x] 8.1 Update README with new CLI usage


    - Document all new commands and configuration options
    - Provide migration guide for new DRT_ environment variable prefix
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 8.2 Create example configuration files


    - Provide sample YAML, JSON, and TOML configuration files
    - Document all configuration options with examples
    - _Requirements: 2.1, 3.4, 3.5_