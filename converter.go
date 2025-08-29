package main

import (
	"fmt"
	"os"
	"regexp"
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
		// If the job has parameters, we need to handle them in the workflow invocations
		taskCall := fmt.Sprintf("task %s", jobName)
		
		newJob := Job{
			Executor:   job.Executor,
			Docker:     job.Docker,
			Machine:    job.Machine,
			Parameters: job.Parameters, // Keep parameters for workflow invocations
			Steps: []Step{
				map[string]interface{}{"run": taskCall},
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

	// Add environment variable defaults for local development
	addLocalEnvDefaults(&taskfile, config)

	return newConfig, taskfile
}

// CommandInfo holds information about a command including usage count
type CommandInfo struct {
	Command string
	Count   int
}

// extractAllCommands extracts all commands from the CircleCI config with usage counts
func extractAllCommands(config CircleCIConfig) []CommandInfo {
	commandCounts := make(map[string]int)
	
	// Extract from jobs
	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			if cmd := extractCommand(step); cmd != "" {
				subCommands := extractIndividualCommands(cmd)
				for _, subCmd := range subCommands {
					cleanCmd := cleanCommandForAnalysis(subCmd)
					if cleanCmd != "" {
						commandCounts[cleanCmd]++
					}
				}
			}
		}
	}
	
	// Extract from commands
	for _, command := range config.Commands {
		for _, step := range command.Steps {
			if cmd := extractCommand(step); cmd != "" {
				subCommands := extractIndividualCommands(cmd)
				for _, subCmd := range subCommands {
					cleanCmd := cleanCommandForAnalysis(subCmd)
					if cleanCmd != "" {
						commandCounts[cleanCmd]++
					}
				}
			}
		}
	}
	
	// Convert map to sorted slice
	var commands []CommandInfo
	for cmd, count := range commandCounts {
		commands = append(commands, CommandInfo{Command: cmd, Count: count})
	}
	
	// Sort by count (descending) then by command name
	for i := 0; i < len(commands); i++ {
		for j := i + 1; j < len(commands); j++ {
			if commands[i].Count < commands[j].Count || 
			   (commands[i].Count == commands[j].Count && commands[i].Command > commands[j].Command) {
				commands[i], commands[j] = commands[j], commands[i]
			}
		}
	}
	
	return commands
}

// extractIndividualCommands splits multi-line commands into individual command lines
func extractIndividualCommands(cmd string) []string {
	var commands []string
	
	// Split by newlines and also by && operators
	lines := strings.Split(cmd, "\n")
	
	for _, line := range lines {
		// Split by && to get individual commands on same line
		parts := strings.Split(line, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" && !strings.HasPrefix(part, "#") { // Skip empty lines and comments
				commands = append(commands, part)
			}
		}
	}
	
	return commands
}

// cleanCommandForAnalysis cleans up commands for technology analysis
func cleanCommandForAnalysis(cmd string) string {
	// Remove parameter syntax and variables for cleaner analysis
	cleaned := convertParameterSyntax(cmd)
	
	// Remove environment variables for cleaner output
	envRegex := regexp.MustCompile(`\$[A-Z_][A-Z0-9_]*|\$\{[A-Z_][A-Z0-9_]*\}`)
	cleaned = envRegex.ReplaceAllString(cleaned, "${VAR}")
	
	// Normalize whitespace but preserve line breaks for multi-line commands
	cleaned = strings.TrimSpace(cleaned)
	
	// Skip empty or very short commands
	if len(cleaned) < 3 {
		return ""
	}
	
	return cleaned
}

// generateTechnologyAnalysis creates a markdown file with all commands for AI analysis
func generateTechnologyAnalysis(config CircleCIConfig, outputDir string) error {
	commands := extractAllCommands(config)
	
	if len(commands) == 0 {
		return nil // No commands to analyze
	}
	
	var content strings.Builder
	
	content.WriteString("# Technology Analysis Report\n\n")
	content.WriteString("This file contains all commands extracted from the CircleCI configuration for technology categorization.\n\n")
	content.WriteString("## Instructions for AI Analysis\n\n")
	content.WriteString("Please categorize these commands by technology/tool type. Commands are sorted by usage frequency (most used first).\n\n")
	content.WriteString("Suggested categories:\n")
	content.WriteString("- **Package Managers**: npm, yarn, pip, composer, etc.\n")
	content.WriteString("- **Build Tools**: webpack, gulp, maven, gradle, etc.\n")
	content.WriteString("- **Testing**: jest, pytest, phpunit, go test, etc.\n")
	content.WriteString("- **Cloud/Infrastructure**: aws, gcloud, kubectl, terraform, etc.\n")
	content.WriteString("- **Containers**: docker, podman, etc.\n")
	content.WriteString("- **Languages**: node, python, php, go, java, etc.\n")
	content.WriteString("- **Databases**: mysql, postgres, redis, etc.\n")
	content.WriteString("- **Other Tools**: git, curl, ssh, etc.\n\n")
	
	// Calculate total usage
	totalUsage := 0
	for _, cmd := range commands {
		totalUsage += cmd.Count
	}
	
	content.WriteString(fmt.Sprintf("## All Commands (%d unique commands, %d total usages)\n\n", len(commands), totalUsage))
	
	for i, cmd := range commands {
		percentage := float64(cmd.Count) / float64(totalUsage) * 100
		content.WriteString(fmt.Sprintf("%d. `%s` **(used %d times, %.1f%%)**\n", i+1, cmd.Command, cmd.Count, percentage))
	}
	
	content.WriteString("\n")
	content.WriteString("## Usage Summary\n\n")
	content.WriteString("Commands ordered by frequency can help prioritize which technologies are most important in this configuration.\n\n")
	
	content.WriteString("## Technology Categories\n\n")
	content.WriteString("*Please fill in this section after AI analysis*\n\n")
	content.WriteString("### Package Managers\n- \n\n")
	content.WriteString("### Build Tools\n- \n\n") 
	content.WriteString("### Testing Frameworks\n- \n\n")
	content.WriteString("### Cloud/Infrastructure\n- \n\n")
	content.WriteString("### Container Tools\n- \n\n")
	content.WriteString("### Programming Languages\n- \n\n")
	content.WriteString("### Databases\n- \n\n")
	content.WriteString("### Other Tools\n- \n\n")
	
	analysisPath := fmt.Sprintf("%s/TECHNOLOGY_ANALYSIS.md", outputDir)
	return writeTextFile(analysisPath, content.String())
}

// writeTextFile writes a text file to the filesystem
func writeTextFile(path string, content string) error {
	return writeFileContent(path, []byte(content))
}

// writeFileContent writes content to a file
func writeFileContent(path string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()
	
	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("error writing content: %w", err)
	}
	
	return nil
}

// convertParameterSyntax converts CircleCI parameter syntax to go-task variable syntax
func convertParameterSyntax(cmd string) string {
	// Convert << parameters.name >> to {{.NAME}}
	result := cmd
	// Find all << parameters.xxx >> patterns and convert them
	for {
		start := strings.Index(result, "<< parameters.")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], " >>")
		if end == -1 {
			break
		}
		end += start
		
		// Extract parameter name
		paramPart := result[start+14:end] // Skip "<< parameters."
		paramName := strings.ToUpper(paramPart)
		
		// Replace with go-task syntax
		result = result[:start] + "{{." + paramName + "}}" + result[end+3:]
	}
	
	return result
}

// convertJobToTask converts a CircleCI job to a go-task Task  
func convertJobToTask(jobName string, job Job, patterns map[string]Task, commands map[string]Command) Task {
	var cmds []string
	var deps []string
	var workingDir string
	vars := make(map[string]string)

	// Convert job parameters to go-task variables
	if job.Parameters != nil {
		for paramName, paramDef := range job.Parameters {
			if paramMap, ok := paramDef.(map[string]interface{}); ok {
				defaultValue := ""
				if defVal, hasDefault := paramMap["default"]; hasDefault {
					defaultValue = fmt.Sprintf("%v", defVal)
				}
				vars[strings.ToUpper(paramName)] = fmt.Sprintf("{{.%s | default \"%s\"}}", strings.ToUpper(paramName), defaultValue)
			}
		}
	}

	// Extract working directory if specified
	if job.Environment != nil {
		// Could extract WORKDIR or similar env vars
	}

	for _, step := range job.Steps {
		if cmd := extractCommand(step); cmd != "" {
			// Convert parameter syntax in commands
			convertedCmd := convertParameterSyntax(cmd)
			// Check if this command matches a common pattern
			normalized := normalizeCommand(convertedCmd)
			if taskName := findPatternTask(normalized, patterns); taskName != "" {
				deps = append(deps, taskName)
			} else {
				cmds = append(cmds, convertedCmd)
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
			// This step invokes a CircleCI command with parameters
			taskCall := generateTaskCallWithParams(commandName, step, commands)
			cmds = append(cmds, taskCall)
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

	if len(vars) > 0 {
		task.Vars = vars
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
		vars := make(map[string]string)
		
		// Convert CircleCI parameters to go-task variables with defaults
		if command.Parameters != nil {
			for paramName, paramDef := range command.Parameters {
				if paramMap, ok := paramDef.(map[string]interface{}); ok {
					defaultValue := ""
					if defVal, hasDefault := paramMap["default"]; hasDefault {
						defaultValue = fmt.Sprintf("%v", defVal)
					}
					vars[strings.ToUpper(paramName)] = fmt.Sprintf("{{.%s | default \"%s\"}}", strings.ToUpper(paramName), defaultValue)
				}
			}
		}
		
		for _, step := range command.Steps {
			if cmd := extractCommand(step); cmd != "" {
				// Replace CircleCI parameter syntax with go-task variable syntax
				convertedCmd := convertParameterSyntax(cmd)
				cmds = append(cmds, convertedCmd)
			} else {
				// Handle other step types
				converted := convertStepToCommand(step)
				if !strings.Contains(converted, "Skipping") {
					convertedCmd := convertParameterSyntax(converted)
					cmds = append(cmds, convertedCmd)
				} else {
					cmds = append(cmds, fmt.Sprintf("# %s", converted))
				}
			}
		}
		
		desc := command.Description
		if desc == "" {
			desc = fmt.Sprintf("Task converted from CircleCI command: %s", commandName)
		}
		
		task := Task{
			Desc:   desc,
			Cmds:   cmds,
			Silent: false,
		}
		
		if len(vars) > 0 {
			task.Vars = vars
		}
		
		tasks[commandName] = task
	}
	
	return tasks
}

// generateTaskCallWithParams generates a go-task call with parameters
func generateTaskCallWithParams(commandName string, step Step, commands map[string]Command) string {
	stepMap, ok := step.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("task %s", commandName)
	}
	
	commandParams, ok := stepMap[commandName]
	if !ok {
		return fmt.Sprintf("task %s", commandName)
	}
	
	paramMap, ok := commandParams.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("task %s", commandName)
	}
	
	var paramPairs []string
	for paramName, paramValue := range paramMap {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%v", strings.ToUpper(paramName), paramValue))
	}
	
	if len(paramPairs) > 0 {
		return fmt.Sprintf("task %s %s", commandName, strings.Join(paramPairs, " "))
	}
	
	return fmt.Sprintf("task %s", commandName)
}

// addLocalEnvDefaults adds environment variable defaults for local development
func addLocalEnvDefaults(taskfile *Taskfile, config CircleCIConfig) {
	envVars := make(map[string]string)
	
	// Collect all environment variables used in the config
	envVarsUsed := extractEnvironmentVariables(config)
	
	// Add defaults for common CircleCI environment variables
	circleCIDefaults := map[string]string{
		"CIRCLE_PROJECT_REPONAME":     "local-repo",
		"CIRCLE_PROJECT_USERNAME":     "local-user", 
		"CIRCLE_BRANCH":               "main",
		"CIRCLE_BUILD_NUM":            "1",
		"CIRCLE_SHA1":                 "local-sha",
		"CIRCLE_WORKING_DIRECTORY":    ".",
		"CIRCLE_TEST_REPORTS":         "./test-results",
		"HOME":                        "$HOME",
		"PWD":                         "$PWD",
		"NODE_ENV":                    "development",
		"AWS_DEFAULT_REGION":          "us-east-1",
	}
	
	// Only add defaults for env vars that are actually used
	for envVar := range envVarsUsed {
		if defaultValue, hasDefault := circleCIDefaults[envVar]; hasDefault {
			envVars[envVar] = defaultValue
		} else {
			// Add a placeholder for unknown env vars
			envVars[envVar] = fmt.Sprintf("# TODO: Set %s for local development", envVar)
		}
	}
	
	if len(envVars) > 0 {
		taskfile.Env = envVars
	}
}

// extractEnvironmentVariables finds all environment variables used in the config
func extractEnvironmentVariables(config CircleCIConfig) map[string]bool {
	envVars := make(map[string]bool)
	envRegex := regexp.MustCompile(`\$([A-Z_][A-Z0-9_]*)\b|\$\{([A-Z_][A-Z0-9_]*)\}`)
	
	// Check all jobs
	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			if cmd := extractCommand(step); cmd != "" {
				matches := envRegex.FindAllStringSubmatch(cmd, -1)
				for _, match := range matches {
					if match[1] != "" {
						envVars[match[1]] = true
					}
					if match[2] != "" {
						envVars[match[2]] = true
					}
				}
			}
		}
	}
	
	// Check all commands
	for _, command := range config.Commands {
		for _, step := range command.Steps {
			if cmd := extractCommand(step); cmd != "" {
				matches := envRegex.FindAllStringSubmatch(cmd, -1)
				for _, match := range matches {
					if match[1] != "" {
						envVars[match[1]] = true
					}
					if match[2] != "" {
						envVars[match[2]] = true
					}
				}
			}
		}
	}
	
	return envVars
}

// Helper to get job dependencies from workflow
func getJobDependencies(jobName string, workflow Workflow) []string {
	var deps []string
	// TODO: Parse workflow dependencies if needed
	return deps
}
