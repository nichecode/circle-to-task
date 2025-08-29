package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CircleCI structures
type CircleCIConfig struct {
	Version   string                 `yaml:"version"`
	Jobs      map[string]Job         `yaml:"jobs"`
	Commands  map[string]Command     `yaml:"commands,omitempty"`
	Workflows map[string]interface{} `yaml:"workflows"`
	Executors map[string]interface{} `yaml:"executors,omitempty"`
}

type Job struct {
	Executor    interface{}   `yaml:"executor,omitempty"`
	Docker      []DockerImage `yaml:"docker,omitempty"`
	Machine     interface{}   `yaml:"machine,omitempty"`
	Steps       []Step        `yaml:"steps"`
	Environment interface{}   `yaml:"environment,omitempty"`
}

type DockerImage struct {
	Image string `yaml:"image"`
}

type Command struct {
	Description string                 `yaml:"description,omitempty"`
	Parameters  map[string]interface{} `yaml:"parameters,omitempty"`
	Steps       []Step                 `yaml:"steps"`
}

type Step interface{}

type Workflow struct {
	Version interface{}   `yaml:"version,omitempty"`
	Jobs    []interface{} `yaml:"jobs"`
}

type WorkflowJob map[string]WorkflowJobConfig

type WorkflowJobConfig struct {
	Requires []string `yaml:"requires,omitempty"`
}

// Taskfile structures
type Taskfile struct {
	Version string             `yaml:"version"`
	Tasks   map[string]Task    `yaml:"tasks"`
	Vars    map[string]string  `yaml:"vars,omitempty"`
}

type Task struct {
	Desc    string   `yaml:"desc,omitempty"`
	Cmds    []string `yaml:"cmds"`
	Deps    []string `yaml:"deps,omitempty"`
	Dir     string   `yaml:"dir,omitempty"`
	Silent  bool     `yaml:"silent,omitempty"`
}

// Main converter logic
func main() {
	var inputFile = flag.String("input", "", "Input CircleCI config file (required)")
	var outputDir = flag.String("output", ".", "Output directory for generated files")
	var help = flag.Bool("help", false, "Show help message")
	
	flag.Parse()

	if *help || *inputFile == "" {
		fmt.Println("Circle-to-Task Converter")
		fmt.Println("========================")
		fmt.Println()
		fmt.Println("Converts CircleCI config to orchestration-only config + Taskfile")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s -input <circleci-config.yml> -output <output-dir>\n", os.Args[0])
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Printf("  %s -input .circleci/config.yml -output ./converted\n", os.Args[0])
		fmt.Printf("  %s -input config.yml\n", os.Args[0])
		return
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatal("Error creating output directory:", err)
	}
	
	// Read CircleCI config
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatal("Error reading input file:", err)
	}

	var config CircleCIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatal("Error parsing YAML:", err)
	}

	// Convert
	newConfig, taskfile := convertConfig(config)

	// Write new CircleCI config
	newConfigData, err := yaml.Marshal(newConfig)
	if err != nil {
		log.Fatal("Error marshaling new config:", err)
	}
	
	configPath := filepath.Join(*outputDir, "config.yml")
	if err := os.WriteFile(configPath, newConfigData, 0644); err != nil {
		log.Fatal("Error writing new config:", err)
	}

	// Write Taskfile
	taskfileData, err := yaml.Marshal(taskfile)
	if err != nil {
		log.Fatal("Error marshaling taskfile:", err)
	}

	taskfilePath := filepath.Join(*outputDir, "Taskfile.yml")
	if err := os.WriteFile(taskfilePath, taskfileData, 0644); err != nil {
		log.Fatal("Error writing taskfile:", err)
	}

	fmt.Printf("✅ Successfully converted CircleCI config!\n")
	fmt.Printf("📋 Converted %d jobs into tasks\n", len(config.Jobs))
	fmt.Printf("📁 Output files:\n")
	fmt.Printf("   - %s (new CircleCI config)\n", configPath)
	fmt.Printf("   - %s (go-task configuration)\n", taskfilePath)
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   1. Review generated files\n")
	fmt.Printf("   2. Test locally: cd %s && task <job-name>\n", *outputDir)
	fmt.Printf("   3. Install go-task if needed: go install github.com/go-task/task/v3/cmd/task@latest\n")
}

func convertConfig(config CircleCIConfig) (CircleCIConfig, Taskfile) {
	newConfig := CircleCIConfig{
		Version:   config.Version,
		Jobs:      make(map[string]Job),
		Commands:  config.Commands,
		Workflows: config.Workflows,
		Executors: config.Executors,
	}

	taskfile := Taskfile{
		Version: "3",
		Tasks:   make(map[string]Task),
	}

	// Extract common patterns and deduplicate
	patterns := analyzePatterns(config)
	
	// Convert each job
	for jobName, job := range config.Jobs {
		// Create task from job steps
		task := convertJobToTask(jobName, job, patterns)
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

func analyzePatterns(config CircleCIConfig) map[string]Task {
	patterns := make(map[string]Task)
	commandCounts := make(map[string]int)
	
	// Count command occurrences across all jobs
	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			if cmd := extractCommand(step); cmd != "" {
				// Normalize command for pattern matching
				normalized := normalizeCommand(cmd)
				commandCounts[normalized]++
			}
		}
	}

	// Create tasks for common patterns (appears in 2+ jobs)
	for cmd, count := range commandCounts {
		if count >= 2 {
			taskName := generateTaskName(cmd)
			patterns[taskName] = Task{
				Desc: fmt.Sprintf("Common task - used in %d jobs", count),
				Cmds: []string{cmd},
			}
		}
	}

	return patterns
}

func convertJobToTask(jobName string, job Job, patterns map[string]Task) Task {
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

func extractCommand(step Step) string {
	stepMap, ok := step.(map[string]interface{})
	if !ok {
		return ""
	}
	
	if run, ok := stepMap["run"]; ok {
		switch v := run.(type) {
		case string:
			return v
		case map[string]interface{}:
			if command, exists := v["command"]; exists {
				if cmdStr, ok := command.(string); ok {
					return cmdStr
				}
			}
		}
	}
	return ""
}

func normalizeCommand(cmd string) string {
	// Basic normalization - remove extra spaces, etc.
	cmd = strings.TrimSpace(cmd)
	cmd = strings.ReplaceAll(cmd, "\n", " ")
	return strings.Join(strings.Fields(cmd), " ")
}

func generateTaskName(cmd string) string {
	// Generate a task name from command
	words := strings.Fields(cmd)
	if len(words) == 0 {
		return "common-task"
	}
	
	// Take first few meaningful words
	var parts []string
	for i, word := range words {
		if i >= 3 {
			break
		}
		// Skip flags and common words
		if !strings.HasPrefix(word, "-") && !isCommonWord(word) {
			parts = append(parts, word)
		}
	}
	
	if len(parts) == 0 {
		return "common-task"
	}
	
	return strings.Join(parts, "-")
}

func isCommonWord(word string) bool {
	common := map[string]bool{
		"and": true, "or": true, "the": true, "a": true, "an": true,
		"with": true, "for": true, "to": true, "of": true, "in": true,
	}
	return common[strings.ToLower(word)]
}

func findPatternTask(normalized string, patterns map[string]Task) string {
	for taskName, task := range patterns {
		if len(task.Cmds) > 0 && normalizeCommand(task.Cmds[0]) == normalized {
			return taskName
		}
	}
	return ""
}

func convertStepToCommand(step Step) string {
	// Handle string steps (like "checkout")
	if stepStr, ok := step.(string); ok {
		switch stepStr {
		case "checkout":
			return "git checkout HEAD"
		default:
			return fmt.Sprintf("echo 'Step: %s'", stepStr)
		}
	}

	// Handle map steps
	stepMap, ok := step.(map[string]interface{})
	if !ok {
		return "echo 'Unknown step type'"
	}

	for key, value := range stepMap {
		switch key {
		case "checkout":
			return "git checkout HEAD" // Local equivalent
		case "setup_remote_docker":
			return "echo 'Skipping setup_remote_docker (CircleCI server only)'"
		case "save_cache":
			// Create local cache simulation
			if cacheConfig, ok := value.(map[string]interface{}); ok {
				if paths, exists := cacheConfig["paths"]; exists {
					return fmt.Sprintf("# Local cache: would save %v", paths)
				}
			}
			return "echo 'Skipping save_cache (CircleCI server only)'"
		case "restore_cache":
			return "echo 'Skipping restore_cache (CircleCI server only)'"
		case "persist_to_workspace":
			if workspaceConfig, ok := value.(map[string]interface{}); ok {
				if paths, exists := workspaceConfig["paths"]; exists {
					return fmt.Sprintf("mkdir -p ./workspace && cp -r %v ./workspace/", paths)
				}
			}
			return "mkdir -p ./workspace"
		case "attach_workspace":
			return "echo 'Using local workspace if available'"
		case "store_artifacts":
			if artifactConfig, ok := value.(map[string]interface{}); ok {
				if path, exists := artifactConfig["path"]; exists {
					return fmt.Sprintf("mkdir -p ./artifacts && cp -r %s ./artifacts/", path)
				}
			}
			return "mkdir -p ./artifacts"
		case "store_test_results":
			if testConfig, ok := value.(map[string]interface{}); ok {
				if path, exists := testConfig["path"]; exists {
					return fmt.Sprintf("mkdir -p ./test-results && cp -r %s ./test-results/", path)
				}
			}
			return "mkdir -p ./test-results"
		default:
			// Custom command or orb usage
			if valueStr, ok := value.(string); ok {
				return valueStr
			}
			return fmt.Sprintf("echo 'Custom step not converted: %s'", key)
		}
	}
	return "echo 'Unknown step type'"
}

// Helper to get job dependencies from workflow
func getJobDependencies(jobName string, workflow Workflow) []string {
	var deps []string
	// TODO: Parse workflow dependencies if needed
	return deps
}

// Add helpful local development tasks
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
