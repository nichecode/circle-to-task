# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Circle-to-Task Converter is a Go CLI tool that converts CircleCI configurations into orchestration-only configs that call go-task, enabling local development and testing of CI/CD pipelines.

The tool converts CircleCI configs into two files:
- **New CircleCI config**: Jobs become simple `task <job-name>` calls  
- **Taskfile.yml**: Contains all the actual build logic for local execution

## Build & Development Commands

```bash
# Build the binary
task build

# Test with example config
task test

# Build and test together
task dev

# Clean build artifacts
task clean

# Show all available tasks
task help
```

For manual builds without go-task:
```bash
go build -o circle-to-task .
```

## Code Architecture

The codebase is organized into logical modules:

- **main.go**: CLI entry point, argument parsing, file I/O orchestration
- **types.go**: Type definitions for CircleCI configs and Taskfile structures
- **converter.go**: Core conversion logic from CircleCI jobs to go-task tasks
- **steps.go**: Step-specific conversion logic (checkout, run, persist_to_workspace, etc.)
- **patterns.go**: Pattern analysis and deduplication of common command sequences

### Key Components

**CircleCIConfig** and **Job** types handle parsing CircleCI YAML structures including executors, docker images, and step definitions.

**Taskfile** and **Task** types represent the go-task YAML structure with commands, dependencies, and descriptions.

**convertConfig()** is the main orchestrator that:
1. Analyzes patterns across jobs to deduplicate common commands
2. Converts CircleCI commands to reusable tasks
3. Transforms each job into a task with proper dependencies
4. Creates minimal CircleCI jobs that just call `task <job-name>`
5. Adds local development helper tasks (setup-local, clean, ci-local)

**Step conversion logic** handles different CircleCI step types:
- `checkout` → `git checkout HEAD`
- `run` commands → executed as-is
- `persist_to_workspace` → copied to `./workspace/` for local simulation
- `save_cache`/`restore_cache` → commented out (server-only)

## Testing

Use the examples/ directory to test conversions:
- `examples/input-config.yml` - sample CircleCI config
- `examples/input-with-commands.yml` - config with reusable commands
- `examples/output/` - generated files after conversion

Run `task test` to verify the converter works with the example configs.

## Dependencies

- Go 1.21+
- gopkg.in/yaml.v3 for YAML processing
- go-task for running the generated Taskfiles locally