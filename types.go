package main

// CircleCI structures
type CircleCIConfig struct {
	Version   string                    `yaml:"version"`
	Jobs      map[string]Job            `yaml:"jobs"`
	Commands  map[string]Command        `yaml:"commands,omitempty"`
	Workflows map[string]interface{}    `yaml:"workflows"`
	Executors map[string]interface{}    `yaml:"executors,omitempty"`
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
	Desc   string   `yaml:"desc,omitempty"`
	Cmds   []string `yaml:"cmds"`
	Deps   []string `yaml:"deps,omitempty"`
	Dir    string   `yaml:"dir,omitempty"`
	Silent bool     `yaml:"silent,omitempty"`
}
