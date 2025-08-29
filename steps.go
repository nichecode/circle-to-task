package main

import (
	"fmt"
	"strings"
)

// extractCommand extracts the command string from a CircleCI step
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

// convertStepToCommand converts CircleCI steps to local equivalent commands
func convertStepToCommand(step Step) string {
	// Handle string steps (like "checkout" or command name)
	if stepStr, ok := step.(string); ok {
		switch stepStr {
		case "checkout":
			return "git checkout HEAD"
		default:
			// This could be a command invocation without parameters
			return fmt.Sprintf("task %s", stepStr)
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

// isCommandInvocation checks if a step is a command invocation
func isCommandInvocation(step Step) (string, bool) {
	stepMap, ok := step.(map[string]interface{})
	if !ok {
		return "", false
	}
	
	// Look for keys that aren't built-in CircleCI steps
	builtInSteps := map[string]bool{
		"run": true, "checkout": true, "setup_remote_docker": true,
		"save_cache": true, "restore_cache": true, "persist_to_workspace": true,
		"attach_workspace": true, "store_artifacts": true, "store_test_results": true,
		"when": true, "unless": true,
	}
	
	for key := range stepMap {
		if !builtInSteps[key] {
			return key, true
		}
	}
	
	return "", false
}

// normalizeCommand performs basic command normalization
func normalizeCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	cmd = strings.ReplaceAll(cmd, "\n", " ")
	return strings.Join(strings.Fields(cmd), " ")
}
