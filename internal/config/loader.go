package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads creo.yml from the given directory and returns a validated Config
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, "creo.yml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("✗ creo.yml not found in %s. Fix: Run: creo init", dir)
	}

	// 1. Parse YAML into an AST Node to check for duplicate keys, invalid types, and nulls
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("✗ failed to parse creo.yml: %w. Fix: Check YAML syntax", err)
	}

	if err := checkDuplicateKeys(&node); err != nil {
		return nil, err
	}

	if err := validateWorkflowsAndNulls(&node); err != nil {
		return nil, err
	}

	// 2. Unmarshal into the final Config struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("✗ failed to parse creo.yml: %w. Fix: Check YAML syntax", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func checkDuplicateKeys(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]bool)
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if seen[keyNode.Value] {
				return fmt.Errorf("✗ duplicate key '%s' found in YAML on line %d", keyNode.Value, keyNode.Line)
			}
			seen[keyNode.Value] = true
			if err := checkDuplicateKeys(node.Content[i+1]); err != nil {
				return err
			}
		}
	} else if node.Kind == yaml.SequenceNode || node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			if err := checkDuplicateKeys(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateWorkflowsAndNulls(node *yaml.Node) error {
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) > 0 {
			return validateWorkflowsAndNulls(node.Content[0])
		}
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]

		if key == "workflows" && val.Kind == yaml.MappingNode {
			for j := 0; j < len(val.Content); j += 2 {
				wfName := val.Content[j].Value
				wfVal := val.Content[j+1]
				if wfVal.Kind == yaml.ScalarNode && wfVal.Tag == "!!str" {
					return fmt.Errorf("✗ workflow '%s' must be a list of services, not a string", wfName)
				}
				if wfVal.Kind != yaml.SequenceNode {
					return fmt.Errorf("✗ workflow '%s' must be a list of services", wfName)
				}
			}
		}

		if key == "services" && val.Kind == yaml.MappingNode {
			for j := 0; j < len(val.Content); j += 2 {
				svcName := val.Content[j].Value
				svcVal := val.Content[j+1]
				if svcVal.Kind == yaml.MappingNode {
					for k := 0; k < len(svcVal.Content); k += 2 {
						propKey := svcVal.Content[k].Value
						propVal := svcVal.Content[k+1]
						if propVal.Tag == "!!null" || propVal.Value == "" {
							if propKey == "command" {
								return fmt.Errorf("✗ service '%s' has an empty command. Fix: Provide a valid start command", svcName)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func validate(cfg *Config) error {
	if cfg.Project.Name == "" {
		return fmt.Errorf("✗ project.name is required in creo.yml. Fix: Add 'name: your-project' under 'project:'")
	}
	if len(cfg.Services) == 0 {
		return fmt.Errorf("✗ no services defined in creo.yml. Fix: Add at least one service under 'services:'")
	}

	for name, svc := range cfg.Services {
		// P1: Service name character limits
		if len([]rune(name)) > 50 {
			return fmt.Errorf("✗ service name '%s' exceeds 50 characters", name)
		}

		// P0: Path traversal on service name
		badNameChars := []string{"..", "/", "\\", "*", "?"}
		for _, bc := range badNameChars {
			if strings.Contains(name, bc) {
				return fmt.Errorf("✗ invalid character in service name")
			}
		}

		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, name)
		if !matched {
			return fmt.Errorf("invalid service name: %q", name)
		}

		// YAML reserved words check
		reserved := map[string]bool{"yes": true, "no": true, "on": true, "off": true, "true": true, "false": true, "null": true}
		if reserved[strings.ToLower(name)] {
			return fmt.Errorf("✗ service name '%s' is a YAML reserved word", name)
		}

		// YAML null values — treat null/empty path as "."
		if svc.Path == "" {
			svc.Path = "."
			cfg.Services[name] = svc
		}

		// P0: Path traversal on service path
		badPathChars := []string{"..", "*", "?"}
		for _, bc := range badPathChars {
			if strings.Contains(svc.Path, bc) {
				return fmt.Errorf("✗ invalid character in path: %s", svc.Path)
			}
		}

		if svc.Command == "" {
			return fmt.Errorf("✗ service '%s' has no command. Fix: Add 'command: your-start-cmd' to the service definition", name)
		}

		// P1: Service command character limits
		if len([]rune(svc.Command)) > 200 {
			return fmt.Errorf("✗ service '%s' command exceeds 200 characters", name)
		}

		// P0: Command injection checks
		badCmdChars := []string{"&&", "||", "|", ";", ">", "<", "`", "&", "\n", "\r"}
		for _, bc := range badCmdChars {
			if strings.Contains(svc.Command, bc) {
				return fmt.Errorf("✗ invalid character in command: %s", bc)
			}
		}
	}

	// Validate workflow service references
	for wfName, services := range cfg.Workflows {
		for _, svcName := range services {
			if _, ok := cfg.Services[svcName]; !ok {
				return fmt.Errorf("✗ workflow '%s' references unknown service '%s'. Fix: Remove '%s' from workflow or define it in 'services:'", wfName, svcName, svcName)
			}
		}
	}
	return nil
}
