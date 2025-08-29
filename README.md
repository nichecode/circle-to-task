# Circle-to-Task Converter

Convert CircleCI configurations to orchestration-only configs that call go-task, enabling local development and testing of your CI/CD pipeline.

## What it does

🔄 **Converts your CircleCI config** into two files:
- **New CircleCI config**: Jobs become simple `task <job-name>` calls
- **Taskfile.yml**: Contains all your actual build logic

🏠 **Enables local development**: Run any CI job locally with `task <job-name>`

🎯 **Smart step handling**:
- ✅ **Local-runnable**: `run`, `checkout`, build commands
- ⚠️ **Simulated**: `persist_to_workspace` → copies to `./workspace`  
- ❌ **Server-only**: `save_cache`, `setup_remote_docker` (appropriately skipped)

🔧 **Deduplicates common patterns**: Extracts repeated commands into reusable tasks

## Installation

```bash
# Clone and build
git clone https://github.com/nichecode/circle-to-task.git
cd circle-to-task
go build -o circle-to-task main.go

# Or install directly
go install github.com/nichecode/circle-to-task@latest
```

## Usage

```bash
# Basic conversion
./circle-to-task -input .circleci/config.yml -output ./converted

# Convert to current directory  
./circle-to-task -input config.yml

# Show help
./circle-to-task -help
```

## Example

### Before (CircleCI config.yml):
```yaml
version: 2.1
jobs:
  build:
    docker:
      - image: node:16
    steps:
      - checkout
      - run: npm install
      - run: npm run build
      - store_artifacts:
          path: dist/
  test:
    docker:
      - image: node:16  
    steps:
      - checkout
      - run: npm install
      - run: npm test
workflows:
  build-test:
    jobs: [build, test]
```

### After conversion:

**config.yml** (CircleCI orchestration only):
```yaml
version: 2.1
jobs:
  build:
    docker:
      - image: node:16
    steps:
      - run: task build
  test:
    docker:
      - image: node:16
    steps:
      - run: task test
workflows:
  build-test:
    jobs: [build, test]
```

**Taskfile.yml** (actual build logic):
```yaml
version: '3'
tasks:
  npm-install:
    desc: "Common task - used in 2 jobs"
    cmds:
      - npm install
  
  build:
    desc: "Task converted from CircleCI job: build"
    deps: [npm-install]
    cmds:
      - git checkout HEAD
      - npm run build
      - mkdir -p ./artifacts && cp -r dist/ ./artifacts/
  
  test:
    desc: "Task converted from CircleCI job: test"
    deps: [npm-install]
    cmds:
      - git checkout HEAD
      - npm test

  setup-local:
    desc: "Setup local environment for CircleCI simulation"
    cmds:
      - mkdir -p ./workspace ./artifacts ./test-results
      - echo 'Local CircleCI directories created'

  clean:
    desc: "Clean local build artifacts"
    cmds:
      - rm -rf ./workspace ./artifacts ./test-results
```

## Local Development Workflow

After conversion:

```bash
# Setup local environment
task setup-local

# Run individual jobs locally
task build    # Just the build logic
task test     # Just the test logic  

# Clean up
task clean

# List all available tasks
task --list
```

## Step Conversion Reference

| CircleCI Step | Local Equivalent | Notes |
|---------------|------------------|-------|
| `checkout` | `git checkout HEAD` | Gets current branch |
| `run: <cmd>` | `<cmd>` | Executed as-is |
| `persist_to_workspace` | `cp files ./workspace/` | Local simulation |
| `store_artifacts` | `cp files ./artifacts/` | Local simulation |
| `store_test_results` | `cp files ./test-results/` | Local simulation |
| `save_cache` | `# Skipped (server only)` | Commented out |
| `restore_cache` | `# Skipped (server only)` | Commented out |
| `setup_remote_docker` | `# Skipped (server only)` | Commented out |

## Migration Strategy

1. **Convert existing config**: Generate both files side-by-side
2. **Test locally**: Verify tasks work with `task <job-name>`
3. **Gradual rollout**: Use feature flags or parallel workflows
4. **Compare outputs**: Ensure local matches CI behavior

## Requirements

- [go-task](https://taskfile.dev/) installed locally
- Go 1.19+ for building from source
- Git for checkout operations

## Benefits

- 🏃‍♂️ **Faster development**: Test CI logic without pushing
- 🐛 **Easier debugging**: Run individual steps in isolation  
- 🔄 **Consistent environments**: Same commands locally and in CI
- ♻️ **Reusable patterns**: Common commands become shared tasks
- 🎯 **Focused CI**: CircleCI handles orchestration, go-task handles execution

## Contributing

Issues and PRs welcome! This tool helps bridge the gap between local development and CI/CD environments.

## License

MIT License - see LICENSE file for details.
