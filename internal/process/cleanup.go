package process

import (
	"github.com/tanmaythakare/creo/internal/config"
	"os"
)

// CleanStale removes dead PIDs from the registry
func CleanStale(root string) {
	reg, err := LoadRegistry(root)
	if err != nil {
		return
	}

	changed := false
	for name, pid := range reg {
		proc, err := os.FindProcess(pid)
		if err != nil {
			delete(reg, name)
			changed = true
			continue
		}
		if checkProcess(proc) != nil {
			delete(reg, name)
			changed = true
		}
	}

	if changed {
		_ = SaveRegistry(root, reg)
	}
}

// CleanOrphans removes PIDs for services that are no longer in the configuration
func CleanOrphans(root string, cfg *config.Config) {
	if cfg == nil {
		return
	}
	reg, err := LoadRegistry(root)
	if err != nil {
		return
	}

	changed := false
	for name := range reg {
		if _, ok := cfg.Services[name]; !ok {
			delete(reg, name)
			changed = true
		}
	}

	if changed {
		_ = SaveRegistry(root, reg)
	}
}
