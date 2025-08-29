package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	var inputFile = flag.String("input", "", "Input CircleCI config file (required)")
	var outputDir = flag.String("output", ".", "Output directory for generated files")
	var help = flag.Bool("help", false, "Show help message")
	
	flag.Parse()

	if *help || *inputFile == "" {
		showHelp()
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
	configPath := filepath.Join(*outputDir, "config.yml")
	if err := writeYAMLFile(configPath, newConfig); err != nil {
		log.Fatal("Error writing new config:", err)
	}

	// Write Taskfile
	taskfilePath := filepath.Join(*outputDir, "Taskfile.yml")
	if err := writeYAMLFile(taskfilePath, taskfile); err != nil {
		log.Fatal("Error writing taskfile:", err)
	}

	// Generate technology analysis
	if err := generateTechnologyAnalysis(config, *outputDir); err != nil {
		log.Printf("Warning: Error generating technology analysis: %v", err)
	}

	// Show success message
	showSuccess(len(config.Jobs), configPath, taskfilePath, *outputDir)
}

func showHelp() {
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
}

func showSuccess(jobCount int, configPath, taskfilePath, outputDir string) {
	fmt.Printf("✅ Successfully converted CircleCI config!\n")
	fmt.Printf("📋 Converted %d jobs into tasks\n", jobCount)
	fmt.Printf("📁 Output files:\n")
	fmt.Printf("   - %s (new CircleCI config)\n", configPath)
	fmt.Printf("   - %s (go-task configuration)\n", taskfilePath)
	fmt.Printf("   - %s/TECHNOLOGY_ANALYSIS.md (commands for AI categorization)\n", outputDir)
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   1. Review generated files\n")
	fmt.Printf("   2. Use TECHNOLOGY_ANALYSIS.md to categorize commands by technology\n")
	fmt.Printf("   3. Test locally: cd %s && task <job-name>\n", outputDir)
	fmt.Printf("   4. Install go-task if needed: go install github.com/go-task/task/v3/cmd/task@latest\n")
}

func writeYAMLFile(path string, data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}
	
	return os.WriteFile(path, yamlData, 0644)
}
