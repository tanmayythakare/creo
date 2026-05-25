package config

// Config is the top-level structure for creo.yml
type Config struct {
	Project   Project             `yaml:"project"`
	Services  map[string]Service  `yaml:"services"`
	Workflows map[string][]string `yaml:"workflows"`
}

// Project holds project-level metadata
type Project struct {
	Name string `yaml:"name"`
}

// Service defines a single runnable service
type Service struct {
	Path    string `yaml:"path"`
	Command string `yaml:"command"`
	Alias   string `yaml:"alias,omitempty"`
}
