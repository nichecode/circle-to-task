package main

import (
	"fmt"
	"strings"
)

// analyzePatterns finds common command patterns across jobs
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

// generateTaskName creates a meaningful task name from a command
func generateTaskName(cmd string) string {
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

// isCommonWord checks if a word should be excluded from task names
func isCommonWord(word string) bool {
	common := map[string]bool{
		"and": true, "or": true, "the": true, "a": true, "an": true,
		"with": true, "for": true, "to": true, "of": true, "in": true,
	}
	return common[strings.ToLower(word)]
}

// findPatternTask finds if a normalized command matches an existing pattern
func findPatternTask(normalized string, patterns map[string]Task) string {
	for taskName, task := range patterns {
		if len(task.Cmds) > 0 && normalizeCommand(task.Cmds[0]) == normalized {
			return taskName
		}
	}
	return ""
}
