package main

import (
	"fmt"
	"strings"
)

// convertConfig converts CircleCI config to orchestration-only config + Taskfile
func convertConfig(config CircleCIConfig) (CircleCIConfig, Taskfile) {
	newConfig := CircleCIConfig{
		Version:   config.Version,
		Jobs:      make(map[string]Job),
		Commands:  nil, // Remove commands from new config - they become tasks
		Workflows: config.Workflows,
		Executors: config.Executors,
	}

	taskfile := Taskfile{
		Version: "3",
		Tasks:   make(map[string]Task),
	}

	// Extract common patterns and deduplicate
	patterns := analyzePatterns(config)
	
	// Convert CircleCI commands to tasks
	commandTasks := convertCommandsToTasks(config.Commands)
	for name, task := range commandTasks {
		taskfile.Tasks[name] = task
	}
	
	// Convert each job
	for jobName, job := range config.Jobs {
		// Create task from job steps
		task := convertJobToTask(jobName, job, patterns, config.Commands)
		taskfile.Tasks[jobName] = task

		// Create minimal CircleCI job that just calls the task
		newJob := Job{
			Executor: job.Executor,
			Docker:   job.Docker,
			Machine:  job.Machine,
			Steps: []Step{
				map[string]interface{}{"run": fmt.Sprintf("task %s", jobName)},
			},
		}
		newConfig.Jobs[jobName] = newJob
	}

	// Add common pattern tasks
	for name, task := range patterns {
		taskfile.Tasks[name] = task
	}

	// Add local development helpers
	addLocalDevTasks(&taskfile)

	return newConfig, taskfile
}

// convertJobToTask converts a CircleCI job to a go-task Task  
func convertJobToTask(jobName string, job Job, patterns map[string]Task, commands map[string]Command) Task {
	var cmds []string
	var deps []string
	var workingDir string

	// Extract working directory if specified
	if job.Environment != nil {
		// Could extract WORKDIR or similar env vars
	}

	for _, step := range job.Steps {
		if cmd := extractCommand(step); cmd != "" {
			// Check if this command matches a common pattern
			normalized := normalizeCommand(cmd)
			if taskName := findPatternTask(normalized, patterns); taskName != "" {
				deps = append(deps, taskName)
			} else {
				cmds = append(cmds, cmd)
			}
		} else if stepStr, ok := step.(string); ok {
			// Check if this string step is a command invocation
			if _, isCommandDefined := commands[stepStr]; isCommandDefined {
				deps = append(deps, stepStr)
			} else {
				// Handle built-in steps like "checkout"
				converted := convertStepToCommand(step)
				if !strings.Contains(converted, "Skipping") && !strings.Contains(converted, "task ") {
					cmds = append(cmds, converted)
				} else {
					cmds = append(cmds, fmt.Sprintf("# %s", converted))
				}
			}
		} else if commandName, isCommand := isCommandInvocation(step); isCommand {
			// This step invokes a CircleCI command with parameters, add it as a task dependency
			deps = append(deps, commandName)
		} else {
			// Handle other step types (checkout, etc.)
			converted := convertStepToCommand(step)
			if !strings.Contains(converted, "Skipping") {
				cmds = append(cmds, converted)
			} else {
				// Add as comment for visibility
				cmds = append(cmds, fmt.Sprintf("# %s", converted))
			}
		}
	}

	task := Task{
		Desc:   fmt.Sprintf("Task converted from CircleCI job: %s", jobName),
		Cmds:   cmds,
		Deps:   deps,
		Silent: false,
	}

	if workingDir != "" {
		task.Dir = workingDir
	}

	return task
}

// addLocalDevTasks adds helpful local development tasks
func addLocalDevTasks(taskfile *Taskfile) {
	// Clean up local artifacts
	taskfile.Tasks["clean"] = Task{
		Desc: "Clean local build artifacts",
		Cmds: []string{
			"rm -rf ./workspace ./artifacts ./test-results",
			"echo 'Cleaned local CircleCI simulation directories'",
		},
	}

	// Setup local environment to mimic CircleCI
	taskfile.Tasks["setup-local"] = Task{
		Desc: "Setup local environment for CircleCI simulation",
		Cmds: []string{
			"mkdir -p ./workspace ./artifacts ./test-results",
			"echo 'Local CircleCI directories created'",
			"echo 'Note: Some steps are CircleCI-server only and will be skipped'",
		},
	}

	// Run all jobs in dependency order (simulate full CI)
	taskfile.Tasks["ci-local"] = Task{
		Desc: "Run full CI pipeline locally (where possible)",
		Deps: []string{"setup-local"},
		Cmds: []string{
			"echo 'Running local CI simulation...'",
			"echo 'Note: This runs the build logic, but skips server-only features'",
		},
	}
}

// convertCommandsToTasks converts CircleCI commands to go-task tasks
func convertCommandsToTasks(commands map[string]Command) map[string]Task {
	tasks := make(map[string]Task)
	
	for commandName, command := range commands {
		var cmds []string
		
		for _, step := range command.Steps {
			if cmd := extractCommand(step); cmd != "" {
				cmds = append(cmds, cmd)
			} else {
				// Handle other step types
				converted := convertStepToCommand(step)
				if !strings.Contains(converted, "Skipping") {
					cmds = append(cmds, converted)
				} else {
					cmds = append(cmds, fmt.Sprintf("# %s", converted))
				}
			}
		}
		
		desc := command.Description
		if desc == "" {
			desc = fmt.Sprintf("Task converted from CircleCI command: %s", commandName)
		}
		
		tasks[commandName] = Task{
			Desc:   desc,
			Cmds:   cmds,
			Silent: false,
		}
	}
	
	return tasks
}

// Helper to get job dependencies from workflow
func getJobDependencies(jobName string, workflow Workflow) []string {
	var deps []string
	// TODO: Parse workflow dependencies if needed
	return deps
}
